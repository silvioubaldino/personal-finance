package usecase

import (
	"context"
	"fmt"

	"personal-finance/internal/domain"

	"github.com/google/uuid"
)

type WalletRepository interface {
	Add(ctx context.Context, wallet domain.Wallet, userID string) (domain.Wallet, error)
	FindAll(ctx context.Context, userID string) ([]domain.Wallet, error)
	FindByID(ctx context.Context, ID *uuid.UUID, userID string) (domain.Wallet, error)
	Update(ctx context.Context, wallet domain.Wallet, userID string) (domain.Wallet, error)
	Delete(ctx context.Context, ID *uuid.UUID) error
	RecalculateBalance(ctx context.Context, walletID *uuid.UUID, userID string) error
}

type Wallet interface {
	Add(ctx context.Context, wallet domain.Wallet, userID string) (domain.Wallet, error)
	FindAll(ctx context.Context, userID string) ([]domain.Wallet, error)
	FindByID(ctx context.Context, ID *uuid.UUID, userID string) (domain.Wallet, error)
	Update(ctx context.Context, wallet domain.Wallet, userID string) (domain.Wallet, error)
	Delete(ctx context.Context, ID *uuid.UUID) error
	RecalculateBalance(ctx context.Context, walletID *uuid.UUID, userID string) error
}

type walletUseCase struct {
	repo WalletRepository
}

func NewWallet(repo WalletRepository) Wallet {
	return walletUseCase{
		repo: repo,
	}
}

func (uc walletUseCase) RecalculateBalance(ctx context.Context, walletID *uuid.UUID, userID string) error {
	err := uc.repo.RecalculateBalance(ctx, walletID, userID)
	if err != nil {
		return fmt.Errorf("erro ao recalcular saldo da carteira: %w", err)
	}
	return nil
}

func (uc walletUseCase) Add(ctx context.Context, wallet domain.Wallet, userID string) (domain.Wallet, error) {
	result, err := uc.repo.Add(ctx, wallet, userID)
	if err != nil {
		return domain.Wallet{}, fmt.Errorf("erro ao adicionar carteira: %w", err)
	}
	return result, nil
}

func (uc walletUseCase) FindAll(ctx context.Context, userID string) ([]domain.Wallet, error) {
	resultList, err := uc.repo.FindAll(ctx, userID)
	if err != nil {
		return []domain.Wallet{}, fmt.Errorf("erro ao buscar carteiras: %w", err)
	}
	return resultList, nil
}

func (uc walletUseCase) FindByID(ctx context.Context, id *uuid.UUID, userID string) (domain.Wallet, error) {
	result, err := uc.repo.FindByID(ctx, id, userID)
	if err != nil {
		return domain.Wallet{}, fmt.Errorf("erro ao buscar carteira: %w", err)
	}
	return result, nil
}

func (uc walletUseCase) Update(ctx context.Context, wallet domain.Wallet, userID string) (domain.Wallet, error) {
	result, err := uc.repo.Update(ctx, wallet, userID)
	if err != nil {
		return domain.Wallet{}, fmt.Errorf("erro ao atualizar carteira: %w", err)
	}
	return result, nil
}

func (uc walletUseCase) Delete(ctx context.Context, id *uuid.UUID) error {
	err := uc.repo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("erro ao deletar carteira: %w", err)
	}
	return nil
}
