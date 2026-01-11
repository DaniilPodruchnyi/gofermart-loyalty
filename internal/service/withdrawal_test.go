package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/models"
	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/repository"

	"github.com/stretchr/testify/assert"
)

type mockWithdrawalRepository struct {
	createFunc      func(ctx context.Context, withdrawal *models.Withdrawal) error
	getByUserIDFunc func(ctx context.Context, userID int64) ([]models.Withdrawal, error)
}

func (m *mockWithdrawalRepository) Create(ctx context.Context, withdrawal *models.Withdrawal) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, withdrawal)
	}
	withdrawal.ID = 1
	return nil
}

func (m *mockWithdrawalRepository) GetByUserID(ctx context.Context, userID int64) ([]models.Withdrawal, error) {
	if m.getByUserIDFunc != nil {
		return m.getByUserIDFunc(ctx, userID)
	}
	return []models.Withdrawal{}, nil
}

func TestWithdrawalService_Withdraw(t *testing.T) {
	tests := []struct {
		name           string
		userID         int64
		orderNumber    string
		sum            float64
		withdrawalRepo *mockWithdrawalRepository
		balanceRepo    *mockBalanceRepository
		wantErr        error
	}{
		{
			name:        "success",
			userID:      1,
			orderNumber: "12345678903",
			sum:         100.0,
			withdrawalRepo: &mockWithdrawalRepository{
				createFunc: func(ctx context.Context, withdrawal *models.Withdrawal) error {
					withdrawal.ID = 1
					return nil
				},
			},
			balanceRepo: &mockBalanceRepository{
				withdrawFunc: func(ctx context.Context, userID int64, amount float64) error {
					return nil
				},
			},
			wantErr: nil,
		},
		{
			name:           "invalid order number",
			userID:         1,
			orderNumber:    "1234567890",
			sum:            100.0,
			withdrawalRepo: &mockWithdrawalRepository{},
			balanceRepo:    &mockBalanceRepository{},
			wantErr:        ErrInvalidOrderNumber,
		},
		{
			name:           "invalid sum - zero",
			userID:         1,
			orderNumber:    "12345678903",
			sum:            0,
			withdrawalRepo: &mockWithdrawalRepository{},
			balanceRepo:    &mockBalanceRepository{},
			wantErr:        errors.New("invalid withdrawal amount"),
		},
		{
			name:           "invalid sum - negative",
			userID:         1,
			orderNumber:    "12345678903",
			sum:            -100.0,
			withdrawalRepo: &mockWithdrawalRepository{},
			balanceRepo:    &mockBalanceRepository{},
			wantErr:        errors.New("invalid withdrawal amount"),
		},
		{
			name:        "insufficient funds",
			userID:      1,
			orderNumber: "12345678903",
			sum:         1000.0,
			withdrawalRepo: &mockWithdrawalRepository{
				createFunc: func(ctx context.Context, withdrawal *models.Withdrawal) error {
					return nil
				},
			},
			balanceRepo: &mockBalanceRepository{
				withdrawFunc: func(ctx context.Context, userID int64, amount float64) error {
					return repository.ErrInsufficientFunds
				},
			},
			wantErr: ErrInsufficientFunds,
		},
		{
			name:        "database error on withdraw",
			userID:      1,
			orderNumber: "12345678903",
			sum:         100.0,
			withdrawalRepo: &mockWithdrawalRepository{
				createFunc: func(ctx context.Context, withdrawal *models.Withdrawal) error {
					return nil
				},
			},
			balanceRepo: &mockBalanceRepository{
				withdrawFunc: func(ctx context.Context, userID int64, amount float64) error {
					return errors.New("database error")
				},
			},
			wantErr: errors.New("database error"),
		},
		{
			name:        "database error on create withdrawal",
			userID:      1,
			orderNumber: "12345678903",
			sum:         100.0,
			withdrawalRepo: &mockWithdrawalRepository{
				createFunc: func(ctx context.Context, withdrawal *models.Withdrawal) error {
					return errors.New("database error")
				},
			},
			balanceRepo: &mockBalanceRepository{
				withdrawFunc: func(ctx context.Context, userID int64, amount float64) error {
					return nil
				},
				addAccrualFunc: func(ctx context.Context, userID int64, amount float64) error {
					return nil
				},
			},
			wantErr: errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewWithdrawalService(tt.withdrawalRepo, tt.balanceRepo)
			err := service.Withdraw(context.Background(), tt.userID, tt.orderNumber, tt.sum)

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

func TestWithdrawalService_GetWithdrawals(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name           string
		userID         int64
		withdrawalRepo *mockWithdrawalRepository
		balanceRepo    *mockBalanceRepository
		wantLen        int
		wantErr        bool
	}{
		{
			name:   "success with withdrawals",
			userID: 1,
			withdrawalRepo: &mockWithdrawalRepository{
				getByUserIDFunc: func(ctx context.Context, userID int64) ([]models.Withdrawal, error) {
					return []models.Withdrawal{
						{
							ID:          1,
							UserID:      userID,
							OrderNumber: "12345678903",
							Sum:         500.0,
							ProcessedAt: now,
						},
						{
							ID:          2,
							UserID:      userID,
							OrderNumber: "4561261212345467",
							Sum:         250.0,
							ProcessedAt: now,
						},
					}, nil
				},
			},
			balanceRepo: &mockBalanceRepository{},
			wantLen:     2,
			wantErr:     false,
		},
		{
			name:   "no withdrawals",
			userID: 1,
			withdrawalRepo: &mockWithdrawalRepository{
				getByUserIDFunc: func(ctx context.Context, userID int64) ([]models.Withdrawal, error) {
					return []models.Withdrawal{}, nil
				},
			},
			balanceRepo: &mockBalanceRepository{},
			wantLen:     0,
			wantErr:     false,
		},
		{
			name:   "database error",
			userID: 1,
			withdrawalRepo: &mockWithdrawalRepository{
				getByUserIDFunc: func(ctx context.Context, userID int64) ([]models.Withdrawal, error) {
					return nil, errors.New("database error")
				},
			},
			balanceRepo: &mockBalanceRepository{},
			wantLen:     0,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewWithdrawalService(tt.withdrawalRepo, tt.balanceRepo)
			withdrawals, err := service.GetWithdrawals(context.Background(), tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Len(t, withdrawals, tt.wantLen)

			// Проверяем формат данных
			for _, w := range withdrawals {
				assert.NotEmpty(t, w.Order)
				assert.Greater(t, w.Sum, 0.0)
				assert.NotEmpty(t, w.ProcessedAt)

				// Проверяем формат времени RFC3339
				_, err := time.Parse(time.RFC3339, w.ProcessedAt)
				assert.NoError(t, err, "ProcessedAt should be in RFC3339 format")
			}
		})
	}
}

func TestWithdrawalService_Withdraw_ValidOrderNumbers(t *testing.T) {
	validNumbers := []string{
		"12345678903",
		"4561261212345467",
		"79927398713",
	}

	for _, number := range validNumbers {
		t.Run("valid_"+number, func(t *testing.T) {
			withdrawalRepo := &mockWithdrawalRepository{
				createFunc: func(ctx context.Context, withdrawal *models.Withdrawal) error {
					withdrawal.ID = 1
					return nil
				},
			}
			balanceRepo := &mockBalanceRepository{
				withdrawFunc: func(ctx context.Context, userID int64, amount float64) error {
					return nil
				},
			}

			service := NewWithdrawalService(withdrawalRepo, balanceRepo)
			err := service.Withdraw(context.Background(), 1, number, 100.0)

			assert.NoError(t, err)
		})
	}
}

func TestWithdrawalService_Withdraw_InvalidOrderNumbers(t *testing.T) {
	invalidNumbers := []string{
		"1234567890",
		"abcd1234",
		"",
		"1234 5678 903",
	}

	for _, number := range invalidNumbers {
		t.Run("invalid_"+number, func(t *testing.T) {
			withdrawalRepo := &mockWithdrawalRepository{}
			balanceRepo := &mockBalanceRepository{}

			service := NewWithdrawalService(withdrawalRepo, balanceRepo)
			err := service.Withdraw(context.Background(), 1, number, 100.0)

			assert.Error(t, err)
			assert.Equal(t, ErrInvalidOrderNumber, err)
		})
	}
}
