package service

import (
	"context"

	"personal-finance/internal/domain/estimate/repository"
	"personal-finance/internal/model"

	"github.com/google/uuid"
)

type Service interface {
	FindByMonth(ctx context.Context, month int, year int) ([]model.OutputEstimateCategories, error)
	AddEstimate(ctx context.Context, category model.EstimateCategories) (model.OutputEstimateCategories, error)
	AddSubEstimate(ctx context.Context, subEstimate model.EstimateSubCategories) (model.OutputEstimateSubCategories, error)
	UpdateEstimateAmount(ctx context.Context, id *uuid.UUID, amount float64) (model.OutputEstimateCategories, error)
	UpdateSubEstimateAmount(ctx context.Context, id *uuid.UUID, amount float64) (model.OutputEstimateSubCategories, error)
}

type service struct {
	repo repository.Repository
}

func NewEstimateService(repo repository.Repository) Service {
	return service{repo}
}

func (s service) FindByMonth(
	ctx context.Context,
	month int,
	year int,
) ([]model.OutputEstimateCategories, error) {
	estimateCategories, err := s.repo.FindCategoriesByMonth(ctx, month, year)
	if err != nil {
		return nil, err
	}

	estimateSubCategories, err := s.repo.FindSubcategoriesByMonth(ctx, month, year)
	if err != nil {
		return nil, err
	}

	outputEstimate := make([]model.OutputEstimateCategories, len(estimateCategories))
	for i, category := range estimateCategories {
		outputEstimate[i] = model.ToOutputEstimateCategories(category)
	}

	subCategoriesMap := make(map[uuid.UUID][]model.OutputEstimateSubCategories)
	for _, subCategory := range estimateSubCategories {
		subCategoriesMap[*subCategory.EstimateCategoryID] = append(
			subCategoriesMap[*subCategory.EstimateCategoryID],
			model.ToOutputEstimateSubCategories(subCategory))
	}

	for i, category := range outputEstimate {
		if subCategories, found := subCategoriesMap[*category.ID]; found {
			outputEstimate[i].EstimateSubCategories = subCategories
		}
	}

	return outputEstimate, nil
}

func (s service) AddEstimate(
	ctx context.Context,
	estimate model.EstimateCategories,
) (model.OutputEstimateCategories, error) {
	estimate, err := s.repo.AddEstimate(ctx, estimate)
	if err != nil {
		return model.OutputEstimateCategories{}, err
	}

	return model.ToOutputEstimateCategories(estimate), nil
}

func (s service) AddSubEstimate(
	ctx context.Context,
	subEstimate model.EstimateSubCategories,
) (model.OutputEstimateSubCategories, error) {
	subEstimate, err := s.repo.AddSubEstimate(ctx, subEstimate)
	if err != nil {
		return model.OutputEstimateSubCategories{}, err
	}

	return model.ToOutputEstimateSubCategories(subEstimate), nil
}

func (s service) UpdateEstimateAmount(
	ctx context.Context,
	id *uuid.UUID,
	amount float64,
) (model.OutputEstimateCategories, error) {
	estimate, err := s.repo.UpdateEstimateAmount(ctx, id, amount)
	if err != nil {
		return model.OutputEstimateCategories{}, err
	}
	return model.ToOutputEstimateCategories(estimate), nil
}

func (s service) UpdateSubEstimateAmount(
	ctx context.Context,
	id *uuid.UUID,
	amount float64,
) (model.OutputEstimateSubCategories, error) {
	subEstimate, err := s.repo.UpdateSubEstimateAmount(ctx, id, amount)
	if err != nil {
		return model.OutputEstimateSubCategories{}, err
	}
	return model.ToOutputEstimateSubCategories(subEstimate), nil
}
