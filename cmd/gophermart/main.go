package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/config"
	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/database"
	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/server"
	"github.com/Daniil-Podruchny/gofermart-loyalty/pkg/logger"

	"go.uber.org/zap"
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
		zap.String("database_uri", cfg.DatabaseURI),
	)

	// Применение миграций
	if err := database.RunMigrations(cfg.DatabaseURI); err != nil {
		logger.Error("failed to run migrations", zap.Error(err))
		os.Exit(1)
	}

	logger.Info("migrations applied successfully")

	// Создание сервера
	srv, err := server.New(context.Background(), cfg)
	if err != nil {
		logger.Error("failed to create server", zap.Error(err))
		os.Exit(1)
	}
	defer srv.Shutdown()

	// Создание HTTP сервера
	httpServer := &http.Server{
		Addr:         cfg.RunAddress,
		Handler:      srv.Router(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Запуск сервера в горутине
	go func() {
		logger.Info("starting server", zap.String("address", cfg.RunAddress))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", zap.Error(err))
			os.Exit(1)
		}
	}()

	// Ожидание сигнала остановки
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", zap.Error(err))
	}

	logger.Info("server stopped")
}
