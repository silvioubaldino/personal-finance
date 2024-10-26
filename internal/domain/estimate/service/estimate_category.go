package service

import (
	"context"
	"github.com/google/uuid"

	"personal-finance/internal/domain/estimate/repository"
	"personal-finance/internal/model"
)

type Service interface {
	FindByMonth(ctx context.Context, month int, year int, userID string) ([]model.OutputEstimateCategories, error)
}

type service struct {
	repo repository.Repository
}

func NewEstimateService(repo repository.Repository) Service {
	return service{repo}
}

func (s service) FindByMonth(ctx context.Context, month int, year int, userID string) ([]model.OutputEstimateCategories, error) {
	estimateCategories, err := s.repo.FindCategoriesByMonth(ctx, month, year, userID)
	if err != nil {
		return nil, err
	}

	estimateSubCategories, err := s.repo.FindSubcategoriesByMonth(ctx, month, year, userID)
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
