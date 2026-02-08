package service

import (
	"context"
	"errors"
	"time"

	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/models"
	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/repository"
	"github.com/Daniil-Podruchny/gofermart-loyalty/pkg/luhn"
)

var (
	ErrInvalidOrderNumber = errors.New("invalid order number")
	ErrOrderExists        = errors.New("order already exists")
	ErrOrderConflict      = errors.New("order uploaded by another user")
)

type OrderService interface {
	UploadOrder(ctx context.Context, userID int64, orderNumber string) error
	GetUserOrders(ctx context.Context, userID int64) ([]models.OrderResponse, error)
	GetBalance(ctx context.Context, userID int64) (*models.Balance, error)
}

type orderService struct {
	orderRepo   repository.OrderRepository
	balanceRepo repository.BalanceRepository
}

func NewOrderService(orderRepo repository.OrderRepository, balanceRepo repository.BalanceRepository) OrderService {
	return &orderService{
		orderRepo:   orderRepo,
		balanceRepo: balanceRepo,
	}
}

func (s *orderService) UploadOrder(ctx context.Context, userID int64, orderNumber string) error {
	// Проверка по алгоритму Луна
	if !luhn.Validate(orderNumber) {
		return ErrInvalidOrderNumber
	}

	// Проверить, существует ли заказ
	existingOrder, err := s.orderRepo.GetByNumber(ctx, orderNumber)
	if err != nil {
		return err
	}

	// Если заказ уже существует
	if existingOrder != nil {
		if existingOrder.UserID == userID {
			// Заказ уже загружен этим пользователем
			return ErrOrderExists
		}
		// Заказ загружен другим пользователем
		return ErrOrderConflict
	}

	// Создать новый заказ
	order := &models.Order{
		UserID:     userID,
		Number:     orderNumber,
		Status:     models.OrderStatusNew,
		Accrual:    0,
		UploadedAt: time.Now(),
	}

	err = s.orderRepo.Create(ctx, order)
	if err != nil {
		return err
	}

	// Инициализировать баланс пользователя, если его нет
	_ = s.balanceRepo.InitializeBalance(ctx, userID)

	return nil
}

func (s *orderService) GetUserOrders(ctx context.Context, userID int64) ([]models.OrderResponse, error) {
	orders, err := s.orderRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if len(orders) == 0 {
		return []models.OrderResponse{}, nil
	}

	responses := make([]models.OrderResponse, 0, len(orders))
	for _, order := range orders {
		response := models.OrderResponse{
			Number:     order.Number,
			Status:     order.Status,
			UploadedAt: order.UploadedAt.Format(time.RFC3339),
		}

		// Добавить accrual только если он больше 0
		if order.Accrual > 0 {
			response.Accrual = &order.Accrual
		}

		responses = append(responses, response)
	}

	return responses, nil
}

func (s *orderService) GetBalance(ctx context.Context, userID int64) (*models.Balance, error) {
	// Инициализировать баланс, если его нет
	_ = s.balanceRepo.InitializeBalance(ctx, userID)

	current, withdrawn, err := s.balanceRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &models.Balance{
		Current:   current,
		Withdrawn: withdrawn,
	}, nil
}
