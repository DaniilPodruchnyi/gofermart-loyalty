package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/config"
	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/server"
	"github.com/Daniil-Podruchny/gofermart-loyalty/pkg/logger"

	"go.uber.org/zap"
)

func main() {
	if err := logger.Initialize(); err != nil {
		panic(err)
	}
	defer logger.Sync()

	cfg, err := config.Load()
	if err != nil {
		logger.Log.Fatal("failed to load config", zap.Error(err))
	}

	ctx := context.Background()
	srv, err := server.New(ctx, cfg)
	if err != nil {
		logger.Log.Fatal("failed to create server", zap.Error(err))
	}
	defer srv.Shutdown()

	httpServer := &http.Server{
		Addr:    cfg.RunAddress,
		Handler: srv.Router(),
	}

	go func() {
		logger.Log.Info("starting server", zap.String("address", cfg.RunAddress))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("server failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("shutting down gracefully")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Log.Error("server shutdown error", zap.Error(err))
	}
}
