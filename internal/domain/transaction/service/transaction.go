package service

import (
	"context"
	"fmt"

	"personal-finance/internal/domain/transaction/repository"
	"personal-finance/internal/model"
)

type Service interface {
	Add(ctx context.Context, transaction model.Transaction) (model.Transaction, error)
	FindAll(ctx context.Context) ([]model.Transaction, error)
	FindByID(ctx context.Context, ID int) (model.Transaction, error)
	Update(ctx context.Context, ID int, transaction model.Transaction) (model.Transaction, error)
	Delete(ctx context.Context, ID int) error
}

type service struct {
	repo repository.Repository
}

func NewTransactionService(repo repository.Repository) Service {
	return service{
		repo: repo,
	}
}

func (s service) Add(ctx context.Context, transaction model.Transaction) (model.Transaction, error) {
	result, err := s.repo.Add(ctx, transaction)
	if err != nil {
		return model.Transaction{}, fmt.Errorf("error to add transactions: %w", err)
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
