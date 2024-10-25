package repository

import (
	"context"
	"gorm.io/gorm"

	"personal-finance/internal/model"
)

type Repository interface {
	Add(ctx context.Context, category model.EstimateCategories, userID string) (model.EstimateCategories, error)
	FindCategoriesByMonth(_ context.Context, month int, year int, userID string) ([]model.EstimateCategories, error)
	FindSubcategoriesByMonth(_ context.Context, month int, year int, userID string) ([]model.EstimateSubCategories, error)
	Update(ctx context.Context, id int, category model.EstimateCategories, userID string) (model.EstimateCategories, error)
}

type PgRepository struct {
	gorm *gorm.DB
}

func NewPgRepository(gorm *gorm.DB) Repository {
	return PgRepository{gorm}
}

func (p PgRepository) Add(ctx context.Context, category model.EstimateCategories, userID string) (model.EstimateCategories, error) {
	//TODO implement me
	panic("implement me")
}

func (p PgRepository) FindCategoriesByMonth(_ context.Context, month int, year int, userID string) ([]model.EstimateCategories, error) {
	var estimates []model.EstimateCategories
	resultCategories := p.gorm.
		Where("estimate_categories.user_id = ?", userID).
		Where("estimate_categories.month = ? AND estimate_categories.year = ?", month, year).
		Joins("LEFT JOIN categories c ON estimate_categories.category_id = c.id").
		Select([]string{
			"estimate_categories.*",
			"c.description as category_name",
		}).
		Find(&estimates)
	if err := resultCategories.Error; err != nil {
		return []model.EstimateCategories{}, err
	}
	return estimates, nil
}

func (p PgRepository) FindSubcategoriesByMonth(_ context.Context, month int, year int, userID string) ([]model.EstimateSubCategories, error) {
	var estimateSubcategories []model.EstimateSubCategories
	resultSubCategories := p.gorm.
		Where("estimate_sub_categories.user_id = ?", userID).
		Where("estimate_sub_categories.month = ? AND estimate_sub_categories.year = ?", month, year).
		Joins("LEFT JOIN sub_categories sc ON estimate_sub_categories.sub_category_id = sc.id").
		Select([]string{
			"estimate_sub_categories.*",
			"sc.description as sub_category_name",
		}).
		Find(&estimateSubcategories)
	if err := resultSubCategories.Error; err != nil {
		return []model.EstimateSubCategories{}, err
	}

	return estimateSubcategories, nil
}

func (p PgRepository) Update(ctx context.Context, id int, category model.EstimateCategories, userID string) (model.EstimateCategories, error) {
	//TODO implement me
	panic("implement me")
}
