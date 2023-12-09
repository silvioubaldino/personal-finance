package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"personal-finance/internal/model"
)

type Repository interface {
	Add(ctx context.Context, category model.Category, userID string) (model.Category, error)
	FindAll(ctx context.Context, userID string) ([]model.Category, error)
	FindByID(ctx context.Context, id int, userID string) (model.Category, error)
	Update(ctx context.Context, id int, category model.Category, userID string) (model.Category, error)
	Delete(ctx context.Context, id int) error
}

type PgRepository struct {
	Gorm *gorm.DB
}

func NewPgRepository(gorm *gorm.DB) Repository {
	return PgRepository{Gorm: gorm}
}

func (p PgRepository) Add(_ context.Context, category model.Category, userID string) (model.Category, error) {
	now := time.Now()
	category.DateCreate = now
	category.DateUpdate = now
	category.UserID = userID
	result := p.Gorm.Create(&category)
	if err := result.Error; err != nil {
		return model.Category{}, err
	}
	return category, nil
}

func (p PgRepository) FindAll(_ context.Context, userID string) ([]model.Category, error) {
	var categories []model.Category
	result := p.Gorm.Where("categories.user_id=?", userID).
		Preload("SubCategories").
		Find(&categories)
	if err := result.Error; err != nil {
		return []model.Category{}, err
	}
	return categories, nil
}

func (p PgRepository) FindByID(_ context.Context, id int, userID string) (model.Category, error) {
	var category model.Category
	result := p.Gorm.Where("user_id=?", userID).
		Preload("SubCategories").
		First(&category, id)
	if err := result.Error; err != nil {
		return model.Category{}, err
	}
	return category, nil
}

func (p PgRepository) Update(_ context.Context, id int, category model.Category, userID string) (model.Category, error) {
	cat, err := p.FindByID(context.Background(), id, userID)
	if err != nil {
		return model.Category{}, err
	}
	cat.Description = category.Description
	cat.DateUpdate = time.Now()
	result := p.Gorm.Save(&cat)
	if result.Error != nil {
		return model.Category{}, result.Error
	}
	return cat, nil
}

func (p PgRepository) Delete(_ context.Context, id int) error {
	if err := p.Gorm.Delete(&model.Category{}, id).Error; err != nil {
		return err
	}
	return nil
}
