package repository

import (
	"context"
	"errors"

	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/models"
	"github.com/jackc/pgx/v5"
)

type OrderRepository interface {
	Create(ctx context.Context, order *models.Order) error
	GetByNumber(ctx context.Context, number string) (*models.Order, error)
	GetByUserID(ctx context.Context, userID int64) ([]models.Order, error)
	UpdateStatus(ctx context.Context, number string, status models.OrderStatus, accrual float64) error
	GetPendingOrders(ctx context.Context) ([]models.Order, error)
}

type orderRepository struct {
	*BaseRepository[*models.Order]
}

func NewOrderRepository(pool PgxPool) OrderRepository {
	return &orderRepository{
		BaseRepository: NewBaseRepository[*models.Order](pool, "orders"),
	}
}

func (r *orderRepository) Create(ctx context.Context, order *models.Order) error {
	query := `
		INSERT INTO orders (user_id, number, status, accrual, uploaded_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	err := r.QueryRow(ctx, query,
		order.UserID,
		order.Number,
		order.Status,
		order.Accrual,
		order.UploadedAt,
	).Scan(&order.ID)

	return err
}

func (r *orderRepository) GetByNumber(ctx context.Context, number string) (*models.Order, error) {
	query := `
		SELECT id, user_id, number, status, accrual, uploaded_at
		FROM orders
		WHERE number = $1
	`

	order := &models.Order{}
	err := r.QueryRow(ctx, query, number).Scan(
		&order.ID,
		&order.UserID,
		&order.Number,
		&order.Status,
		&order.Accrual,
		&order.UploadedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return order, nil
}

func (r *orderRepository) GetByUserID(ctx context.Context, userID int64) ([]models.Order, error) {
	query := `
		SELECT id, user_id, number, status, accrual, uploaded_at
		FROM orders
		WHERE user_id = $1
		ORDER BY uploaded_at DESC
	`

	rows, err := r.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var order models.Order
		err := rows.Scan(
			&order.ID,
			&order.UserID,
			&order.Number,
			&order.Status,
			&order.Accrual,
			&order.UploadedAt,
		)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}

	return orders, rows.Err()
}

func (r *orderRepository) UpdateStatus(ctx context.Context, number string, status models.OrderStatus, accrual float64) error {
	query := `
		UPDATE orders
		SET status = $1, accrual = $2
		WHERE number = $3
	`

	return r.ExecContext(ctx, query, status, accrual, number)
}

func (r *orderRepository) GetPendingOrders(ctx context.Context) ([]models.Order, error) {
	query := `
		SELECT id, user_id, number, status, accrual, uploaded_at
		FROM orders
		WHERE status IN ($1, $2)
		ORDER BY uploaded_at ASC
	`

	rows, err := r.Query(ctx, query, models.OrderStatusNew, models.OrderStatusProcessing)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var order models.Order
		err := rows.Scan(
			&order.ID,
			&order.UserID,
			&order.Number,
			&order.Status,
			&order.Accrual,
			&order.UploadedAt,
		)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}

	return orders, rows.Err()
}
