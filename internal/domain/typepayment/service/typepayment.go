package service

import (
	"context"
	"fmt"

	"personal-finance/internal/domain/typepayment/repository"
	"personal-finance/internal/model"
)

type Service interface {
	Add(ctx context.Context, typePayment model.TypePayment, userID string) (model.TypePayment, error)
	FindAll(ctx context.Context, userID string) ([]model.TypePayment, error)
	FindByID(ctx context.Context, ID int, userID string) (model.TypePayment, error)
	Update(ctx context.Context, ID int, typePayment model.TypePayment, userID string) (model.TypePayment, error)
	Delete(ctx context.Context, ID int) error
}

type service struct {
	repo repository.Repository
}

func NewTypePaymentService(repo repository.Repository) Service {
	return service{
		repo: repo,
	}
}

func (s service) Add(ctx context.Context, typePayment model.TypePayment, userID string) (model.TypePayment, error) {
	result, err := s.repo.Add(ctx, typePayment, userID)
	if err != nil {
		return model.TypePayment{}, fmt.Errorf("error to add typePayments: %w", err)
	}
	return result, nil
}

func (s service) FindAll(ctx context.Context, userID string) ([]model.TypePayment, error) {
	resultList, err := s.repo.FindAll(ctx, userID)
	if err != nil {
		return []model.TypePayment{}, fmt.Errorf("error to find typePayments: %w", err)
	}
	return resultList, nil
}

func (s service) FindByID(ctx context.Context, id int, userID string) (model.TypePayment, error) {
	result, err := s.repo.FindByID(ctx, id, userID)
	if err != nil {
		return model.TypePayment{}, fmt.Errorf("error to find typePayments: %w", err)
	}
	return result, nil
}

func (s service) Update(ctx context.Context, id int, typePayment model.TypePayment, userID string) (model.TypePayment, error) {
	result, err := s.repo.Update(ctx, id, typePayment, userID)
	if err != nil {
		return model.TypePayment{}, fmt.Errorf("error updating typePayments: %w", err)
	}
	return result, nil
}

func (s service) Delete(ctx context.Context, id int) error {
	err := s.repo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("error deleting typePayments: %w", err)
	}
	return nil
}
