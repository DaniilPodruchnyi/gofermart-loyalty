package service

import (
	"context"
	"errors"
	"time"

	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/models"
	"github.com/Daniil-Podruchny/gofermart-loyalty/internal/repository"
	"github.com/Daniil-Podruchny/gofermart-loyalty/pkg/luhn"
)

var (
	ErrInsufficientFunds = errors.New("insufficient funds")
)

// WithdrawalService определяет методы для работы со списаниями
type WithdrawalService interface {
	Withdraw(ctx context.Context, userID int64, orderNumber string, sum float64) error
	GetWithdrawals(ctx context.Context, userID int64) ([]models.WithdrawalResponse, error)
}

type withdrawalService struct {
	withdrawalRepo repository.WithdrawalRepository
	balanceRepo    repository.BalanceRepository
}

// NewWithdrawalService создает новый сервис для списаний
func NewWithdrawalService(withdrawalRepo repository.WithdrawalRepository, balanceRepo repository.BalanceRepository) WithdrawalService {
	return &withdrawalService{
		withdrawalRepo: withdrawalRepo,
		balanceRepo:    balanceRepo,
	}
}

// Withdraw списывает баллы со счета пользователя
func (s *withdrawalService) Withdraw(ctx context.Context, userID int64, orderNumber string, sum float64) error {
	// Проверка номера заказа по алгоритму Луна
	if !luhn.Validate(orderNumber) {
		return ErrInvalidOrderNumber
	}

	// Проверка суммы
	if sum <= 0 {
		return errors.New("invalid withdrawal amount")
	}

	// Списываем баллы (с проверкой достаточности средств)
	err := s.balanceRepo.Withdraw(ctx, userID, sum)
	if err != nil {
		if errors.Is(err, repository.ErrInsufficientFunds) {
			return ErrInsufficientFunds
		}
		return err
	}

	// Создаем запись о списании
	withdrawal := &models.Withdrawal{
		UserID:      userID,
		OrderNumber: orderNumber,
		Sum:         sum,
		ProcessedAt: time.Now(),
	}

	err = s.withdrawalRepo.Create(ctx, withdrawal)
	if err != nil {
		// Если не удалось создать запись, откатываем списание
		// В реальном приложении лучше использовать распределенные транзакции
		_ = s.balanceRepo.AddAccrual(ctx, userID, sum)
		return err
	}

	return nil
}

// GetWithdrawals возвращает историю списаний пользователя
func (s *withdrawalService) GetWithdrawals(ctx context.Context, userID int64) ([]models.WithdrawalResponse, error) {
	withdrawals, err := s.withdrawalRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if len(withdrawals) == 0 {
		return []models.WithdrawalResponse{}, nil
	}

	responses := make([]models.WithdrawalResponse, 0, len(withdrawals))
	for _, w := range withdrawals {
		responses = append(responses, models.WithdrawalResponse{
			Order:       w.OrderNumber,
			Sum:         w.Sum,
			ProcessedAt: w.ProcessedAt.Format(time.RFC3339),
		})
	}

	return responses, nil
}
