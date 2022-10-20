package service

import (
	"context"
	"fmt"

	"personal-finance/internal/domain/transactionstatus/repository"
	"personal-finance/internal/model"
)

type Service interface {
	FindAll(ctx context.Context) ([]model.TransactionStatus, error)
}

type service struct {
	repo repository.Repository
}

func NewTransactionStatusService(repo repository.Repository) Service {
	return service{
		repo: repo,
	}
}

func (s service) FindAll(ctx context.Context) ([]model.TransactionStatus, error) {
	resultList, err := s.repo.FindAll(ctx)
	if err != nil {
		return []model.TransactionStatus{}, fmt.Errorf("error to find categories: %w", err)
	}
	return resultList, nil
}
