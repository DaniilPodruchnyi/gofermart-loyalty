package service

import (
	"context"
	"errors"
	"time"

	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/models"
	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrLoginExists        = errors.New("login already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type UserService interface {
	Register(ctx context.Context, req models.RegisterRequest) (string, error)
	Login(ctx context.Context, req models.LoginRequest) (string, error)
}

type userService struct {
	repo      repository.UserRepository
	jwtSecret string
}

func NewUserService(repo repository.UserRepository, jwtSecret string) UserService {
	return &userService{
		repo:      repo,
		jwtSecret: jwtSecret,
	}
}

func (s *userService) Register(ctx context.Context, req models.RegisterRequest) (string, error) {
	existing, err := s.repo.GetByLogin(ctx, req.Login)
	if err != nil {
		return "", err
	}
	if existing != nil {
		return "", ErrLoginExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	user := &models.User{
		Login:        req.Login,
		PasswordHash: string(hashedPassword),
		CreatedAt:    time.Now(),
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return "", err
	}

	return s.generateToken(user.ID, user.Login)
}

func (s *userService) Login(ctx context.Context, req models.LoginRequest) (string, error) {
	user, err := s.repo.GetByLogin(ctx, req.Login)
	if err != nil {
		return "", err
	}
	if user == nil {
		return "", ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return "", ErrInvalidCredentials
	}

	return s.generateToken(user.ID, user.Login)
}

func (s *userService) generateToken(userID int64, login string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"login":   login,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}
