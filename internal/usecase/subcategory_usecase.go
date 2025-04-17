package usecase

import (
	"context"
	"fmt"

	"personal-finance/internal/domain"

	"github.com/google/uuid"
)

type SubCategoryRepository interface {
	Add(ctx context.Context, subcategory domain.SubCategory) (domain.SubCategory, error)
	FindAll(ctx context.Context) (domain.SubCategoryList, error)
	FindByID(ctx context.Context, ID uuid.UUID) (domain.SubCategory, error)
	FindByCategoryID(ctx context.Context, categoryID uuid.UUID) (domain.SubCategoryList, error)
	IsSubCategoryBelongsToCategory(ctx context.Context, subcategoryID uuid.UUID, categoryID uuid.UUID) (bool, error)
	Update(ctx context.Context, subcategory domain.SubCategory) (domain.SubCategory, error)
	Delete(ctx context.Context, ID uuid.UUID) error
}

type SubCategory interface {
	Add(ctx context.Context, subcategory domain.SubCategory) (domain.SubCategory, error)
	FindAll(ctx context.Context) (domain.SubCategoryList, error)
	FindByID(ctx context.Context, ID uuid.UUID) (domain.SubCategory, error)
	FindByCategoryID(ctx context.Context, categoryID uuid.UUID) (domain.SubCategoryList, error)
	IsSubCategoryBelongsToCategory(ctx context.Context, subcategoryID uuid.UUID, categoryID uuid.UUID) (bool, error)
	Update(ctx context.Context, subcategory domain.SubCategory) (domain.SubCategory, error)
	Delete(ctx context.Context, ID uuid.UUID) error
}

type subCategoryUseCase struct {
	repo SubCategoryRepository
}

func NewSubCategory(repo SubCategoryRepository) SubCategory {
	return subCategoryUseCase{
		repo: repo,
	}
}

func (uc subCategoryUseCase) Add(ctx context.Context, subcategory domain.SubCategory) (domain.SubCategory, error) {
	result, err := uc.repo.Add(ctx, subcategory)
	if err != nil {
		return domain.SubCategory{}, fmt.Errorf("erro ao adicionar subcategoria: %w", err)
	}
	return result, nil
}

func (uc subCategoryUseCase) FindAll(ctx context.Context) (domain.SubCategoryList, error) {
	resultList, err := uc.repo.FindAll(ctx)
	if err != nil {
		return domain.SubCategoryList{}, fmt.Errorf("erro ao buscar subcategorias: %w", err)
	}
	return resultList, nil
}

func (uc subCategoryUseCase) FindByID(ctx context.Context, id uuid.UUID) (domain.SubCategory, error) {
	result, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return domain.SubCategory{}, fmt.Errorf("erro ao buscar subcategoria: %w", err)
	}
	return result, nil
}

func (uc subCategoryUseCase) FindByCategoryID(ctx context.Context, categoryID uuid.UUID) (domain.SubCategoryList, error) {
	resultList, err := uc.repo.FindByCategoryID(ctx, categoryID)
	if err != nil {
		return domain.SubCategoryList{}, fmt.Errorf("erro ao buscar subcategorias por categoria: %w", err)
	}
	return resultList, nil
}

func (uc subCategoryUseCase) IsSubCategoryBelongsToCategory(ctx context.Context, subcategoryID uuid.UUID, categoryID uuid.UUID) (bool, error) {
	result, err := uc.repo.IsSubCategoryBelongsToCategory(ctx, subcategoryID, categoryID)
	if err != nil {
		return false, fmt.Errorf("erro ao verificar se a subcategoria pertence à categoria: %w", err)
	}
	return result, nil
}

func (uc subCategoryUseCase) Update(ctx context.Context, subcategory domain.SubCategory) (domain.SubCategory, error) {
	result, err := uc.repo.Update(ctx, subcategory)
	if err != nil {
		return domain.SubCategory{}, fmt.Errorf("erro ao atualizar subcategoria: %w", err)
	}
	return result, nil
}

func (uc subCategoryUseCase) Delete(ctx context.Context, id uuid.UUID) error {
	err := uc.repo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("erro ao deletar subcategoria: %w", err)
	}
	return nil
}
