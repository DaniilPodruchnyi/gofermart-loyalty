package handlers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/models"
	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/server/middleware"
	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/service"

	"github.com/stretchr/testify/assert"
)

type mockWithdrawalService struct {
	withdrawFunc       func(ctx context.Context, userID int64, orderNumber string, sum float64) error
	getWithdrawalsFunc func(ctx context.Context, userID int64) ([]models.WithdrawalResponse, error)
}

func (m *mockWithdrawalService) Withdraw(ctx context.Context, userID int64, orderNumber string, sum float64) error {
	if m.withdrawFunc != nil {
		return m.withdrawFunc(ctx, userID, orderNumber, sum)
	}
	return nil
}

func (m *mockWithdrawalService) GetWithdrawals(ctx context.Context, userID int64) ([]models.WithdrawalResponse, error) {
	if m.getWithdrawalsFunc != nil {
		return m.getWithdrawalsFunc(ctx, userID)
	}
	return []models.WithdrawalResponse{}, nil
}

func TestWithdrawalHandler_Withdraw(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		userID         int64
		mockService    *mockWithdrawalService
		expectedStatus int
		expectedBody   string
	}{
		{
			name:   "success",
			body:   `{"order":"12345678903","sum":100.5}`,
			userID: 1,
			mockService: &mockWithdrawalService{
				withdrawFunc: func(ctx context.Context, userID int64, orderNumber string, sum float64) error {
					return nil
				},
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name:           "invalid json",
			body:           `{invalid}`,
			userID:         1,
			mockService:    &mockWithdrawalService{},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "invalid request format",
		},
		{
			name:           "empty order number",
			body:           `{"order":"","sum":100}`,
			userID:         1,
			mockService:    &mockWithdrawalService{},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "order number is required",
		},
		{
			name:           "zero sum",
			body:           `{"order":"12345678903","sum":0}`,
			userID:         1,
			mockService:    &mockWithdrawalService{},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "sum must be positive",
		},
		{
			name:           "negative sum",
			body:           `{"order":"12345678903","sum":-100}`,
			userID:         1,
			mockService:    &mockWithdrawalService{},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "sum must be positive",
		},
		{
			name:   "invalid order number format",
			body:   `{"order":"1234567890","sum":100}`,
			userID: 1,
			mockService: &mockWithdrawalService{
				withdrawFunc: func(ctx context.Context, userID int64, orderNumber string, sum float64) error {
					return service.ErrInvalidOrderNumber
				},
			},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody:   "invalid order number format",
		},
		{
			name:   "insufficient funds",
			body:   `{"order":"12345678903","sum":1000}`,
			userID: 1,
			mockService: &mockWithdrawalService{
				withdrawFunc: func(ctx context.Context, userID int64, orderNumber string, sum float64) error {
					return service.ErrInsufficientFunds
				},
			},
			expectedStatus: http.StatusPaymentRequired,
			expectedBody:   "insufficient funds",
		},
		{
			name:   "internal server error",
			body:   `{"order":"12345678903","sum":100}`,
			userID: 1,
			mockService: &mockWithdrawalService{
				withdrawFunc: func(ctx context.Context, userID int64, orderNumber string, sum float64) error {
					return errors.New("database error")
				},
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewWithdrawalHandler(tt.mockService)

			req := httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			ctx := context.WithValue(req.Context(), middleware.UserIDKey, tt.userID)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()

			handler.Withdraw(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedBody != "" {
				body, _ := io.ReadAll(rr.Body)
				assert.Contains(t, string(body), tt.expectedBody)
			}
		})
	}
}

func TestWithdrawalHandler_GetWithdrawals(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name           string
		userID         int64
		mockService    *mockWithdrawalService
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:   "success with withdrawals",
			userID: 1,
			mockService: &mockWithdrawalService{
				getWithdrawalsFunc: func(ctx context.Context, userID int64) ([]models.WithdrawalResponse, error) {
					return []models.WithdrawalResponse{
						{
							Order:       "12345678903",
							Sum:         500.0,
							ProcessedAt: now.Format(time.RFC3339),
						},
						{
							Order:       "4561261212345467",
							Sum:         250.0,
							ProcessedAt: now.Format(time.RFC3339),
						},
					}, nil
				},
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
				body, _ := io.ReadAll(rr.Body)
				assert.Contains(t, string(body), "12345678903")
				assert.Contains(t, string(body), "500")
			},
		},
		{
			name:   "no withdrawals",
			userID: 1,
			mockService: &mockWithdrawalService{
				getWithdrawalsFunc: func(ctx context.Context, userID int64) ([]models.WithdrawalResponse, error) {
					return []models.WithdrawalResponse{}, nil
				},
			},
			expectedStatus: http.StatusNoContent,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				// Nothing to check
			},
		},
		{
			name:   "internal server error",
			userID: 1,
			mockService: &mockWithdrawalService{
				getWithdrawalsFunc: func(ctx context.Context, userID int64) ([]models.WithdrawalResponse, error) {
					return nil, errors.New("database error")
				},
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				body, _ := io.ReadAll(rr.Body)
				assert.Contains(t, string(body), "internal server error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewWithdrawalHandler(tt.mockService)

			req := httptest.NewRequest(http.MethodGet, "/api/user/withdrawals", nil)

			ctx := context.WithValue(req.Context(), middleware.UserIDKey, tt.userID)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()

			handler.GetWithdrawals(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			tt.checkResponse(t, rr)
		})
	}
}
