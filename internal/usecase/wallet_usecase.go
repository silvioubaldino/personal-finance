package usecase

import (
	"context"
	"fmt"

	"personal-finance/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WalletRepository interface {
	Add(ctx context.Context, wallet domain.Wallet) (domain.Wallet, error)
	AddConsistent(ctx context.Context, tx *gorm.DB, wallet domain.Wallet) (domain.Wallet, error)
	FindAll(ctx context.Context) ([]domain.Wallet, error)
	FindByID(ctx context.Context, ID *uuid.UUID) (domain.Wallet, error)
	Update(ctx context.Context, wallet domain.Wallet) (domain.Wallet, error)
	UpdateAmount(ctx context.Context, tx *gorm.DB, walletID *uuid.UUID, amout float64) error
	Delete(ctx context.Context, ID *uuid.UUID) error
	RecalculateBalance(ctx context.Context, walletID *uuid.UUID) error
}

type Wallet struct {
	repo WalletRepository
}

func NewWallet(repo WalletRepository) Wallet {
	return Wallet{
		repo: repo,
	}
}

func (uc Wallet) RecalculateBalance(ctx context.Context, walletID *uuid.UUID) error {
	err := uc.repo.RecalculateBalance(ctx, walletID)
	if err != nil {
		return fmt.Errorf("erro ao recalcular saldo da carteira: %w", err)
	}
	return nil
}

func (uc Wallet) Add(ctx context.Context, wallet domain.Wallet) (domain.Wallet, error) {
	result, err := uc.repo.Add(ctx, wallet)
	if err != nil {
		return domain.Wallet{}, fmt.Errorf("erro ao adicionar carteira: %w", err)
	}
	return result, nil
}

func (uc Wallet) FindAll(ctx context.Context) ([]domain.Wallet, error) {
	resultList, err := uc.repo.FindAll(ctx)
	if err != nil {
		return []domain.Wallet{}, fmt.Errorf("erro ao buscar carteiras: %w", err)
	}
	return resultList, nil
}

func (uc Wallet) FindByID(ctx context.Context, id *uuid.UUID) (domain.Wallet, error) {
	result, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return domain.Wallet{}, fmt.Errorf("erro ao buscar carteira: %w", err)
	}
	return result, nil
}

func (uc Wallet) Update(ctx context.Context, wallet domain.Wallet) (domain.Wallet, error) {
	result, err := uc.repo.Update(ctx, wallet)
	if err != nil {
		return domain.Wallet{}, fmt.Errorf("erro ao atualizar carteira: %w", err)
	}
	return result, nil
}

func (uc Wallet) Delete(ctx context.Context, id *uuid.UUID) error {
	err := uc.repo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("erro ao deletar carteira: %w", err)
	}
	return nil
}
