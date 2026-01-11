package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/server/middleware"
	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/service"
	"github.com/Daniil-Podruchny/gofermart-loyalty/pkg/logger"

	"go.uber.org/zap"
)

type OrderHandler struct {
	service service.OrderService
}

func NewOrderHandler(service service.OrderService) *OrderHandler {
	return &OrderHandler{service: service}
}

func (h *OrderHandler) UploadOrder(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(int64)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error("failed to read request body", zap.Error(err))
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	orderNumber := strings.TrimSpace(string(body))
	if orderNumber == "" {
		http.Error(w, "order number is required", http.StatusBadRequest)
		return
	}

	err = h.service.UploadOrder(r.Context(), userID, orderNumber)
	if err != nil {
		if errors.Is(err, service.ErrInvalidOrderNumber) {
			http.Error(w, "invalid order number format", http.StatusUnprocessableEntity)
			return
		}
		if errors.Is(err, service.ErrOrderExists) {
			w.WriteHeader(http.StatusOK)
			return
		}
		if errors.Is(err, service.ErrOrderConflict) {
			http.Error(w, "order already uploaded by another user", http.StatusConflict)
			return
		}
		logger.Error("failed to upload order", zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (h *OrderHandler) GetOrders(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(int64)

	orders, err := h.service.GetUserOrders(r.Context(), userID)
	if err != nil {
		logger.Error("failed to get orders", zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(orders); err != nil {
		logger.Error("failed to encode response", zap.Error(err))
	}
}

func (h *OrderHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(int64)

	balance, err := h.service.GetBalance(r.Context(), userID)
	if err != nil {
		logger.Error("failed to get balance", zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(balance); err != nil {
		logger.Error("failed to encode response", zap.Error(err))
	}
}
