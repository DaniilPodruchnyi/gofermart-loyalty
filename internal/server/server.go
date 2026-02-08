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
	config            *config.Config
	router            *chi.Mux
	pool              *pgxpool.Pool
	userHandler       *handlers.UserHandler
	orderHandler      *handlers.OrderHandler
	withdrawalHandler *handlers.WithdrawalHandler
	orderRepo         repository.OrderRepository
	balanceRepo       repository.BalanceRepository
}

type ServerOption func(*Server) error

// WithPool устанавливает connection pool
func WithPool(pool *pgxpool.Pool) ServerOption {
	return func(s *Server) error {
		s.pool = pool
		return nil
	}
}

// WithConfig устанавливает конфигурацию
func WithConfig(cfg *config.Config) ServerOption {
	return func(s *Server) error {
		s.config = cfg
		return nil
	}
}

func New(ctx context.Context, cfg *config.Config, opts ...ServerOption) (*Server, error) {
	// Создаем pool если не передан
	pool, err := database.NewPool(ctx, cfg.DatabaseURI)
	if err != nil {
		return nil, err
	}

	logger.Info("database connection established")

	s := &Server{
		config: cfg,
		pool:   pool,
		router: chi.NewRouter(),
	}

	// Применяем опции
	for _, opt := range opts {
		if err := opt(s); err != nil {
			return nil, err
		}
	}

	// Инициализация репозиториев
	userRepo := repository.NewUserRepository(s.pool)
	s.orderRepo = repository.NewOrderRepository(s.pool)
	s.balanceRepo = repository.NewBalanceRepository(s.pool)
	withdrawalRepo := repository.NewWithdrawalRepository(s.pool)

	// Инициализация сервисов
	userService := service.NewUserService(userRepo, cfg.JWTSecret)
	orderService := service.NewOrderService(s.orderRepo, s.balanceRepo)
	withdrawalService := service.NewWithdrawalService(withdrawalRepo, s.balanceRepo)

	// Инициализация хендлеров
	s.userHandler = handlers.NewUserHandler(userService)
	s.orderHandler = handlers.NewOrderHandler(orderService)
	s.withdrawalHandler = handlers.NewWithdrawalHandler(withdrawalService)

	s.setupRoutes()

	return s, nil
}

func (s *Server) setupRoutes() {
	routerInstance := NewRouter(s.userHandler, s.orderHandler, s.withdrawalHandler, s.config.JWTSecret)
	s.router = routerInstance.Routes().(*chi.Mux)
}

func (s *Server) Router() *chi.Mux {
	return s.router
}

func (s *Server) OrderRepo() repository.OrderRepository {
	return s.orderRepo
}

func (s *Server) BalanceRepo() repository.BalanceRepository {
	return s.balanceRepo
}

func (s *Server) Shutdown() {
	logger.Info("shutting down server")
	s.pool.Close()
}
