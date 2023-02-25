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
	repo      repository.Repository
	trService service.Movement
}

func NewConsolidatedService(trService service.Movement, repo repository.Repository) transaction {
	return transaction{
		repo:      repo,
		trService: trService,
	}
}

func (s transaction) FindByPeriod(ctx context.Context, period model.Period) ([]model.Transaction, error) {
	estimates, err := s.repo.FindByStatusByPeriod(ctx, service.TransactionStatusPlannedID, period)
	if err != nil {
		return []model.Transaction{}, fmt.Errorf("error to find planned transactions: %w", err)
	}

	var transactions []model.Transaction
	for _, estimate := range estimates {
		doneList, err := s.repo.FindByTransactionID(ctx, *estimate.ID, service.TransactionStatusPaidID)
		if err != nil {
			return []model.Transaction{}, fmt.Errorf("error to find realized transactions: %w", err)
		}
		transactions = append(transactions, model.BuildTransaction(estimate, doneList))
	}

	singleTransactions, err := s.repo.FindSingleTransactionByPeriod(ctx, service.TransactionStatusPaidID, period) // TODO mudar forma de implementar add para "done" sem "estimate"
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

func (s transaction) FindByID(ctx context.Context, id uuid.UUID) (model.Transaction, error) {
	estimate, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return model.Transaction{}, fmt.Errorf("error to find planned transactions: %w", err)
	}

	var doneList []model.Movement
	if estimate.MovementStatusID == service.TransactionStatusPlannedID {
		doneList, err = s.repo.FindByTransactionID(ctx, *estimate.ID, service.TransactionStatusPaidID)
		if err != nil {
			return model.Transaction{}, fmt.Errorf("error to find realized transactions: %w", err)
		}
	}

	return model.BuildTransaction(estimate, doneList), nil
}
