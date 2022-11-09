package service

import (
	"context"
	"fmt"

	"personal-finance/internal/domain/transaction/repository"
	"personal-finance/internal/model"

	"github.com/google/uuid"
)

type ConsolidatedService interface {
	FindConsolidatedTransactionByID(ctx context.Context, id uuid.UUID) (model.ConsolidatedTransaction, error)
	FindConsolidatedTransactionByPeriod(ctx context.Context, period model.Period) ([]model.ConsolidatedTransaction, error)
}

type consolidatedService struct {
	repo      repository.Repository
	trService Service
}

func NewConsolidatedService(trService Service, repo repository.Repository) consolidatedService {
	return consolidatedService{
		repo:      repo,
		trService: trService,
	}
}

func (s consolidatedService) FindConsolidatedTransactionByPeriod(ctx context.Context, period model.Period) ([]model.ConsolidatedTransaction, error) {
	plannedTransactions, err := s.repo.FindByTransactionStatusIDByPeriod(ctx, _transactionStatusPlannedID, period)
	if err != nil {
		return []model.ConsolidatedTransaction{}, fmt.Errorf("error to find planned transactions: %w", err)
	}

	var consolidatedTransactions []model.ConsolidatedTransaction
	for _, pt := range plannedTransactions {
		realizedTransactions, err := s.repo.FindByParentTransactionID(ctx, *pt.ID, _transactionStatusPaidID)
		if err != nil {
			return []model.ConsolidatedTransaction{}, fmt.Errorf("error to find realized transactions: %w", err)
		}
		consolidatedTransactions = append(consolidatedTransactions, model.BuildParentTransaction(pt, realizedTransactions))
	}

	singleTransactions, err := s.repo.FindSingleTransaction(ctx, _transactionStatusPaidID)
	if err != nil {
		return []model.ConsolidatedTransaction{}, fmt.Errorf("error to find singleTransactions: %w", err)
	}

	for _, singleTransaction := range singleTransactions {
		consolidatedTransactions = append(consolidatedTransactions, model.BuildParentTransaction(
			model.Transaction{},
			model.TransactionList{singleTransaction}))
	}

	if len(consolidatedTransactions) == 0 {
		return []model.ConsolidatedTransaction{}, model.BuildErrNotfound("resource not found")
	}
	return consolidatedTransactions, nil
}

func (s consolidatedService) FindConsolidatedTransactionByID(ctx context.Context, id uuid.UUID) (model.ConsolidatedTransaction, error) {
	plannedTransaction, err := s.repo.FindByIDByTransactionStatusID(ctx, id, 2)
	if err != nil {
		return model.ConsolidatedTransaction{}, fmt.Errorf("error to find planned transactions: %w", err)
	}

	realizedTransactions, err := s.repo.FindByParentTransactionID(ctx, *plannedTransaction.ID, _transactionStatusPaidID)
	if err != nil {
		return model.ConsolidatedTransaction{}, fmt.Errorf("error to find realized transactions: %w", err)
	}

	return model.BuildParentTransaction(plannedTransaction, realizedTransactions), nil
}
