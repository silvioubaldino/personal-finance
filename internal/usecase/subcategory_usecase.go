package usecase

import (
	"context"
	"fmt"

	"personal-finance/internal/domain"
)

type SubCategoryRepository interface {
	Add(ctx context.Context, subcategory domain.SubCategory, userID string) (domain.SubCategory, error)
	FindAll(ctx context.Context, userID string) (domain.SubCategoryList, error)
	FindByID(ctx context.Context, ID *string, userID string) (domain.SubCategory, error)
	FindByCategoryID(ctx context.Context, categoryID *string, userID string) (domain.SubCategoryList, error)
	Update(ctx context.Context, subcategory domain.SubCategory, userID string) (domain.SubCategory, error)
	Delete(ctx context.Context, ID *string) error
}

type SubCategory interface {
	Add(ctx context.Context, subcategory domain.SubCategory, userID string) (domain.SubCategory, error)
	FindAll(ctx context.Context, userID string) (domain.SubCategoryList, error)
	FindByID(ctx context.Context, ID *string, userID string) (domain.SubCategory, error)
	FindByCategoryID(ctx context.Context, categoryID *string, userID string) (domain.SubCategoryList, error)
	Update(ctx context.Context, subcategory domain.SubCategory, userID string) (domain.SubCategory, error)
	Delete(ctx context.Context, ID *string) error
}

type subCategoryUseCase struct {
	repo SubCategoryRepository
}

func NewSubCategory(repo SubCategoryRepository) SubCategory {
	return subCategoryUseCase{
		repo: repo,
	}
}

func (uc subCategoryUseCase) Add(ctx context.Context, subcategory domain.SubCategory, userID string) (domain.SubCategory, error) {
	result, err := uc.repo.Add(ctx, subcategory, userID)
	if err != nil {
		return domain.SubCategory{}, fmt.Errorf("erro ao adicionar subcategoria: %w", err)
	}
	return result, nil
}

func (uc subCategoryUseCase) FindAll(ctx context.Context, userID string) (domain.SubCategoryList, error) {
	resultList, err := uc.repo.FindAll(ctx, userID)
	if err != nil {
		return domain.SubCategoryList{}, fmt.Errorf("erro ao buscar subcategorias: %w", err)
	}
	return resultList, nil
}

func (uc subCategoryUseCase) FindByID(ctx context.Context, id *string, userID string) (domain.SubCategory, error) {
	result, err := uc.repo.FindByID(ctx, id, userID)
	if err != nil {
		return domain.SubCategory{}, fmt.Errorf("erro ao buscar subcategoria: %w", err)
	}
	return result, nil
}

func (uc subCategoryUseCase) FindByCategoryID(ctx context.Context, categoryID *string, userID string) (domain.SubCategoryList, error) {
	resultList, err := uc.repo.FindByCategoryID(ctx, categoryID, userID)
	if err != nil {
		return domain.SubCategoryList{}, fmt.Errorf("erro ao buscar subcategorias por categoria: %w", err)
	}
	return resultList, nil
}

func (uc subCategoryUseCase) Update(ctx context.Context, subcategory domain.SubCategory, userID string) (domain.SubCategory, error) {
	result, err := uc.repo.Update(ctx, subcategory, userID)
	if err != nil {
		return domain.SubCategory{}, fmt.Errorf("erro ao atualizar subcategoria: %w", err)
	}
	return result, nil
}

func (uc subCategoryUseCase) Delete(ctx context.Context, id *string) error {
	err := uc.repo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("erro ao deletar subcategoria: %w", err)
	}
	return nil
}
