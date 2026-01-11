package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/models"
	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/service"
)

type mockUserService struct {
	registerFunc func(ctx context.Context, req models.RegisterRequest) (string, error)
	loginFunc    func(ctx context.Context, req models.LoginRequest) (string, error)
}

func (m *mockUserService) Register(ctx context.Context, req models.RegisterRequest) (string, error) {
	if m.registerFunc != nil {
		return m.registerFunc(ctx, req)
	}
	return "", nil
}

func (m *mockUserService) Login(ctx context.Context, req models.LoginRequest) (string, error) {
	if m.loginFunc != nil {
		return m.loginFunc(ctx, req)
	}
	return "", nil
}

func TestUserHandler_Register(t *testing.T) {
	tests := []struct {
		name           string
		body           interface{}
		service        *mockUserService
		expectedStatus int
		checkHeader    bool
	}{
		{
			name: "success",
			body: models.RegisterRequest{
				Login:    "testuser",
				Password: "password123",
			},
			service: &mockUserService{
				registerFunc: func(ctx context.Context, req models.RegisterRequest) (string, error) {
					return "test-token", nil
				},
			},
			expectedStatus: http.StatusOK,
			checkHeader:    true,
		},
		{
			name: "login exists",
			body: models.RegisterRequest{
				Login:    "existinguser",
				Password: "password123",
			},
			service: &mockUserService{
				registerFunc: func(ctx context.Context, req models.RegisterRequest) (string, error) {
					return "", service.ErrLoginExists
				},
			},
			expectedStatus: http.StatusConflict,
			checkHeader:    false,
		},
		{
			name:           "invalid json",
			body:           "invalid json",
			service:        &mockUserService{},
			expectedStatus: http.StatusBadRequest,
			checkHeader:    false,
		},
		{
			name: "empty login",
			body: models.RegisterRequest{
				Login:    "",
				Password: "password123",
			},
			service:        &mockUserService{},
			expectedStatus: http.StatusBadRequest,
			checkHeader:    false,
		},
		{
			name: "empty password",
			body: models.RegisterRequest{
				Login:    "testuser",
				Password: "",
			},
			service:        &mockUserService{},
			expectedStatus: http.StatusBadRequest,
			checkHeader:    false,
		},
		{
			name: "internal server error",
			body: models.RegisterRequest{
				Login:    "testuser",
				Password: "password123",
			},
			service: &mockUserService{
				registerFunc: func(ctx context.Context, req models.RegisterRequest) (string, error) {
					return "", errors.New("internal error")
				},
			},
			expectedStatus: http.StatusInternalServerError,
			checkHeader:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewUserHandler(tt.service)

			var body []byte
			if str, ok := tt.body.(string); ok {
				body = []byte(str)
			} else {
				body, _ = json.Marshal(tt.body)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.Register(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkHeader {
				authHeader := w.Header().Get("Authorization")
				if authHeader == "" {
					t.Error("expected Authorization header to be set")
				} else if authHeader != "Bearer test-token" {
					t.Errorf("expected 'Bearer test-token', got '%s'", authHeader)
				}
			}
		})
	}
}

func TestUserHandler_Login(t *testing.T) {
	tests := []struct {
		name           string
		body           interface{}
		service        *mockUserService
		expectedStatus int
		checkHeader    bool
	}{
		{
			name: "success",
			body: models.LoginRequest{
				Login:    "testuser",
				Password: "password123",
			},
			service: &mockUserService{
				loginFunc: func(ctx context.Context, req models.LoginRequest) (string, error) {
					return "test-token", nil
				},
			},
			expectedStatus: http.StatusOK,
			checkHeader:    true,
		},
		{
			name: "invalid credentials",
			body: models.LoginRequest{
				Login:    "testuser",
				Password: "wrongpassword",
			},
			service: &mockUserService{
				loginFunc: func(ctx context.Context, req models.LoginRequest) (string, error) {
					return "", service.ErrInvalidCredentials
				},
			},
			expectedStatus: http.StatusUnauthorized,
			checkHeader:    false,
		},
		{
			name:           "invalid json",
			body:           "invalid json",
			service:        &mockUserService{},
			expectedStatus: http.StatusBadRequest,
			checkHeader:    false,
		},
		{
			name: "empty login",
			body: models.LoginRequest{
				Login:    "",
				Password: "password123",
			},
			service:        &mockUserService{},
			expectedStatus: http.StatusBadRequest,
			checkHeader:    false,
		},
		{
			name: "empty password",
			body: models.LoginRequest{
				Login:    "testuser",
				Password: "",
			},
			service:        &mockUserService{},
			expectedStatus: http.StatusBadRequest,
			checkHeader:    false,
		},
		{
			name: "internal server error",
			body: models.LoginRequest{
				Login:    "testuser",
				Password: "password123",
			},
			service: &mockUserService{
				loginFunc: func(ctx context.Context, req models.LoginRequest) (string, error) {
					return "", errors.New("internal error")
				},
			},
			expectedStatus: http.StatusInternalServerError,
			checkHeader:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewUserHandler(tt.service)

			var body []byte
			if str, ok := tt.body.(string); ok {
				body = []byte(str)
			} else {
				body, _ = json.Marshal(tt.body)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/user/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.Login(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkHeader {
				authHeader := w.Header().Get("Authorization")
				if authHeader == "" {
					t.Error("expected Authorization header to be set")
				} else if authHeader != "Bearer test-token" {
					t.Errorf("expected 'Bearer test-token', got '%s'", authHeader)
				}
			}
		})
	}
}
