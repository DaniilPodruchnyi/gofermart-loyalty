package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type BalanceRepository interface {
	GetByUserID(ctx context.Context, userID int64) (current, withdrawn float64, err error)
	AddAccrual(ctx context.Context, userID int64, amount float64) error
	InitializeBalance(ctx context.Context, userID int64) error
}

type balanceRepository struct {
	pool PgxPool // Используем интерфейс из user.go
}

func NewBalanceRepository(pool *pgxpool.Pool) BalanceRepository {
	return &balanceRepository{pool: pool}
}

func (r *balanceRepository) GetByUserID(ctx context.Context, userID int64) (current, withdrawn float64, err error) {
	query := `
		SELECT current, withdrawn
		FROM balance
		WHERE user_id = $1
	`

	err = r.pool.QueryRow(ctx, query, userID).Scan(&current, &withdrawn)
	return current, withdrawn, err
}

func (r *balanceRepository) AddAccrual(ctx context.Context, userID int64, amount float64) error {
	query := `
		INSERT INTO balance (user_id, current, withdrawn)
		VALUES ($1, $2, 0)
		ON CONFLICT (user_id)
		DO UPDATE SET current = balance.current + $2
	`

	_, err := r.pool.Exec(ctx, query, userID, amount)
	return err
}

func (r *balanceRepository) InitializeBalance(ctx context.Context, userID int64) error {
	query := `
		INSERT INTO balance (user_id, current, withdrawn)
		VALUES ($1, 0, 0)
		ON CONFLICT (user_id) DO NOTHING
	`

	_, err := r.pool.Exec(ctx, query, userID)
	return err
}
