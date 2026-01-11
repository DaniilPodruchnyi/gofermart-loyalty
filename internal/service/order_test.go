package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/models"

	"github.com/stretchr/testify/assert"
)

type mockOrderRepository struct {
	createFunc           func(ctx context.Context, order *models.Order) error
	getByNumberFunc      func(ctx context.Context, number string) (*models.Order, error)
	getByUserIDFunc      func(ctx context.Context, userID int64) ([]models.Order, error)
	updateStatusFunc     func(ctx context.Context, number string, status models.OrderStatus, accrual float64) error
	getPendingOrdersFunc func(ctx context.Context) ([]models.Order, error)
}

func (m *mockOrderRepository) Create(ctx context.Context, order *models.Order) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, order)
	}
	order.ID = 1
	return nil
}

func (m *mockOrderRepository) GetByNumber(ctx context.Context, number string) (*models.Order, error) {
	if m.getByNumberFunc != nil {
		return m.getByNumberFunc(ctx, number)
	}
	return nil, nil
}

func (m *mockOrderRepository) GetByUserID(ctx context.Context, userID int64) ([]models.Order, error) {
	if m.getByUserIDFunc != nil {
		return m.getByUserIDFunc(ctx, userID)
	}
	return []models.Order{}, nil
}

func (m *mockOrderRepository) UpdateStatus(ctx context.Context, number string, status models.OrderStatus, accrual float64) error {
	if m.updateStatusFunc != nil {
		return m.updateStatusFunc(ctx, number, status, accrual)
	}
	return nil
}

func (m *mockOrderRepository) GetPendingOrders(ctx context.Context) ([]models.Order, error) {
	if m.getPendingOrdersFunc != nil {
		return m.getPendingOrdersFunc(ctx)
	}
	return []models.Order{}, nil
}

type mockBalanceRepository struct {
	getByUserIDFunc       func(ctx context.Context, userID int64) (current, withdrawn float64, err error)
	addAccrualFunc        func(ctx context.Context, userID int64, amount float64) error
	initializeBalanceFunc func(ctx context.Context, userID int64) error
}

func (m *mockBalanceRepository) GetByUserID(ctx context.Context, userID int64) (current, withdrawn float64, err error) {
	if m.getByUserIDFunc != nil {
		return m.getByUserIDFunc(ctx, userID)
	}
	return 0, 0, nil
}

func (m *mockBalanceRepository) AddAccrual(ctx context.Context, userID int64, amount float64) error {
	if m.addAccrualFunc != nil {
		return m.addAccrualFunc(ctx, userID, amount)
	}
	return nil
}

func (m *mockBalanceRepository) InitializeBalance(ctx context.Context, userID int64) error {
	if m.initializeBalanceFunc != nil {
		return m.initializeBalanceFunc(ctx, userID)
	}
	return nil
}

