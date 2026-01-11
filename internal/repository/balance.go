package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInsufficientFunds = errors.New("insufficient funds")
)

// BalanceRepository определяет методы для работы с балансом
type BalanceRepository interface {
	GetByUserID(ctx context.Context, userID int64) (current, withdrawn float64, err error)
	AddAccrual(ctx context.Context, userID int64, amount float64) error
	InitializeBalance(ctx context.Context, userID int64) error
	Withdraw(ctx context.Context, userID int64, amount float64) error
}

type balanceRepository struct {
	pool PgxPool
}

// NewBalanceRepository создает новый репозиторий для баланса
func NewBalanceRepository(pool *pgxpool.Pool) BalanceRepository {
	return &balanceRepository{pool: pool}
}

// GetByUserID возвращает текущий баланс и сумму списаний пользователя
func (r *balanceRepository) GetByUserID(ctx context.Context, userID int64) (current, withdrawn float64, err error) {
	query := `SELECT current, withdrawn FROM balance WHERE user_id = $1`

	err = r.pool.QueryRow(ctx, query, userID).Scan(&current, &withdrawn)
	if err != nil {
		return 0, 0, err
	}

	return current, withdrawn, nil
}

// AddAccrual начисляет баллы на счет пользователя
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

// InitializeBalance инициализирует баланс пользователя с нулевыми значениями
func (r *balanceRepository) InitializeBalance(ctx context.Context, userID int64) error {
	query := `
		INSERT INTO balance (user_id, current, withdrawn)
		VALUES ($1, 0, 0)
		ON CONFLICT (user_id) DO NOTHING
	`

	_, err := r.pool.Exec(ctx, query, userID)
	return err
}

// Withdraw списывает баллы со счета пользователя
func (r *balanceRepository) Withdraw(ctx context.Context, userID int64, amount float64) error {
	// Начинаем транзакцию
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Получаем текущий баланс с блокировкой строки
	var current float64
	query := `SELECT current FROM balance WHERE user_id = $1 FOR UPDATE`
	err = tx.QueryRow(ctx, query, userID).Scan(&current)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrInsufficientFunds
		}
		return err
	}

	// Проверяем достаточность средств
	if current < amount {
		return ErrInsufficientFunds
	}

	// Списываем баллы
	updateQuery := `
		UPDATE balance 
		SET current = current - $1, withdrawn = withdrawn + $1
		WHERE user_id = $2
	`
	_, err = tx.Exec(ctx, updateQuery, amount, userID)
	if err != nil {
		return err
	}

	// Коммитим транзакцию
	return tx.Commit(ctx)
}
