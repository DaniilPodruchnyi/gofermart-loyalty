package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/models"
	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/server/middleware"
	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/service"
	"github.com/Daniil-Podruchny/gofermart-loyalty/pkg/logger"

	"go.uber.org/zap"
)

type WithdrawalHandler struct {
	service service.WithdrawalService
}

func NewWithdrawalHandler(service service.WithdrawalService) *WithdrawalHandler {
	return &WithdrawalHandler{service: service}
}

// Withdraw обрабатывает запрос на списание баллов
func (h *WithdrawalHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(int64)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error("failed to read request body", zap.Error(err))
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	var req models.WithdrawRequest
	if err := json.Unmarshal(body, &req); err != nil {
		logger.Error("failed to unmarshal request", zap.Error(err))
		http.Error(w, "invalid request format", http.StatusBadRequest)
		return
	}

	// Валидация запроса
	if req.Order == "" {
		http.Error(w, "order number is required", http.StatusBadRequest)
		return
	}

	if req.Sum <= 0 {
		http.Error(w, "sum must be positive", http.StatusBadRequest)
		return
	}

	err = h.service.Withdraw(r.Context(), userID, req.Order, req.Sum)
	if err != nil {
		if errors.Is(err, service.ErrInvalidOrderNumber) {
			http.Error(w, "invalid order number format", http.StatusUnprocessableEntity)
			return
		}
		if errors.Is(err, service.ErrInsufficientFunds) {
			http.Error(w, "insufficient funds", http.StatusPaymentRequired)
			return
		}
		logger.Error("failed to withdraw", zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// GetWithdrawals возвращает историю списаний пользователя
func (h *WithdrawalHandler) GetWithdrawals(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(int64)

	withdrawals, err := h.service.GetWithdrawals(r.Context(), userID)
	if err != nil {
		logger.Error("failed to get withdrawals", zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if len(withdrawals) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(withdrawals); err != nil {
		logger.Error("failed to encode response", zap.Error(err))
	}
}
