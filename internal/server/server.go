package server

import (
	"context"

	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/config"
	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/database"
	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/handlers"
	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/repository"
	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/service"
	"github.com/Daniil-Podruchny/gofermart-loyalty/pkg/logger"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Server struct {
	config       *config.Config
	router       *chi.Mux
	pool         *pgxpool.Pool
	userHandler  *handlers.UserHandler
	orderHandler *handlers.OrderHandler
}

func New(ctx context.Context, cfg *config.Config) (*Server, error) {
	pool, err := database.NewPool(ctx, cfg.DatabaseURI)
	if err != nil {
		return nil, err
	}

	logger.Info("database connection established")

	// Инициализация репозиториев
	userRepo := repository.NewUserRepository(pool)
	orderRepo := repository.NewOrderRepository(pool)
	balanceRepo := repository.NewBalanceRepository(pool)

	// Инициализация сервисов
	userService := service.NewUserService(userRepo, cfg.JWTSecret)
	orderService := service.NewOrderService(orderRepo, balanceRepo)

	// Инициализация хендлеров
	userHandler := handlers.NewUserHandler(userService)
	orderHandler := handlers.NewOrderHandler(orderService)

	s := &Server{
		config:       cfg,
		router:       chi.NewRouter(),
		pool:         pool,
		userHandler:  userHandler,
		orderHandler: orderHandler,
	}

	s.setupRoutes()

	return s, nil
}

func (s *Server) setupRoutes() {
	routerInstance := NewRouter(s.userHandler, s.orderHandler, s.config.JWTSecret)
	s.router = routerInstance.Routes().(*chi.Mux)
}

func (s *Server) Router() *chi.Mux {
	return s.router
}

func (s *Server) Shutdown() {
	logger.Info("shutting down server")
	s.pool.Close()
}
