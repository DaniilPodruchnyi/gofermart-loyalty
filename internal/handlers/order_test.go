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

type mockOrderService struct {
	uploadOrderFunc   func(ctx context.Context, userID int64, orderNumber string) error
	getUserOrdersFunc func(ctx context.Context, userID int64) ([]models.OrderResponse, error)
	getBalanceFunc    func(ctx context.Context, userID int64) (*models.Balance, error)
}

func (m *mockOrderService) UploadOrder(ctx context.Context, userID int64, orderNumber string) error {
	if m.uploadOrderFunc != nil {
		return m.uploadOrderFunc(ctx, userID, orderNumber)
	}
	return nil
}

func (m *mockOrderService) GetUserOrders(ctx context.Context, userID int64) ([]models.OrderResponse, error) {
	if m.getUserOrdersFunc != nil {
		return m.getUserOrdersFunc(ctx, userID)
	}
	return []models.OrderResponse{}, nil
}

func (m *mockOrderService) GetBalance(ctx context.Context, userID int64) (*models.Balance, error) {
	if m.getBalanceFunc != nil {
		return m.getBalanceFunc(ctx, userID)
	}
	return &models.Balance{}, nil
}

func TestOrderHandler_UploadOrder(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		userID         int64
		mockService    *mockOrderService
		expectedStatus int
		expectedBody   string
	}{
		{
			name:   "success - new order",
			body:   "12345678903",
			userID: 1,
			mockService: &mockOrderService{
				uploadOrderFunc: func(ctx context.Context, userID int64, orderNumber string) error {
					return nil
				},
			},
			expectedStatus: http.StatusAccepted,
			expectedBody:   "",
		},
		{
			name:   "success - order already uploaded by user",
			body:   "12345678903",
			userID: 1,
			mockService: &mockOrderService{
				uploadOrderFunc: func(ctx context.Context, userID int64, orderNumber string) error {
					return service.ErrOrderExists
				},
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name:   "invalid order number format",
			body:   "1234567890",
			userID: 1,
			mockService: &mockOrderService{
				uploadOrderFunc: func(ctx context.Context, userID int64, orderNumber string) error {
					return service.ErrInvalidOrderNumber
				},
			},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody:   "invalid order number format",
		},
		{
			name:   "order uploaded by another user",
			body:   "12345678903",
			userID: 1,
			mockService: &mockOrderService{
				uploadOrderFunc: func(ctx context.Context, userID int64, orderNumber string) error {
					return service.ErrOrderConflict
				},
			},
			expectedStatus: http.StatusConflict,
			expectedBody:   "order already uploaded by another user",
		},
		{
			name:           "empty order number",
			body:           "",
			userID:         1,
			mockService:    &mockOrderService{},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "order number is required",
		},
		{
			name:   "internal server error",
			body:   "12345678903",
			userID: 1,
			mockService: &mockOrderService{
				uploadOrderFunc: func(ctx context.Context, userID int64, orderNumber string) error {
					return errors.New("database error")
				},
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewOrderHandler(tt.mockService)

			req := httptest.NewRequest(http.MethodPost, "/api/user/orders", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "text/plain")

			// Добавляем userID в контекст
			ctx := context.WithValue(req.Context(), middleware.UserIDKey, tt.userID)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()

			handler.UploadOrder(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedBody != "" {
				body, _ := io.ReadAll(rr.Body)
				assert.Contains(t, string(body), tt.expectedBody)
			}
		})
	}
}

func TestOrderHandler_GetOrders(t *testing.T) {
	now := time.Now()
	accrual := 500.0

	tests := []struct {
		name           string
		userID         int64
		mockService    *mockOrderService
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:   "success with orders",
			userID: 1,
			mockService: &mockOrderService{
				getUserOrdersFunc: func(ctx context.Context, userID int64) ([]models.OrderResponse, error) {
					return []models.OrderResponse{
						{
							Number:     "12345678903",
							Status:     models.OrderStatusProcessed,
							Accrual:    &accrual,
							UploadedAt: now.Format(time.RFC3339),
						},
					}, nil
				},
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
				body, _ := io.ReadAll(rr.Body)
				assert.Contains(t, string(body), "12345678903")
				assert.Contains(t, string(body), "PROCESSED")
			},
		},
		{
			name:   "no orders",
			userID: 1,
			mockService: &mockOrderService{
				getUserOrdersFunc: func(ctx context.Context, userID int64) ([]models.OrderResponse, error) {
					return []models.OrderResponse{}, nil
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
			mockService: &mockOrderService{
				getUserOrdersFunc: func(ctx context.Context, userID int64) ([]models.OrderResponse, error) {
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
			handler := NewOrderHandler(tt.mockService)

			req := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)

			// Добавляем userID в контекст
			ctx := context.WithValue(req.Context(), middleware.UserIDKey, tt.userID)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()

			handler.GetOrders(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			tt.checkResponse(t, rr)
		})
	}
}

func TestOrderHandler_GetBalance(t *testing.T) {
	tests := []struct {
		name           string
		userID         int64
		mockService    *mockOrderService
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:   "success",
			userID: 1,
			mockService: &mockOrderService{
				getBalanceFunc: func(ctx context.Context, userID int64) (*models.Balance, error) {
					return &models.Balance{
						Current:   500.5,
						Withdrawn: 42.0,
					}, nil
				},
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
				body, _ := io.ReadAll(rr.Body)
				assert.Contains(t, string(body), "500.5")
				assert.Contains(t, string(body), "42")
			},
		},
		{
			name:   "internal server error",
			userID: 1,
			mockService: &mockOrderService{
				getBalanceFunc: func(ctx context.Context, userID int64) (*models.Balance, error) {
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
			handler := NewOrderHandler(tt.mockService)

			req := httptest.NewRequest(http.MethodGet, "/api/user/balance", nil)

			// Добавляем userID в контекст
			ctx := context.WithValue(req.Context(), middleware.UserIDKey, tt.userID)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()

			handler.GetBalance(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			tt.checkResponse(t, rr)
		})
	}
}
