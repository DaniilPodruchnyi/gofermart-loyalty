package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/assert"
)

func TestOrderRepository_Create(t *testing.T) {
	tests := []struct {
		name    string
		order   *models.Order
		mockFn  func(pgxmock.PgxPoolIface)
		wantErr bool
	}{
		{
			name: "success",
			order: &models.Order{
				UserID:     1,
				Number:     "12345678903",
				Status:     models.OrderStatusNew,
				Accrual:    0,
				UploadedAt: time.Now(),
			},
			mockFn: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id"}).AddRow(int64(1))
				mock.ExpectQuery("INSERT INTO orders").
					WithArgs(int64(1), "12345678903", models.OrderStatusNew, float64(0), pgxmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "database error",
			order: &models.Order{
				UserID:     1,
				Number:     "12345678903",
				Status:     models.OrderStatusNew,
				Accrual:    0,
				UploadedAt: time.Now(),
			},
			mockFn: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("INSERT INTO orders").
					WithArgs(int64(1), "12345678903", models.OrderStatusNew, float64(0), pgxmock.AnyArg()).
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

			repo := &orderRepository{pool: mock}
			err = repo.Create(context.Background(), tt.order)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, int64(1), tt.order.ID)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestOrderRepository_GetByNumber(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		number  string
		mockFn  func(pgxmock.PgxPoolIface)
		want    *models.Order
		wantErr bool
	}{
		{
			name:   "order found",
			number: "12345678903",
			mockFn: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "user_id", "number", "status", "accrual", "uploaded_at"}).
					AddRow(int64(1), int64(1), "12345678903", models.OrderStatusNew, float64(0), now)
				mock.ExpectQuery("SELECT (.+) FROM orders WHERE number").
					WithArgs("12345678903").
					WillReturnRows(rows)
			},
			want: &models.Order{
				ID:         1,
				UserID:     1,
				Number:     "12345678903",
				Status:     models.OrderStatusNew,
				Accrual:    0,
				UploadedAt: now,
			},
			wantErr: false,
		},
		{
			name:   "order not found",
			number: "99999999999",
			mockFn: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT (.+) FROM orders WHERE number").
					WithArgs("99999999999").
					WillReturnError(pgx.ErrNoRows)
			},
			want:    nil,
			wantErr: false,
		},
		{
			name:   "database error",
			number: "12345678903",
			mockFn: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT (.+) FROM orders WHERE number").
					WithArgs("12345678903").
					WillReturnError(assert.AnError)
			},
			want:    nil,
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

			repo := &orderRepository{pool: mock}
			order, err := repo.GetByNumber(context.Background(), tt.number)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, order)
			} else {
				assert.NoError(t, err)
				if tt.want != nil {
					assert.NotNil(t, order)
					assert.Equal(t, tt.want.Number, order.Number)
					assert.Equal(t, tt.want.UserID, order.UserID)
				} else {
					assert.Nil(t, order)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestOrderRepository_GetByUserID(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		userID  int64
		mockFn  func(pgxmock.PgxPoolIface)
		wantLen int
		wantErr bool
	}{
		{
			name:   "orders found",
			userID: 1,
			mockFn: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "user_id", "number", "status", "accrual", "uploaded_at"}).
					AddRow(int64(1), int64(1), "12345678903", models.OrderStatusNew, float64(0), now).
					AddRow(int64(2), int64(1), "4561261212345467", models.OrderStatusProcessed, float64(500), now)
				mock.ExpectQuery("SELECT (.+) FROM orders WHERE user_id").
					WithArgs(int64(1)).
					WillReturnRows(rows)
			},
			wantLen: 2,
			wantErr: false,
		},
		{
			name:   "no orders",
			userID: 1,
			mockFn: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "user_id", "number", "status", "accrual", "uploaded_at"})
				mock.ExpectQuery("SELECT (.+) FROM orders WHERE user_id").
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
				mock.ExpectQuery("SELECT (.+) FROM orders WHERE user_id").
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

			repo := &orderRepository{pool: mock}
			orders, err := repo.GetByUserID(context.Background(), tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, orders, tt.wantLen)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestOrderRepository_UpdateStatus(t *testing.T) {
	tests := []struct {
		name    string
		number  string
		status  models.OrderStatus
		accrual float64
		mockFn  func(pgxmock.PgxPoolIface)
		wantErr bool
	}{
		{
			name:    "success",
			number:  "12345678903",
			status:  models.OrderStatusProcessed,
			accrual: 500,
			mockFn: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("UPDATE orders").
					WithArgs(models.OrderStatusProcessed, float64(500), "12345678903").
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))
			},
			wantErr: false,
		},
		{
			name:    "database error",
			number:  "12345678903",
			status:  models.OrderStatusProcessed,
			accrual: 500,
			mockFn: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("UPDATE orders").
					WithArgs(models.OrderStatusProcessed, float64(500), "12345678903").
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

			repo := &orderRepository{pool: mock}
			err = repo.UpdateStatus(context.Background(), tt.number, tt.status, tt.accrual)

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

func TestOrderRepository_GetPendingOrders(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		mockFn  func(pgxmock.PgxPoolIface)
		wantLen int
		wantErr bool
	}{
		{
			name: "pending orders found",
			mockFn: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "user_id", "number", "status", "accrual", "uploaded_at"}).
					AddRow(int64(1), int64(1), "12345678903", models.OrderStatusNew, float64(0), now).
					AddRow(int64(2), int64(1), "4561261212345467", models.OrderStatusProcessing, float64(0), now)
				mock.ExpectQuery("SELECT (.+) FROM orders WHERE status IN").
					WithArgs(models.OrderStatusNew, models.OrderStatusProcessing).
					WillReturnRows(rows)
			},
			wantLen: 2,
			wantErr: false,
		},
		{
			name: "no pending orders",
			mockFn: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "user_id", "number", "status", "accrual", "uploaded_at"})
				mock.ExpectQuery("SELECT (.+) FROM orders WHERE status IN").
					WithArgs(models.OrderStatusNew, models.OrderStatusProcessing).
					WillReturnRows(rows)
			},
			wantLen: 0,
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

			repo := &orderRepository{pool: mock}
			orders, err := repo.GetPendingOrders(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, orders, tt.wantLen)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %s", err)
			}
		})
	}
}
