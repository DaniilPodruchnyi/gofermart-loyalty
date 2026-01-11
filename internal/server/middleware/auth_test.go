package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestAuth(t *testing.T) {
	jwtSecret := "test-secret"

	validToken := func() string {
		claims := jwt.MapClaims{
			"user_id": float64(1),
			"login":   "testuser",
			"exp":     time.Now().Add(time.Hour).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, _ := token.SignedString([]byte(jwtSecret))
		return tokenString
	}

	expiredToken := func() string {
		claims := jwt.MapClaims{
			"user_id": float64(1),
			"login":   "testuser",
			"exp":     time.Now().Add(-time.Hour).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, _ := token.SignedString([]byte(jwtSecret))
		return tokenString
	}

	invalidSignatureToken := func() string {
		claims := jwt.MapClaims{
			"user_id": float64(1),
			"login":   "testuser",
			"exp":     time.Now().Add(time.Hour).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, _ := token.SignedString([]byte("wrong-secret"))
		return tokenString
	}

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		checkUserID    bool
	}{
		{
			name:           "valid token",
			authHeader:     "Bearer " + validToken(),
			expectedStatus: http.StatusOK,
			checkUserID:    true,
		},
		{
			name:           "no auth header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			checkUserID:    false,
		},
		{
			name:           "invalid format - no Bearer",
			authHeader:     "InvalidFormat",
			expectedStatus: http.StatusUnauthorized,
			checkUserID:    false,
		},
		{
			name:           "token without Bearer prefix",
			authHeader:     validToken(),
			expectedStatus: http.StatusUnauthorized,
			checkUserID:    false,
		},
		{
			name:           "expired token",
			authHeader:     "Bearer " + expiredToken(),
			expectedStatus: http.StatusUnauthorized,
			checkUserID:    false,
		},
		{
			name:           "invalid token",
			authHeader:     "Bearer invalid.token.here",
			expectedStatus: http.StatusUnauthorized,
			checkUserID:    false,
		},
		{
			name:           "invalid signature",
			authHeader:     "Bearer " + invalidSignatureToken(),
			expectedStatus: http.StatusUnauthorized,
			checkUserID:    false,
		},
		{
			name:           "empty Bearer token",
			authHeader:     "Bearer ",
			expectedStatus: http.StatusUnauthorized,
			checkUserID:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var userIDChecked bool
			handler := Auth(jwtSecret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.checkUserID {
					userID := r.Context().Value(UserIDKey)
					if userID == nil {
						t.Error("expected userID in context")
					} else if id, ok := userID.(int64); !ok {
						t.Error("expected userID to be int64")
					} else if id != 1 {
						t.Errorf("expected userID to be 1, got %d", id)
					}
					userIDChecked = true
				}
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkUserID && !userIDChecked {
				t.Error("userID was not checked in handler")
			}
		})
	}
}
