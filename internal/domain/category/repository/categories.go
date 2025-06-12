package repository

import (
	"context"
	"time"

	"personal-finance/internal/model"
	"personal-finance/internal/plataform/authentication"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const DefaultIDCategory = "default_category_id"

type Repository interface {
	Add(ctx context.Context, category model.Category) (model.Category, error)
	FindAll(ctx context.Context) ([]model.Category, error)
	FindByID(ctx context.Context, id uuid.UUID) (model.Category, error)
	Update(ctx context.Context, id uuid.UUID, category model.Category) (model.Category, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type PgRepository struct {
	Gorm *gorm.DB
}

func NewPgRepository(gorm *gorm.DB) Repository {
	return PgRepository{Gorm: gorm}
}

func (p PgRepository) Add(ctx context.Context, category model.Category) (model.Category, error) {
	userID := ctx.Value(authentication.UserID).(string)
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

func (p PgRepository) FindAll(ctx context.Context) ([]model.Category, error) {
	userID := ctx.Value(authentication.UserID).(string)
	var categories []model.Category
	result := p.Gorm.Where("categories.user_id IN(?,?)", userID, DefaultIDCategory).
		Preload("SubCategories",
			p.Gorm.Where(`"sub_categories"."user_id" IN(?,?)`, userID, DefaultIDCategory)).
		Order("categories.description").
		Find(&categories)
	if err := result.Error; err != nil {
		return []model.Category{}, err
	}
	return categories, nil
}

func (p PgRepository) FindByID(ctx context.Context, id uuid.UUID) (model.Category, error) {
	userID := ctx.Value(authentication.UserID).(string)
	var category model.Category
	result := p.Gorm.Where("categories.user_id IN(?,?)", userID, DefaultIDCategory).
		Preload("SubCategories",
			p.Gorm.Where(`"sub_categories"."user_id" IN(?,?)`, userID, DefaultIDCategory)).
		First(&category, id)
	if err := result.Error; err != nil {
		return model.Category{}, err
	}
	return category, nil
}

func (p PgRepository) Update(ctx context.Context, id uuid.UUID, category model.Category) (model.Category, error) {
	cat, err := p.FindByID(ctx, id)
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

func (p PgRepository) Delete(_ context.Context, id uuid.UUID) error {
	if err := p.Gorm.Delete(&model.Category{}, id).Error; err != nil {
		return err
	}
	return nil
}
