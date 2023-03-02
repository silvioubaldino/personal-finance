package service

import (
	"context"
	"fmt"

	"personal-finance/internal/domain/movement/repository"
	"personal-finance/internal/domain/movement/service"

	"personal-finance/internal/model"

	"github.com/google/uuid"
)

type Transaction interface {
	FindByID(ctx context.Context, id uuid.UUID, userID string) (model.Transaction, error)
	FindByPeriod(ctx context.Context, period model.Period, userID string) ([]model.Transaction, error)
}

type transaction struct {
	movementRepo    repository.Repository
	movementService service.Movement
}

func NewTransactionService(repo repository.Repository) Transaction {
	return transaction{
		movementRepo: repo,
	}
}

func (s transaction) FindByID(ctx context.Context, id uuid.UUID, userID string) (model.Transaction, error) {
	estimate, err := s.movementRepo.FindByID(ctx, id, userID)
	if err != nil {
		return model.Transaction{}, fmt.Errorf("error to find estimate transactions: %w", err)
	}

	var doneList []model.Movement
	if estimate.StatusID == service.TransactionStatusPlannedID {
		doneList, err = s.movementRepo.FindByTransactionID(ctx, *estimate.ID, service.TransactionStatusPaidID, userID)
		if err != nil {
			return model.Transaction{}, fmt.Errorf("error to find done transactions: %w", err)
		}
	}

	return model.BuildTransaction(estimate, doneList), nil
}

func (s transaction) FindByPeriod(ctx context.Context, period model.Period, userID string) ([]model.Transaction, error) {
	estimates, err := s.movementRepo.FindByStatusByPeriod(ctx, service.TransactionStatusPlannedID, period, userID)
	if err != nil {
		return []model.Transaction{}, fmt.Errorf("error to find planned transactions: %w", err)
	}

	var transactions []model.Transaction
	for _, estimate := range estimates {
		doneList, err := s.movementRepo.FindByTransactionID(ctx, *estimate.ID, service.TransactionStatusPaidID, userID)
		if err != nil {
			return []model.Transaction{}, fmt.Errorf("error to find realized transactions: %w", err)
		}
		transactions = append(transactions, model.BuildTransaction(estimate, doneList))
	}

	if len(transactions) == 0 {
		return []model.Transaction{}, model.BuildErrNotfound("resource not found")
	}
	return transactions, nil
}
