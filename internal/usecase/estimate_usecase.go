package usecase

import (
	"context"
	"fmt"

	"personal-finance/internal/domain"

	"github.com/google/uuid"
)

type EstimateRepository interface {
	FindCategoriesByMonth(ctx context.Context, month int, year int, userID string) ([]domain.EstimateCategories, error)
	FindSubcategoriesByMonth(ctx context.Context, month int, year int, userID string) ([]domain.EstimateSubCategories, error)
	AddEstimateCategory(ctx context.Context, category domain.EstimateCategories, userID string) (domain.EstimateCategories, error)
	AddEstimateSubCategory(ctx context.Context, subEstimate domain.EstimateSubCategories, userID string) (domain.EstimateSubCategories, error)
	UpdateEstimateCategoryAmount(ctx context.Context, id *uuid.UUID, amount float64, userID string) (domain.EstimateCategories, error)
	UpdateEstimateSubCategoryAmount(ctx context.Context, id *uuid.UUID, amount float64, userID string) (domain.EstimateSubCategories, error)
	DeleteEstimateCategory(ctx context.Context, id *uuid.UUID) error
	DeleteEstimateSubCategory(ctx context.Context, id *uuid.UUID) error
}

type Estimate interface {
	FindByMonth(ctx context.Context, month int, year int, userID string) ([]domain.EstimateCategories, error)
	AddEstimateCategory(ctx context.Context, category domain.EstimateCategories, userID string) (domain.EstimateCategories, error)
	AddEstimateSubCategory(ctx context.Context, subEstimate domain.EstimateSubCategories, userID string) (domain.EstimateSubCategories, error)
	UpdateEstimateCategoryAmount(ctx context.Context, id *uuid.UUID, amount float64, userID string) (domain.EstimateCategories, error)
	UpdateEstimateSubCategoryAmount(ctx context.Context, id *uuid.UUID, amount float64, userID string) (domain.EstimateSubCategories, error)
	DeleteEstimateCategory(ctx context.Context, id *uuid.UUID) error
	DeleteEstimateSubCategory(ctx context.Context, id *uuid.UUID) error
}

type estimateUseCase struct {
	repo EstimateRepository
}

func NewEstimate(repo EstimateRepository) Estimate {
	return estimateUseCase{
		repo: repo,
	}
}

func (uc estimateUseCase) FindByMonth(ctx context.Context, month int, year int, userID string) ([]domain.EstimateCategories, error) {
	estimateCategories, err := uc.repo.FindCategoriesByMonth(ctx, month, year, userID)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar estimativas por mês: %w", err)
	}

	estimateSubCategories, err := uc.repo.FindSubcategoriesByMonth(ctx, month, year, userID)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar subcategorias de estimativas: %w", err)
	}

	subCategoriesByCategory := make(map[uuid.UUID][]domain.EstimateSubCategories)
	for _, subCategory := range estimateSubCategories {
		if subCategory.EstimateCategoryID != nil {
			subCategoriesByCategory[*subCategory.EstimateCategoryID] = append(
				subCategoriesByCategory[*subCategory.EstimateCategoryID], subCategory)
		}
	}

	return estimateCategories, nil
}

func (uc estimateUseCase) AddEstimateCategory(ctx context.Context, category domain.EstimateCategories, userID string) (domain.EstimateCategories, error) {
	result, err := uc.repo.AddEstimateCategory(ctx, category, userID)
	if err != nil {
		return domain.EstimateCategories{}, fmt.Errorf("erro ao adicionar estimativa de categoria: %w", err)
	}
	return result, nil
}

func (uc estimateUseCase) AddEstimateSubCategory(ctx context.Context, subEstimate domain.EstimateSubCategories, userID string) (domain.EstimateSubCategories, error) {
	result, err := uc.repo.AddEstimateSubCategory(ctx, subEstimate, userID)
	if err != nil {
		return domain.EstimateSubCategories{}, fmt.Errorf("erro ao adicionar estimativa de subcategoria: %w", err)
	}
	return result, nil
}

func (uc estimateUseCase) UpdateEstimateCategoryAmount(ctx context.Context, id *uuid.UUID, amount float64, userID string) (domain.EstimateCategories, error) {
	result, err := uc.repo.UpdateEstimateCategoryAmount(ctx, id, amount, userID)
	if err != nil {
		return domain.EstimateCategories{}, fmt.Errorf("erro ao atualizar valor da estimativa de categoria: %w", err)
	}
	return result, nil
}

func (uc estimateUseCase) UpdateEstimateSubCategoryAmount(ctx context.Context, id *uuid.UUID, amount float64, userID string) (domain.EstimateSubCategories, error) {
	result, err := uc.repo.UpdateEstimateSubCategoryAmount(ctx, id, amount, userID)
	if err != nil {
		return domain.EstimateSubCategories{}, fmt.Errorf("erro ao atualizar valor da estimativa de subcategoria: %w", err)
	}
	return result, nil
}

func (uc estimateUseCase) DeleteEstimateCategory(ctx context.Context, id *uuid.UUID) error {
	err := uc.repo.DeleteEstimateCategory(ctx, id)
	if err != nil {
		return fmt.Errorf("erro ao deletar estimativa de categoria: %w", err)
	}
	return nil
}

func (uc estimateUseCase) DeleteEstimateSubCategory(ctx context.Context, id *uuid.UUID) error {
	err := uc.repo.DeleteEstimateSubCategory(ctx, id)
	if err != nil {
		return fmt.Errorf("erro ao deletar estimativa de subcategoria: %w", err)
	}
	return nil
}
