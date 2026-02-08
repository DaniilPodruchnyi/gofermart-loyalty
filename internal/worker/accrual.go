package worker

import (
	"context"
	"time"

	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/client"
	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/models"
	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/repository"
	"github.com/Daniil-Podruchny/gofermart-loyalty/pkg/logger"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type AccrualWorker struct {
	accrualClient *client.AccrualClient
	orderRepo     repository.OrderRepository
	balanceRepo   repository.BalanceRepository
	pollInterval  time.Duration
	concurrency   int // лимит параллельных обработок
	stopped       chan struct{}
}

type WorkerOption func(*AccrualWorker)

// WithPollInterval устанавливает интервал опроса
func WithPollInterval(d time.Duration) WorkerOption {
	return func(w *AccrualWorker) {
		w.pollInterval = d
	}
}

// WithConcurrency устанавливает лимит параллельных обработок
func WithConcurrency(n int) WorkerOption {
	return func(w *AccrualWorker) {
		w.concurrency = n
	}
}

func NewAccrualWorker(
	accrualClient *client.AccrualClient,
	orderRepo repository.OrderRepository,
	balanceRepo repository.BalanceRepository,
	opts ...WorkerOption,
) *AccrualWorker {
	w := &AccrualWorker{
		accrualClient: accrualClient,
		orderRepo:     orderRepo,
		balanceRepo:   balanceRepo,
		pollInterval:  5 * time.Second,
		concurrency:   10, // дефолт: 10 параллельных запросов
		stopped:       make(chan struct{}),
	}

	for _, opt := range opts {
		opt(w)
	}

	return w
}

func (w *AccrualWorker) Start(ctx context.Context) {
	logger.Info("accrual worker started",
		zap.Duration("poll_interval", w.pollInterval),
		zap.Int("concurrency", w.concurrency))

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("accrual worker stopping")
			close(w.stopped)
			return
		case <-ticker.C:
			w.processOrders(ctx)
		}
	}
}

func (w *AccrualWorker) Stop() {
	<-w.stopped
}

func (w *AccrualWorker) processOrders(ctx context.Context) {
	orders, err := w.orderRepo.GetPendingOrders(ctx)
	if err != nil {
		logger.Error("failed to get pending orders", zap.Error(err))
		return
	}

	if len(orders) == 0 {
		return
	}

	logger.Info("processing pending orders", zap.Int("count", len(orders)))

	// Используем errgroup для параллельной обработки с лимитом
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(w.concurrency)

	for i := range orders {
		order := orders[i] // capture loop variable

		g.Go(func() error {
			w.processOrder(ctx, &order)
			return nil // не прерываем обработку других заказов при ошибке
		})
	}

	// Ждем завершения всех горутин
	if err := g.Wait(); err != nil {
		logger.Error("error in parallel processing", zap.Error(err))
	}
}

func (w *AccrualWorker) processOrder(ctx context.Context, order *models.Order) {
	accrualResp, retryAfter, err := w.accrualClient.GetOrderAccrual(ctx, order.Number)

	// Rate limit - ждем
	if err != nil && retryAfter > 0 {
		logger.Warn("rate limit exceeded",
			zap.String("order", order.Number),
			zap.Int("retry_after", retryAfter))
		time.Sleep(time.Duration(retryAfter) * time.Second)
		return
	}

	if err != nil {
		logger.Error("failed to get accrual",
			zap.String("order", order.Number),
			zap.Error(err))
		return
	}

	// Заказ не найден
	if accrualResp == nil {
		return
	}

	// Обновляем статус
	newStatus := accrualResp.Status
	var accrual float64
	if accrualResp.Accrual != nil {
		accrual = *accrualResp.Accrual
	}

	// Только финальные статусы
	if newStatus == models.OrderStatusProcessed || newStatus == models.OrderStatusInvalid {
		err = w.orderRepo.UpdateStatus(ctx, order.Number, newStatus, accrual)
		if err != nil {
			logger.Error("failed to update order status",
				zap.String("order", order.Number),
				zap.Error(err))
			return
		}

		// Начисляем баллы
		if newStatus == models.OrderStatusProcessed && accrual > 0 {
			err = w.balanceRepo.AddAccrual(ctx, order.UserID, accrual)
			if err != nil {
				logger.Error("failed to add accrual",
					zap.String("order", order.Number),
					zap.Float64("accrual", accrual),
					zap.Error(err))
				return
			}

			logger.Info("accrual added successfully",
				zap.String("order", order.Number),
				zap.Int64("user_id", order.UserID),
				zap.Float64("accrual", accrual))
		}
	}
}
