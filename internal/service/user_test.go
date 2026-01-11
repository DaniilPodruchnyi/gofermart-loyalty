package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/models"

	"golang.org/x/crypto/bcrypt"
)

type mockUserRepository struct {
	createFunc     func(ctx context.Context, user *models.User) error
	getByLoginFunc func(ctx context.Context, login string) (*models.User, error)
	getByIDFunc    func(ctx context.Context, id int64) (*models.User, error)
}

func (m *mockUserRepository) Create(ctx context.Context, user *models.User) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, user)
	}
	user.ID = 1
	return nil
}

func (m *mockUserRepository) GetByLogin(ctx context.Context, login string) (*models.User, error) {
	if m.getByLoginFunc != nil {
		return m.getByLoginFunc(ctx, login)
	}
	return nil, nil
}

func (m *mockUserRepository) GetByID(ctx context.Context, id int64) (*models.User, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, nil
}

func TestUserService_Register(t *testing.T) {
	tests := []struct {
		name       string
		req        models.RegisterRequest
		repo       *mockUserRepository
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "success",
			req: models.RegisterRequest{
				Login:    "newuser",
				Password: "password123",
			},
			repo: &mockUserRepository{
				getByLoginFunc: func(ctx context.Context, login string) (*models.User, error) {
					return nil, nil
				},
				createFunc: func(ctx context.Context, user *models.User) error {
					user.ID = 1
					return nil
				},
			},
			wantErr: false,
		},
		{
			name: "login already exists",
			req: models.RegisterRequest{
				Login:    "existinguser",
				Password: "password123",
			},
			repo: &mockUserRepository{
				getByLoginFunc: func(ctx context.Context, login string) (*models.User, error) {
					return &models.User{
						ID:           1,
						Login:        "existinguser",
						PasswordHash: "hash",
						CreatedAt:    time.Now(),
					}, nil
				},
			},
			wantErr:    true,
			wantErrMsg: "login already exists",
		},
		{
			name: "database error on getByLogin",
			req: models.RegisterRequest{
				Login:    "newuser",
				Password: "password123",
			},
			repo: &mockUserRepository{
				getByLoginFunc: func(ctx context.Context, login string) (*models.User, error) {
					return nil, errors.New("database error")
				},
			},
			wantErr:    true,
			wantErrMsg: "database error",
		},
		{
			name: "database error on create",
			req: models.RegisterRequest{
				Login:    "newuser",
				Password: "password123",
			},
			repo: &mockUserRepository{
				getByLoginFunc: func(ctx context.Context, login string) (*models.User, error) {
					return nil, nil
				},
				createFunc: func(ctx context.Context, user *models.User) error {
					return errors.New("create error")
				},
			},
			wantErr:    true,
			wantErrMsg: "create error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewUserService(tt.repo, "test-secret")
			token, err := service.Register(context.Background(), tt.req)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
					return
				}
				if tt.wantErrMsg != "" && err.Error() != tt.wantErrMsg {
					if !errors.Is(err, ErrLoginExists) || tt.wantErrMsg != "login already exists" {
						t.Errorf("expected error message '%s', got '%s'", tt.wantErrMsg, err.Error())
					}
				}
				if token != "" {
					t.Errorf("expected empty token on error")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if token == "" {
					t.Error("expected non-empty token")
				}
			}
		})
	}
}

func TestUserService_Login(t *testing.T) {
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)

	tests := []struct {
		name       string
		req        models.LoginRequest
		repo       *mockUserRepository
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "success",
			req: models.LoginRequest{
				Login:    "testuser",
				Password: "password123",
			},
			repo: &mockUserRepository{
				getByLoginFunc: func(ctx context.Context, login string) (*models.User, error) {
					return &models.User{
						ID:           1,
						Login:        "testuser",
						PasswordHash: string(hashedPassword),
						CreatedAt:    time.Now(),
					}, nil
				},
			},
			wantErr: false,
		},
		{
			name: "user not found",
			req: models.LoginRequest{
				Login:    "nonexistent",
				Password: "password123",
			},
			repo: &mockUserRepository{
				getByLoginFunc: func(ctx context.Context, login string) (*models.User, error) {
					return nil, nil
				},
			},
			wantErr:    true,
			wantErrMsg: "invalid credentials",
		},
		{
			name: "wrong password",
			req: models.LoginRequest{
				Login:    "testuser",
				Password: "wrongpassword",
			},
			repo: &mockUserRepository{
				getByLoginFunc: func(ctx context.Context, login string) (*models.User, error) {
					return &models.User{
						ID:           1,
						Login:        "testuser",
						PasswordHash: string(hashedPassword),
						CreatedAt:    time.Now(),
					}, nil
				},
			},
			wantErr:    true,
			wantErrMsg: "invalid credentials",
		},
		{
			name: "database error",
			req: models.LoginRequest{
				Login:    "testuser",
				Password: "password123",
			},
			repo: &mockUserRepository{
				getByLoginFunc: func(ctx context.Context, login string) (*models.User, error) {
					return nil, errors.New("database error")
				},
			},
			wantErr:    true,
			wantErrMsg: "database error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewUserService(tt.repo, "test-secret")
			token, err := service.Login(context.Background(), tt.req)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
					return
				}
				if tt.wantErrMsg != "" && err.Error() != tt.wantErrMsg {
					if !errors.Is(err, ErrInvalidCredentials) || tt.wantErrMsg != "invalid credentials" {
						t.Errorf("expected error message '%s', got '%s'", tt.wantErrMsg, err.Error())
					}
				}
				if token != "" {
					t.Errorf("expected empty token on error")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if token == "" {
					t.Error("expected non-empty token")
				}
			}
		})
	}
}
