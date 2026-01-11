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
	config      *config.Config
	router      *chi.Mux
	pool        *pgxpool.Pool
	userHandler *handlers.UserHandler
}

func New(ctx context.Context, cfg *config.Config) (*Server, error) {
	pool, err := database.NewPool(ctx, cfg.DatabaseURI)
	if err != nil {
		return nil, err
	}

	logger.Log.Info("database connection established")

	userRepo := repository.NewUserRepository(pool)
	userService := service.NewUserService(userRepo, cfg.JWTSecret)
	userHandler := handlers.NewUserHandler(userService)

	s := &Server{
		config:      cfg,
		router:      chi.NewRouter(),
		pool:        pool,
		userHandler: userHandler,
	}

	s.setupRoutes()

	return s, nil
}

func (s *Server) Router() *chi.Mux {
	return s.router
}

func (s *Server) Shutdown() {
	logger.Log.Info("shutting down server")
	s.pool.Close()
}
