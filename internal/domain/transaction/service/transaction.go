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
	FindByID(ctx context.Context, id uuid.UUID) (model.Transaction, error)
	FindByPeriod(ctx context.Context, period model.Period) ([]model.Transaction, error)
}

type transaction struct {
	movementRepo    repository.Repository
	movementService service.Movement
}

func NewTransactionService(repo repository.Repository) transaction {
	return transaction{
		movementRepo: repo,
	}
}

func (s transaction) FindByID(ctx context.Context, id uuid.UUID) (model.Transaction, error) {
	estimate, err := s.movementRepo.FindByID(ctx, id)
	if err != nil {
		return model.Transaction{}, fmt.Errorf("error to find estimate transactions: %w", err)
	}

	var doneList []model.Movement
	if estimate.MovementStatusID == service.TransactionStatusPlannedID {
		doneList, err = s.movementRepo.FindByTransactionID(ctx, *estimate.ID, service.TransactionStatusPaidID)
		if err != nil {
			return model.Transaction{}, fmt.Errorf("error to find done transactions: %w", err)
		}
	}

	return model.BuildTransaction(estimate, doneList), nil
}

func (s transaction) FindByPeriod(ctx context.Context, period model.Period) ([]model.Transaction, error) {
	estimates, err := s.movementRepo.FindByStatusByPeriod(ctx, service.TransactionStatusPlannedID, period)
	if err != nil {
		return []model.Transaction{}, fmt.Errorf("error to find planned transactions: %w", err)
	}

	var transactions []model.Transaction
	for _, estimate := range estimates {
		doneList, err := s.movementRepo.FindByTransactionID(ctx, *estimate.ID, service.TransactionStatusPaidID)
		if err != nil {
			return []model.Transaction{}, fmt.Errorf("error to find realized transactions: %w", err)
		}
		transactions = append(transactions, model.BuildTransaction(estimate, doneList))
	}

	singleTransactions, err := s.movementRepo.FindSingleTransactionByPeriod(ctx, service.TransactionStatusPaidID, period) // TODO mudar forma de implementar add para "done" sem "estimate"
	if err != nil {
		return []model.Transaction{}, fmt.Errorf("error to find singleTransactions: %w", err)
	}

	for _, singleTransaction := range singleTransactions {
		transactions = append(transactions, model.BuildTransaction(
			model.Movement{},
			model.MovementList{singleTransaction}))
	}

	if len(transactions) == 0 {
		return []model.Transaction{}, model.BuildErrNotfound("resource not found")
	}
	return transactions, nil
}
