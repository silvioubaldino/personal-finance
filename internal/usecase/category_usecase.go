package usecase

import (
	"context"
	"fmt"

	"personal-finance/internal/domain"
)

type CategoryRepository interface {
	Add(ctx context.Context, category domain.Category) (domain.Category, error)
	FindAll(ctx context.Context) ([]domain.Category, error)
	FindByID(ctx context.Context, ID *string) (domain.Category, error)
	Update(ctx context.Context, category domain.Category) (domain.Category, error)
	Delete(ctx context.Context, ID *string) error
}

type Category interface {
	Add(ctx context.Context, category domain.Category) (domain.Category, error)
	FindAll(ctx context.Context) ([]domain.Category, error)
	FindByID(ctx context.Context, ID *string) (domain.Category, error)
	Update(ctx context.Context, category domain.Category) (domain.Category, error)
	Delete(ctx context.Context, ID *string) error
}

type categoryUseCase struct {
	repo CategoryRepository
}

func NewCategory(repo CategoryRepository) Category {
	return categoryUseCase{
		repo: repo,
	}
}

func (uc categoryUseCase) Add(ctx context.Context, category domain.Category) (domain.Category, error) {
	result, err := uc.repo.Add(ctx, category)
	if err != nil {
		return domain.Category{}, fmt.Errorf("erro ao adicionar categoria: %w", err)
	}
	return result, nil
}

func (uc categoryUseCase) FindAll(ctx context.Context) ([]domain.Category, error) {
	resultList, err := uc.repo.FindAll(ctx)
	if err != nil {
		return []domain.Category{}, fmt.Errorf("erro ao buscar categorias: %w", err)
	}
	return resultList, nil
}

func (uc categoryUseCase) FindByID(ctx context.Context, id *string) (domain.Category, error) {
	result, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return domain.Category{}, fmt.Errorf("erro ao buscar categoria: %w", err)
	}
	return result, nil
}

func (uc categoryUseCase) Update(ctx context.Context, category domain.Category) (domain.Category, error) {
	result, err := uc.repo.Update(ctx, category)
	if err != nil {
		return domain.Category{}, fmt.Errorf("erro ao atualizar categoria: %w", err)
	}
	return result, nil
}

func (uc categoryUseCase) Delete(ctx context.Context, id *string) error {
	err := uc.repo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("erro ao deletar categoria: %w", err)
	}
	return nil
}
