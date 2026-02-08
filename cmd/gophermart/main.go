package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/client"
	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/config"
	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/database"
	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/server"
	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/worker"
	"github.com/Daniil-Podruchny/gofermart-loyalty/pkg/logger"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

func main() {
	// Инициализация логгера
	if err := logger.Initialize(); err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	// Загрузка конфигурации
	cfg := config.Load()
	logger.Info("configuration loaded",
		zap.String("run_address", cfg.RunAddress),
		zap.String("database_uri", maskDatabaseURI(cfg.DatabaseURI)),
		zap.String("accrual_address", cfg.AccrualSystemAddress),
	)

	// Применение миграций
	if err := database.RunMigrations(cfg.DatabaseURI); err != nil {
		logger.Error("failed to run migrations", zap.Error(err))
		os.Exit(1)
	}

	logger.Info("migrations applied successfully")

	// Создание контекста с отменой
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Создание сервера с functional options
	srv, err := server.New(ctx, cfg)
	if err != nil {
		logger.Error("failed to create server", zap.Error(err))
		os.Exit(1)
	}
	defer srv.Shutdown()

	// HTTP сервер
	httpServer := &http.Server{
		Addr:         cfg.RunAddress,
		Handler:      srv.Router(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Используем errgroup для structured concurrency
	g, gCtx := errgroup.WithContext(ctx)

	// Горутина для HTTP сервера
	g.Go(func() error {
		logger.Info("starting server", zap.String("address", cfg.RunAddress))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	})

	// Горутина для accrual worker
	if cfg.AccrualSystemAddress != "" {
		accrualClient := client.NewAccrualClient(cfg.AccrualSystemAddress)
		accrualWorker := worker.NewAccrualWorker(
			accrualClient,
			srv.OrderRepo(),
			srv.BalanceRepo(),
			worker.WithConcurrency(10),
			worker.WithPollInterval(5*time.Second),
		)

		g.Go(func() error {
			accrualWorker.Start(gCtx)
			return nil
		})

		logger.Info("accrual worker started", zap.String("address", cfg.AccrualSystemAddress))
	} else {
		logger.Warn("accrual system address not provided, worker disabled")
	}

	// Ожидание сигнала остановки
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		logger.Info("shutting down server...")
	case <-gCtx.Done():
		logger.Error("context cancelled", zap.Error(gCtx.Err()))
	}

	// Останавливаем контекст для worker и сервера
	cancel()

	// Graceful shutdown HTTP сервера
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("server forced to shutdown", zap.Error(err))
	}

	// Ждем завершения всех горутин
	if err := g.Wait(); err != nil && err != http.ErrServerClosed {
		logger.Error("error from errgroup", zap.Error(err))
	}

	logger.Info("server stopped")
}

func maskDatabaseURI(uri string) string {
	if len(uri) > 20 {
		return uri[:10] + "***" + uri[len(uri)-10:]
	}
	return "***"
}
