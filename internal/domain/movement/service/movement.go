package service

import (
	"context"
	"fmt"

	"personal-finance/internal/domain/movement/repository"

	"github.com/google/uuid"

	walletService "personal-finance/internal/domain/wallet/service"
	"personal-finance/internal/model"
)

const (
	TransactionStatusPaidID    = 1
	TransactionStatusPlannedID = 2
)

type Movement interface {
	Add(ctx context.Context, transaction model.Movement) (model.Movement, error)
	FindByID(ctx context.Context, ID uuid.UUID) (model.Movement, error)
	FindByPeriod(ctx context.Context, period model.Period) ([]model.Movement, error)
	BalanceByPeriod(ctx context.Context, period model.Period) (model.Balance, error)
	Update(ctx context.Context, ID uuid.UUID, transaction model.Movement) (model.Movement, error)
	Delete(ctx context.Context, ID uuid.UUID) error
}

type movement struct {
	repo      repository.Repository
	walletSvc walletService.Service
}

func NewMovementService(repo repository.Repository, walletSvc walletService.Service) Movement {
	return movement{
		repo:      repo,
		walletSvc: walletSvc,
	}
}

func (s movement) Add(ctx context.Context, transaction model.Movement) (model.Movement, error) {
	result, err := s.repo.Add(ctx, transaction) // TODO Bug: sucesso em add mesmo ao falhar o update de wallet, garantir que todos sejam executados ou nenhum
	if err != nil {
		return model.Movement{}, fmt.Errorf("error to add transactions: %w", err)
	}

	if transaction.MovementStatusID == TransactionStatusPaidID {
		wallet, err := s.walletSvc.FindByID(ctx, transaction.WalletID, "DWAU6BuOd5OPDkPank6fcrqluuz1") // TODO
		if err != nil {
			return model.Movement{}, fmt.Errorf("error to update balance: %w", err)
		}

		wallet.Balance += transaction.Amount
		_, err = s.walletSvc.Update(ctx, transaction.WalletID, wallet, "DWAU6BuOd5OPDkPank6fcrqluuz1") // TODO
		if err != nil {
			return model.Movement{}, fmt.Errorf("error to update balance: %w", err)
		}
	}
	return result, nil
}

func (s movement) FindByID(ctx context.Context, id uuid.UUID) (model.Movement, error) {
	result, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return model.Movement{}, fmt.Errorf("error to find transactions: %w", err)
	}
	return result, nil
}

func (s movement) FindByPeriod(ctx context.Context, period model.Period) ([]model.Movement, error) {
	result, err := s.repo.FindByPeriod(ctx, period)
	if err != nil {
		return []model.Movement{}, fmt.Errorf("error to find transactions: %w", err)
	}
	return result, nil
}

func (s movement) BalanceByPeriod(ctx context.Context, period model.Period) (model.Balance, error) {
	result, err := s.repo.FindByPeriod(ctx, period)
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

func (s movement) Update(ctx context.Context, id uuid.UUID, transaction model.Movement) (model.Movement, error) {
	result, err := s.repo.Update(ctx, id, transaction)
	if err != nil {
		return model.Movement{}, fmt.Errorf("error updating transactions: %w", err)
	}
	return result, nil
}

func (s movement) Delete(ctx context.Context, id uuid.UUID) error {
	err := s.repo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("error deleting transactions: %w", err)
	}
	return nil
}
