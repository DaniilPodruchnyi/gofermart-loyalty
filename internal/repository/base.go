package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// Entity базовый интерфейс для всех сущностей
type Entity interface {
	GetID() int64
	SetID(id int64)
}

// BaseRepository generic репозиторий для общих операций
type BaseRepository[T Entity] struct {
	pool      PgxPool
	tableName string
}

// NewBaseRepository создает новый базовый репозиторий
func NewBaseRepository[T Entity](pool PgxPool, tableName string) *BaseRepository[T] {
	return &BaseRepository[T]{
		pool:      pool,
		tableName: tableName,
	}
}

// ExecContext выполняет запрос без возврата данных
func (r *BaseRepository[T]) ExecContext(ctx context.Context, query string, args ...interface{}) error {
	_, err := r.pool.Exec(ctx, query, args...)
	return err
}

// QueryRow выполняет запрос с возвратом одной строки
func (r *BaseRepository[T]) QueryRow(ctx context.Context, query string, args ...interface{}) pgx.Row {
	return r.pool.QueryRow(ctx, query, args...)
}

// Query выполняет запрос с возвратом множества строк
func (r *BaseRepository[T]) Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
	return r.pool.Query(ctx, query, args...)
}

// TableName возвращает имя таблицы
func (r *BaseRepository[T]) TableName() string {
	return r.tableName
}
