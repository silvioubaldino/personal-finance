package service

import (
	"context"
	"fmt"

	"personal-finance/internal/domain/wallet/repository"
	"personal-finance/internal/model"

	"github.com/google/uuid"
)

type Service interface {
	RecalculateBalance(ctx context.Context, walletID *uuid.UUID) error
	Add(ctx context.Context, wallet model.Wallet) (model.Wallet, error)
	FindAll(ctx context.Context) ([]model.Wallet, error)
	FindByID(ctx context.Context, id *uuid.UUID) (model.Wallet, error)
	Update(ctx context.Context, id *uuid.UUID, wallet model.Wallet) (model.Wallet, error)
	Delete(ctx context.Context, id *uuid.UUID) error
}

type service struct {
	repo repository.Repository
}

func NewWalletService(repo repository.Repository) Service {
	return service{
		repo: repo,
	}
}

func (s service) RecalculateBalance(ctx context.Context, walletID *uuid.UUID) error {
	return s.repo.RecalculateBalance(ctx, walletID)
}

func (s service) Add(ctx context.Context, wallet model.Wallet) (model.Wallet, error) {
	result, err := s.repo.Add(ctx, wallet)
	if err != nil {
		return model.Wallet{}, fmt.Errorf("error to add wallets: %w", err)
	}
	return result, nil
}

func (s service) FindAll(ctx context.Context) ([]model.Wallet, error) {
	resultList, err := s.repo.FindAll(ctx)
	if err != nil {
		return []model.Wallet{}, fmt.Errorf("error to find wallets: %w", err)
	}
	return resultList, nil
}

func (s service) FindByID(ctx context.Context, id *uuid.UUID) (model.Wallet, error) {
	result, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return model.Wallet{}, fmt.Errorf("error to find wallets: %w", err)
	}
	return result, nil
}

func (s service) Update(ctx context.Context, id *uuid.UUID, wallet model.Wallet) (model.Wallet, error) {
	result, err := s.repo.Update(ctx, id, wallet)
	if err != nil {
		return model.Wallet{}, fmt.Errorf("error updating wallets: %w", err)
	}
	return result, nil
}

func (s service) Delete(ctx context.Context, id *uuid.UUID) error {
	err := s.repo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("error deleting wallets: %w", err)
	}
	return nil
}
