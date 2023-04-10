package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"personal-finance/internal/domain/movement/repository"
	"personal-finance/internal/domain/transaction/service"
	"personal-finance/internal/model"
)

type Movement interface {
	Add(ctx context.Context, transaction model.Movement, userID string) (model.Movement, error)
	FindByID(ctx context.Context, id uuid.UUID, userID string) (model.Movement, error)
	FindByPeriod(ctx context.Context, period model.Period, userID string) ([]model.Movement, error)
	BalanceByPeriod(ctx context.Context, period model.Period, userID string) (model.Balance, error)
	Update(ctx context.Context, id uuid.UUID, transaction model.Movement, userID string) (model.Movement, error)
	Delete(ctx context.Context, id uuid.UUID, userID string) error
}

type movement struct {
	repo           repository.Repository
	transactionSvc service.Transaction
}

func NewMovementService(repo repository.Repository, transactionSvc service.Transaction) Movement {
	return movement{
		repo:           repo,
		transactionSvc: transactionSvc,
	}
}

func (s movement) Add(ctx context.Context, movement model.Movement, userID string) (model.Movement, error) {
	if movement.TransactionID == nil {
		if movement.StatusID == model.TransactionStatusPlannedID {
			movement, err := s.repo.Add(ctx, movement, userID)
			if err != nil {
				return model.Movement{}, fmt.Errorf("error to add transactions: %w", err)
			}
			return movement, nil
		}

		if movement.StatusID == model.TransactionStatusPaidID {
			transaction, err := s.transactionSvc.AddDirectDoneTransaction(ctx, movement)
			if err != nil {
				return model.Movement{}, fmt.Errorf("error to add transactions: %w", err)
			}
			return *transaction.Estimate, nil
		}
	}

	if movement.StatusID == model.TransactionStatusPlannedID {
		return model.Movement{}, errors.New("planned transactions must not have transactionID")
	}

	movement, err := s.repo.AddUpdatingWallet(ctx, nil, movement, userID)
	if err != nil {
		return model.Movement{}, fmt.Errorf("error to add transactions: %w", err)
	}
	return movement, nil
}

func (s movement) FindByID(ctx context.Context, id uuid.UUID, userID string) (model.Movement, error) {
	result, err := s.repo.FindByID(ctx, id, userID)
	if err != nil {
		return model.Movement{}, fmt.Errorf("error to find transactions: %w", err)
	}
	return result, nil
}

func (s movement) FindByPeriod(ctx context.Context, period model.Period, userID string) ([]model.Movement, error) {
	result, err := s.repo.FindByPeriod(ctx, period, userID)
	if err != nil {
		return []model.Movement{}, fmt.Errorf("error to find transactions: %w", err)
	}
	return result, nil
}

func (s movement) BalanceByPeriod(ctx context.Context, period model.Period, userID string) (model.Balance, error) {
	result, err := s.repo.FindByPeriod(ctx, period, userID)
	balance := model.Balance{Period: period}
	if err != nil {
		return model.Balance{}, fmt.Errorf("error to find transactions: %w", err)
	}
	for _, transaction := range result {
		if transaction.Amount > 0 {
			balance.Income += transaction.Amount
		}
		if transaction.Amount < 0 {
			balance.Expense += transaction.Amount
		}
	}
	return balance, nil
}

func (s movement) Update(ctx context.Context, id uuid.UUID, transaction model.Movement, userID string) (model.Movement, error) {
	result, err := s.repo.Update(ctx, id, transaction, userID)
	if err != nil {
		return model.Movement{}, fmt.Errorf("error updating transactions: %w", err)
	}
	return result, nil
}

func (s movement) Delete(ctx context.Context, id uuid.UUID, userID string) error {
	err := s.repo.Delete(ctx, id, userID)
	if err != nil {
		return fmt.Errorf("error deleting transactions: %w", err)
	}
	return nil
}
