package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v3"
)

func TestUserRepository_Create(t *testing.T) {
	tests := []struct {
		name    string
		user    *models.User
		mockFn  func(pgxmock.PgxPoolIface)
		wantErr bool
	}{
		{
			name: "success",
			user: &models.User{
				Login:        "testuser",
				PasswordHash: "hashedpassword",
				CreatedAt:    time.Now(),
			},
			mockFn: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id"}).AddRow(int64(1))
				mock.ExpectQuery("INSERT INTO users").
					WithArgs("testuser", "hashedpassword", pgxmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "duplicate login error",
			user: &models.User{
				Login:        "testuser",
				PasswordHash: "hashedpassword",
				CreatedAt:    time.Now(),
			},
			mockFn: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("INSERT INTO users").
					WithArgs("testuser", "hashedpassword", pgxmock.AnyArg()).
					WillReturnError(errors.New("duplicate key"))
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

			repo := &userRepository{pool: mock}
			err = repo.Create(context.Background(), tt.user)

			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.user.ID == 0 {
				t.Error("expected user ID to be set")
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestUserRepository_GetByLogin(t *testing.T) {
	tests := []struct {
		name      string
		login     string
		mockFn    func(pgxmock.PgxPoolIface)
		wantUser  bool
		wantLogin string
		wantErr   bool
	}{
		{
			name:  "user found",
			login: "testuser",
			mockFn: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "login", "password_hash", "created_at"}).
					AddRow(int64(1), "testuser", "hashedpassword", time.Now())
				mock.ExpectQuery("SELECT (.+) FROM users WHERE login").
					WithArgs("testuser").
					WillReturnRows(rows)
			},
			wantUser:  true,
			wantLogin: "testuser",
			wantErr:   false,
		},
		{
			name:  "user not found",
			login: "nonexistent",
			mockFn: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT (.+) FROM users WHERE login").
					WithArgs("nonexistent").
					WillReturnError(pgx.ErrNoRows)
			},
			wantUser: false,
			wantErr:  false,
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

			repo := &userRepository{pool: mock}
			user, err := repo.GetByLogin(context.Background(), tt.login)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetByLogin() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantUser {
				if user == nil {
					t.Error("expected user to be found")
					return
				}
				if user.Login != tt.wantLogin {
					t.Errorf("expected login %s, got %s", tt.wantLogin, user.Login)
				}
			} else if user != nil {
				t.Error("expected user to be nil")
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestUserRepository_GetByID(t *testing.T) {
	tests := []struct {
		name     string
		id       int64
		mockFn   func(pgxmock.PgxPoolIface)
		wantUser bool
		wantErr  bool
	}{
		{
			name: "user found",
			id:   1,
			mockFn: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "login", "password_hash", "created_at"}).
					AddRow(int64(1), "testuser", "hashedpassword", time.Now())
				mock.ExpectQuery("SELECT (.+) FROM users WHERE id").
					WithArgs(int64(1)).
					WillReturnRows(rows)
			},
			wantUser: true,
			wantErr:  false,
		},
		{
			name: "user not found",
			id:   999,
			mockFn: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT (.+) FROM users WHERE id").
					WithArgs(int64(999)).
					WillReturnError(pgx.ErrNoRows)
			},
			wantUser: false,
			wantErr:  false,
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

			repo := &userRepository{pool: mock}
			user, err := repo.GetByID(context.Background(), tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantUser && user == nil {
				t.Error("expected user to be found")
			} else if !tt.wantUser && user != nil {
				t.Error("expected user to be nil")
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %s", err)
			}
		})
	}
}
