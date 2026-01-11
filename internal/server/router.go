package server

import (
	"net/http"

	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/handlers"
	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/server/middleware"

	"github.com/go-chi/chi/v5"
)

type Router struct {
	userHandler       *handlers.UserHandler
	orderHandler      *handlers.OrderHandler
	withdrawalHandler *handlers.WithdrawalHandler
	jwtSecret         string
}

func NewRouter(
	userHandler *handlers.UserHandler,
	orderHandler *handlers.OrderHandler,
	withdrawalHandler *handlers.WithdrawalHandler,
	jwtSecret string,
) *Router {
	return &Router{
		userHandler:       userHandler,
		orderHandler:      orderHandler,
		withdrawalHandler: withdrawalHandler,
		jwtSecret:         jwtSecret,
	}
}

func (rt *Router) Routes() http.Handler {
	r := chi.NewRouter()

	r.Route("/api/user", func(r chi.Router) {
		// Публичные эндпоинты
		r.Post("/register", rt.userHandler.Register)
		r.Post("/login", rt.userHandler.Login)

		// Защищенные эндпоинты
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(rt.jwtSecret))

			// Orders
			r.Post("/orders", rt.orderHandler.UploadOrder)
			r.Get("/orders", rt.orderHandler.GetOrders)

			// Balance
			r.Get("/balance", rt.orderHandler.GetBalance)
			r.Post("/balance/withdraw", rt.withdrawalHandler.Withdraw)

			// Withdrawals
			r.Get("/withdrawals", rt.withdrawalHandler.GetWithdrawals)
		})
	})

	return r
}
