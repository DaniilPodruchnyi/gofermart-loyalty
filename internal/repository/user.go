package repository

import (
	"context"
	"errors"

	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByLogin(ctx context.Context, login string) (*models.User, error)
	GetByID(ctx context.Context, id int64) (*models.User, error)
}

type userRepository struct {
	*BaseRepository[*models.User]
}

func NewUserRepository(pool PgxPool) UserRepository {
	return &userRepository{
		BaseRepository: NewBaseRepository[*models.User](pool, "users"),
	}
}

func (r *userRepository) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (login, password_hash, created_at)
		VALUES ($1, $2, $3)
		RETURNING id
	`

	err := r.QueryRow(ctx, query, user.Login, user.PasswordHash, user.CreatedAt).Scan(&user.ID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrLoginExists
		}
		return err
	}

	return nil
}

func (r *userRepository) GetByLogin(ctx context.Context, login string) (*models.User, error) {
	query := `
		SELECT id, login, password_hash, created_at
		FROM users
		WHERE login = $1
	`

	user := &models.User{}
	err := r.QueryRow(ctx, query, login).Scan(
		&user.ID,
		&user.Login,
		&user.PasswordHash,
		&user.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return user, nil
}

func (r *userRepository) GetByID(ctx context.Context, id int64) (*models.User, error) {
	query := `
		SELECT id, login, password_hash, created_at
		FROM users
		WHERE id = $1
	`

	user := &models.User{}
	err := r.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Login,
		&user.PasswordHash,
		&user.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return user, nil
}
