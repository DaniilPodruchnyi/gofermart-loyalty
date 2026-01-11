package repository

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/assert"
)

func TestBalanceRepository_GetByUserID(t *testing.T) {
	tests := []struct {
		name          string
		userID        int64
		mockFn        func(pgxmock.PgxPoolIface)
		wantCurrent   float64
		wantWithdrawn float64
		wantErr       bool
	}{
		{
			name:   "balance found",
			userID: 1,
			mockFn: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"current", "withdrawn"}).
					AddRow(float64(500.5), float64(42.0))
				mock.ExpectQuery("SELECT current, withdrawn FROM balance WHERE user_id").
					WithArgs(int64(1)).
					WillReturnRows(rows)
			},
			wantCurrent:   500.5,
			wantWithdrawn: 42.0,
			wantErr:       false,
		},
		{
			name:   "balance not found",
			userID: 999,
			mockFn: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT current, withdrawn FROM balance WHERE user_id").
					WithArgs(int64(999)).
					WillReturnError(pgx.ErrNoRows)
			},
			wantCurrent:   0,
			wantWithdrawn: 0,
			wantErr:       true,
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

			repo := &balanceRepository{pool: mock}
			current, withdrawn, err := repo.GetByUserID(context.Background(), tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantCurrent, current)
				assert.Equal(t, tt.wantWithdrawn, withdrawn)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestBalanceRepository_AddAccrual(t *testing.T) {
	tests := []struct {
		name    string
		userID  int64
		amount  float64
		mockFn  func(pgxmock.PgxPoolIface)
		wantErr bool
	}{
		{
			name:   "success",
			userID: 1,
			amount: 100.0,
			mockFn: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("INSERT INTO balance").
					WithArgs(int64(1), float64(100.0)).
					WillReturnResult(pgxmock.NewResult("INSERT", 1))
			},
			wantErr: false,
		},
		{
			name:   "database error",
			userID: 1,
			amount: 100.0,
			mockFn: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("INSERT INTO balance").
					WithArgs(int64(1), float64(100.0)).
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

			repo := &balanceRepository{pool: mock}
			err = repo.AddAccrual(context.Background(), tt.userID, tt.amount)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestBalanceRepository_InitializeBalance(t *testing.T) {
	tests := []struct {
		name    string
		userID  int64
		mockFn  func(pgxmock.PgxPoolIface)
		wantErr bool
	}{
		{
			name:   "success",
			userID: 1,
			mockFn: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("INSERT INTO balance").
					WithArgs(int64(1)).
					WillReturnResult(pgxmock.NewResult("INSERT", 1))
			},
			wantErr: false,
		},
		{
			name:   "already exists (ignored)",
			userID: 1,
			mockFn: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("INSERT INTO balance").
					WithArgs(int64(1)).
					WillReturnResult(pgxmock.NewResult("INSERT", 0))
			},
			wantErr: false,
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

			repo := &balanceRepository{pool: mock}
			err = repo.InitializeBalance(context.Background(), tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %s", err)
			}
		})
	}
}
