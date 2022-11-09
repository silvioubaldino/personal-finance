package service

import (
	"context"
	"fmt"

	"personal-finance/internal/domain/transaction/repository"
	walletService "personal-finance/internal/domain/wallet/service"
	"personal-finance/internal/model"
)

const (
	_transactionStatusPaidID    = 1
	_transactionStatusPlannedID = 2
)

type Service interface {
	Add(ctx context.Context, transaction model.Transaction) (model.Transaction, error)
	FindAll(ctx context.Context) ([]model.Transaction, error)
	FindByID(ctx context.Context, ID int) (model.Transaction, error)
	FindByMonth(ctx context.Context, period model.Period) ([]model.Transaction, error)
	BalanceByPeriod(ctx context.Context, period model.Period) (model.Balance, error)
	Update(ctx context.Context, ID int, transaction model.Transaction) (model.Transaction, error)
	Delete(ctx context.Context, ID int) error
}

type service struct {
	repo      repository.Repository
	walletSvc walletService.Service
}

func NewTransactionService(repo repository.Repository, walletSvc walletService.Service) Service {
	return service{
		repo:      repo,
		walletSvc: walletSvc,
	}
}

func (s service) Add(ctx context.Context, transaction model.Transaction) (model.Transaction, error) {
	result, err := s.repo.Add(ctx, transaction)
	if err != nil {
		return model.Transaction{}, fmt.Errorf("error to add transactions: %w", err)
	}

	if transaction.TransactionStatusID == _transactionStatusPaidID {
		wallet, err := s.walletSvc.FindByID(ctx, transaction.WalletID)
		if err != nil {
			return model.Transaction{}, fmt.Errorf("error to update balance: %w", err)
		}

		wallet.Balance += transaction.Amount
		_, err = s.walletSvc.Update(ctx, transaction.WalletID, wallet)
		if err != nil {
			return model.Transaction{}, fmt.Errorf("error to update balance: %w", err)
		}
	}
	return result, nil
}

func (s service) FindAll(ctx context.Context) ([]model.Transaction, error) {
	resultList, err := s.repo.FindAll(ctx)
	if err != nil {
		return []model.Transaction{}, fmt.Errorf("error to find transactions: %w", err)
	}
	return resultList, nil
}

func (s service) FindByID(ctx context.Context, id int) (model.Transaction, error) {
	result, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return model.Transaction{}, fmt.Errorf("error to find transactions: %w", err)
	}
	return result, nil
}

func (s service) FindByMonth(ctx context.Context, period model.Period) ([]model.Transaction, error) {
	result, err := s.repo.FindByMonth(ctx, period)
	if err != nil {
		return []model.Transaction{}, fmt.Errorf("error to find transactions: %w", err)
	}
	return result, nil
}

func (s service) BalanceByPeriod(ctx context.Context, period model.Period) (model.Balance, error) {
	result, err := s.repo.FindByMonth(ctx, period)
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

func (s service) Update(ctx context.Context, id int, transaction model.Transaction) (model.Transaction, error) {
	result, err := s.repo.Update(ctx, id, transaction)
	if err != nil {
		return model.Transaction{}, fmt.Errorf("error updating transactions: %w", err)
	}
	return result, nil
}

func (s service) Delete(ctx context.Context, id int) error {
	err := s.repo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("error deleting transactions: %w", err)
	}
	return nil
}
