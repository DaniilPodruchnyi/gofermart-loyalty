package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/models"

	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/assert"
)

func TestWithdrawalRepository_Create(t *testing.T) {
	tests := []struct {
		name       string
		withdrawal *models.Withdrawal
		mockFn     func(pgxmock.PgxPoolIface)
		wantErr    bool
	}{
		{
			name: "success",
			withdrawal: &models.Withdrawal{
				UserID:      1,
				OrderNumber: "12345678903",
				Sum:         500.0,
				ProcessedAt: time.Now(),
			},
			mockFn: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id"}).AddRow(int64(1))
				mock.ExpectQuery("INSERT INTO withdrawals").
					WithArgs(int64(1), "12345678903", float64(500.0), pgxmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "database error",
			withdrawal: &models.Withdrawal{
				UserID:      1,
				OrderNumber: "12345678903",
				Sum:         500.0,
				ProcessedAt: time.Now(),
			},
			mockFn: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("INSERT INTO withdrawals").
					WithArgs(int64(1), "12345678903", float64(500.0), pgxmock.AnyArg()).
					WillReturnError(assert.AnError)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatal(err)
			}
			defer mock.Close()

			tt.mockFn(mock)

			repo := &withdrawalRepository{
				BaseRepository: NewBaseRepository[*models.Withdrawal](mock, "withdrawals"),
			}
			err = repo.Create(context.Background(), tt.withdrawal)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, int64(1), tt.withdrawal.ID)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestWithdrawalRepository_GetByUserID(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		userID  int64
		mockFn  func(pgxmock.PgxPoolIface)
		wantLen int
		wantErr bool
	}{
		{
			name:   "withdrawals found",
			userID: 1,
			mockFn: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "user_id", "order_number", "sum", "processed_at"}).
					AddRow(int64(1), int64(1), "12345678903", float64(500.0), now).
					AddRow(int64(2), int64(1), "4561261212345467", float64(250.0), now)
				mock.ExpectQuery("SELECT (.+) FROM withdrawals WHERE user_id").
					WithArgs(int64(1)).
					WillReturnRows(rows)
			},
			wantLen: 2,
			wantErr: false,
		},
		{
			name:   "no withdrawals",
			userID: 1,
			mockFn: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "user_id", "order_number", "sum", "processed_at"})
				mock.ExpectQuery("SELECT (.+) FROM withdrawals WHERE user_id").
					WithArgs(int64(1)).
					WillReturnRows(rows)
			},
			wantLen: 0,
			wantErr: false,
		},
		{
			name:   "database error",
			userID: 1,
			mockFn: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT (.+) FROM withdrawals WHERE user_id").
					WithArgs(int64(1)).
					WillReturnError(assert.AnError)
			},
			wantLen: 0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatal(err)
			}
			defer mock.Close()

			tt.mockFn(mock)

			repo := &withdrawalRepository{
				BaseRepository: NewBaseRepository[*models.Withdrawal](mock, "withdrawals"),
			}
			withdrawals, err := repo.GetByUserID(context.Background(), tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, withdrawals, tt.wantLen)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %s", err)
			}
		})
	}
}
