package service

import (
	"context"
	"fmt"
	"personal-finance/internal/domain/category/repository"

	"personal-finance/internal/model"
)

type Service interface {
	Add(ctx context.Context, car model.Category) (model.Category, error)
	FindAll(ctx context.Context) ([]model.Category, error)
	FindByID(ctx context.Context, ID int) (model.Category, error)
	Update(ctx context.Context, ID int, car model.Category) (model.Category, error)
	Delete(ctx context.Context, ID int) error
}

type service struct {
	repo repository.Repository
}

func NewCategoryService(repo repository.Repository) Service {
	return service{
		repo: repo,
	}
}

func (s service) Add(ctx context.Context, category model.Category) (model.Category, error) {
	result, err := s.repo.Add(ctx, category)
	if err != nil {
		return model.Category{}, fmt.Errorf("error to add categories: %w", err)
	}
	return result, nil
}

func (s service) FindAll(ctx context.Context) ([]model.Category, error) {
	resultList, err := s.repo.FindAll(ctx)
	if err != nil {
		return []model.Category{}, fmt.Errorf("error to find categories: %w", err)
	}
	return resultList, nil
}

func (s service) FindByID(ctx context.Context, id int) (model.Category, error) {
	result, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return model.Category{}, fmt.Errorf("error to find categories: %w", err)
	}
	return result, nil
}

func (s service) Update(ctx context.Context, id int, car model.Category) (model.Category, error) {
	result, err := s.repo.Update(ctx, id, car)
	if err != nil {
		return model.Category{}, fmt.Errorf("error updating categories: %w", err)
	}
	return result, nil
}

func (s service) Delete(ctx context.Context, id int) error {
	err := s.repo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("error deleting categories: %w", err)
	}
	return nil
}
