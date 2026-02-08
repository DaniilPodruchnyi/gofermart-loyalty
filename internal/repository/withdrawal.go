package repository

import (
	"context"

	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/models"
)

type WithdrawalRepository interface {
	Create(ctx context.Context, withdrawal *models.Withdrawal) error
	GetByUserID(ctx context.Context, userID int64) ([]models.Withdrawal, error)
}

type withdrawalRepository struct {
	*BaseRepository[*models.Withdrawal]
}

func NewWithdrawalRepository(pool PgxPool) WithdrawalRepository {
	return &withdrawalRepository{
		BaseRepository: NewBaseRepository[*models.Withdrawal](pool, "withdrawals"),
	}
}

func (r *withdrawalRepository) Create(ctx context.Context, withdrawal *models.Withdrawal) error {
	query := `
		INSERT INTO withdrawals (user_id, order_number, sum, processed_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	err := r.QueryRow(ctx, query,
		withdrawal.UserID,
		withdrawal.OrderNumber,
		withdrawal.Sum,
		withdrawal.ProcessedAt,
	).Scan(&withdrawal.ID)

	return err
}

func (r *withdrawalRepository) GetByUserID(ctx context.Context, userID int64) ([]models.Withdrawal, error) {
	query := `
		SELECT id, user_id, order_number, sum, processed_at
		FROM withdrawals
		WHERE user_id = $1
		ORDER BY processed_at DESC
	`

	rows, err := r.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var withdrawals []models.Withdrawal
	for rows.Next() {
		var w models.Withdrawal
		err := rows.Scan(
			&w.ID,
			&w.UserID,
			&w.OrderNumber,
			&w.Sum,
			&w.ProcessedAt,
		)
		if err != nil {
			return nil, err
		}
		withdrawals = append(withdrawals, w)
	}

	return withdrawals, rows.Err()
}