func TestOrderService_UploadOrder(t *testing.T) {
	tests := []struct {
		name        string
		userID      int64
		orderNumber string
		orderRepo   *mockOrderRepository
		balanceRepo *mockBalanceRepository
		wantErr     error
	}{
		{
			name:        "success",
			userID:      1,
			orderNumber: "12345678903",
			orderRepo: &mockOrderRepository{
				getByNumberFunc: func(ctx context.Context, number string) (*models.Order, error) {
					return nil, nil
				},
				createFunc: func(ctx context.Context, order *models.Order) error {
					order.ID = 1
					return nil
				},
			},
			balanceRepo: &mockBalanceRepository{
				initializeBalanceFunc: func(ctx context.Context, userID int64) error {
					return nil
				},
			},
			wantErr: nil,
		},
		{
			name:        "invalid order number",
			userID:      1,
			orderNumber: "1234567890",
			orderRepo:   &mockOrderRepository{},
			balanceRepo: &mockBalanceRepository{},
			wantErr:     ErrInvalidOrderNumber,
		},
		{
			name:        "order already exists for same user",
			userID:      1,
			orderNumber: "12345678903",
			orderRepo: &mockOrderRepository{
				getByNumberFunc: func(ctx context.Context, number string) (*models.Order, error) {
					return &models.Order{
						ID:         1,
						UserID:     1,
						Number:     number,
						Status:     models.OrderStatusNew,
						UploadedAt: time.Now(),
					}, nil
				},
			},
			balanceRepo: &mockBalanceRepository{},
			wantErr:     ErrOrderExists,
		},
		{
			name:        "order exists for different user",
			userID:      1,
			orderNumber: "12345678903",
			orderRepo: &mockOrderRepository{
				getByNumberFunc: func(ctx context.Context, number string) (*models.Order, error) {
					return &models.Order{
						ID:         1,
						UserID:     2,
						Number:     number,
						Status:     models.OrderStatusNew,
						UploadedAt: time.Now(),
					}, nil
				},
			},
			balanceRepo: &mockBalanceRepository{},
			wantErr:     ErrOrderConflict,
		},
		{
			name:        "database error on GetByNumber",
			userID:      1,
			orderNumber: "12345678903",
			orderRepo: &mockOrderRepository{
				getByNumberFunc: func(ctx context.Context, number string) (*models.Order, error) {
					return nil, errors.New("database error")
				},
			},
			balanceRepo: &mockBalanceRepository{},
			wantErr:     errors.New("database error"),
		},
		{
			name:        "database error on Create",
			userID:      1,
			orderNumber: "12345678903",
			orderRepo: &mockOrderRepository{
				getByNumberFunc: func(ctx context.Context, number string) (*models.Order, error) {
					return nil, nil
				},
				createFunc: func(ctx context.Context, order *models.Order) error {
					return errors.New("database error")
				},
			},
			balanceRepo: &mockBalanceRepository{},
			wantErr:     errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewOrderService(tt.orderRepo, tt.balanceRepo)
			err := service.UploadOrder(context.Background(), tt.userID, tt.orderNumber)

			if tt.wantErr != nil {
				assert.Error(t, err)
				if !errors.Is(err, tt.wantErr) && err.Error() != tt.wantErr.Error() {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestOrderService_GetUserOrders(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		userID      int64
		orderRepo   *mockOrderRepository
		balanceRepo *mockBalanceRepository
		wantEmpty   bool
		wantLen     int
		wantErr     bool
	}{
		{
			name:   "success with orders",
			userID: 1,
			orderRepo: &mockOrderRepository{
				getByUserIDFunc: func(ctx context.Context, userID int64) ([]models.Order, error) {
					return []models.Order{
						{
							ID:         1,
							UserID:     userID,
							Number:     "12345678903",
							Status:     models.OrderStatusProcessed,
							Accrual:    500,
							UploadedAt: now,
						},
						{
							ID:         2,
							UserID:     userID,
							Number:     "4561261212345467",
							Status:     models.OrderStatusNew,
							Accrual:    0,
							UploadedAt: now,
						},
					}, nil
				},
			},
			balanceRepo: &mockBalanceRepository{},
			wantEmpty:   false,
			wantLen:     2,
			wantErr:     false,
		},
		{
			name:   "no orders",
			userID: 1,
			orderRepo: &mockOrderRepository{
				getByUserIDFunc: func(ctx context.Context, userID int64) ([]models.Order, error) {
					return []models.Order{}, nil
				},
			},
			balanceRepo: &mockBalanceRepository{},
			wantEmpty:   true,
			wantLen:     0,
			wantErr:     false,
		},
		{
			name:   "database error",
			userID: 1,
			orderRepo: &mockOrderRepository{
				getByUserIDFunc: func(ctx context.Context, userID int64) ([]models.Order, error) {
					return nil, errors.New("database error")
				},
			},
			balanceRepo: &mockBalanceRepository{},
			wantEmpty:   false,
			wantLen:     0,
			wantErr:     true,
		},
		{
			name:   "order with zero accrual should not include accrual field",
			userID: 1,
			orderRepo: &mockOrderRepository{
				getByUserIDFunc: func(ctx context.Context, userID int64) ([]models.Order, error) {
					return []models.Order{
						{
							ID:         1,
							UserID:     userID,
							Number:     "12345678903",
							Status:     models.OrderStatusNew,
							Accrual:    0,
							UploadedAt: now,
						},
					}, nil
				},
			},
			balanceRepo: &mockBalanceRepository{},
			wantEmpty:   false,
			wantLen:     1,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewOrderService(tt.orderRepo, tt.balanceRepo)
			orders, err := service.GetUserOrders(context.Background(), tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Len(t, orders, tt.wantLen)

			if tt.wantEmpty {
				assert.Empty(t, orders)
			}

			// Проверяем что accrual присутствует только когда > 0
			for _, order := range orders {
				assert.NotEmpty(t, order.Number)
				assert.NotEmpty(t, order.Status)
				assert.NotEmpty(t, order.UploadedAt)

				// Проверяем формат времени RFC3339
				_, err := time.Parse(time.RFC3339, order.UploadedAt)
				assert.NoError(t, err, "UploadedAt should be in RFC3339 format")
			}
		})
	}
}

func TestOrderService_GetBalance(t *testing.T) {
	tests := []struct {
		name          string
		userID        int64
		orderRepo     *mockOrderRepository
		balanceRepo   *mockBalanceRepository
		wantCurrent   float64
		wantWithdrawn float64
		wantErr       bool
	}{
		{
			name:      "success",
			userID:    1,
			orderRepo: &mockOrderRepository{},
			balanceRepo: &mockBalanceRepository{
				initializeBalanceFunc: func(ctx context.Context, userID int64) error {
					return nil
				},
				getByUserIDFunc: func(ctx context.Context, userID int64) (current, withdrawn float64, err error) {
					return 500.5, 42.0, nil
				},
			},
			wantCurrent:   500.5,
			wantWithdrawn: 42.0,
			wantErr:       false,
		},
		{
			name:      "zero balance",
			userID:    1,
			orderRepo: &mockOrderRepository{},
			balanceRepo: &mockBalanceRepository{
				initializeBalanceFunc: func(ctx context.Context, userID int64) error {
					return nil
				},
				getByUserIDFunc: func(ctx context.Context, userID int64) (current, withdrawn float64, err error) {
					return 0, 0, nil
				},
			},
			wantCurrent:   0,
			wantWithdrawn: 0,
			wantErr:       false,
		},
		{
			name:      "database error",
			userID:    1,
			orderRepo: &mockOrderRepository{},
			balanceRepo: &mockBalanceRepository{
				initializeBalanceFunc: func(ctx context.Context, userID int64) error {
					return nil
				},
				getByUserIDFunc: func(ctx context.Context, userID int64) (current, withdrawn float64, err error) {
					return 0, 0, errors.New("database error")
				},
			},
			wantCurrent:   0,
			wantWithdrawn: 0,
			wantErr:       true,
		},
		{
			name:      "initialize balance error is ignored",
			userID:    1,
			orderRepo: &mockOrderRepository{},
			balanceRepo: &mockBalanceRepository{
				initializeBalanceFunc: func(ctx context.Context, userID int64) error {
					return errors.New("balance already exists")
				},
				getByUserIDFunc: func(ctx context.Context, userID int64) (current, withdrawn float64, err error) {
					return 100.0, 50.0, nil
				},
			},
			wantCurrent:   100.0,
			wantWithdrawn: 50.0,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewOrderService(tt.orderRepo, tt.balanceRepo)
			balance, err := service.GetBalance(context.Background(), tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, balance)
			assert.Equal(t, tt.wantCurrent, balance.Current)
			assert.Equal(t, tt.wantWithdrawn, balance.Withdrawn)
		})
	}
}

func TestOrderService_UploadOrder_ValidNumbers(t *testing.T) {
	// Тест с различными валидными номерами заказов
	validNumbers := []string{
		"12345678903",
		"4561261212345467",
		"79927398713",
		"0",
	}

	for _, number := range validNumbers {
		t.Run("valid_"+number, func(t *testing.T) {
			orderRepo := &mockOrderRepository{
				getByNumberFunc: func(ctx context.Context, n string) (*models.Order, error) {
					return nil, nil
				},
				createFunc: func(ctx context.Context, order *models.Order) error {
					order.ID = 1
					return nil
				},
			}
			balanceRepo := &mockBalanceRepository{
				initializeBalanceFunc: func(ctx context.Context, userID int64) error {
					return nil
				},
			}

			service := NewOrderService(orderRepo, balanceRepo)
			err := service.UploadOrder(context.Background(), 1, number)

			assert.NoError(t, err)
		})
	}
}

func TestOrderService_UploadOrder_InvalidNumbers(t *testing.T) {
	// Тест с различными невалидными номерами заказов
	invalidNumbers := []string{
		"1234567890",
		"abcd1234",
		"",
		"1234 5678 903",
	}

	for _, number := range invalidNumbers {
		t.Run("invalid_"+number, func(t *testing.T) {
			orderRepo := &mockOrderRepository{}
			balanceRepo := &mockBalanceRepository{}

			service := NewOrderService(orderRepo, balanceRepo)
			err := service.UploadOrder(context.Background(), 1, number)

			assert.Error(t, err)
			assert.Equal(t, ErrInvalidOrderNumber, err)
		})
	}
}
