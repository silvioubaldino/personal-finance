package repository

import (
	"context"
	"time"

	"personal-finance/internal/domain/category/repository"
	"personal-finance/internal/model"
	"personal-finance/internal/plataform/authentication"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	FindByID(ctx context.Context, id uuid.UUID) (model.SubCategory, error)
	Add(ctx context.Context, subCategory model.SubCategory) (model.SubCategory, error)
	Update(ctx context.Context, id uuid.UUID, subCategory model.SubCategory) (model.SubCategory, error)
}

type PgRepository struct {
	Gorm *gorm.DB
}

func NewPgRepository(gorm *gorm.DB) Repository {
	return PgRepository{Gorm: gorm}
}

func (p PgRepository) FindByID(ctx context.Context, id uuid.UUID) (model.SubCategory, error) {
	userID := ctx.Value(authentication.UserID).(string)
	var subCategory model.SubCategory
	result := p.Gorm.Where("user_id IN(?,?)", userID, repository.DefaultIDCategory).
		First(&subCategory, id)
	if err := result.Error; err != nil {
		return model.SubCategory{}, err
	}
	return subCategory, nil
}

func (p PgRepository) Add(ctx context.Context, subCategory model.SubCategory) (model.SubCategory, error) {
	userID := ctx.Value(authentication.UserID).(string)
	now := time.Now()
	subCategory.DateCreate = now
	subCategory.DateUpdate = now
	subCategory.UserID = userID
	result := p.Gorm.Create(&subCategory)
	if err := result.Error; err != nil {
		return model.SubCategory{}, err
	}
	return subCategory, nil
}

func (p PgRepository) Update(ctx context.Context, id uuid.UUID, category model.SubCategory) (model.SubCategory, error) {
	subCategory, err := p.FindByID(ctx, id)
	if err != nil {
		return model.SubCategory{}, err
	}
	subCategory.Description = category.Description
	subCategory.DateUpdate = time.Now()
	result := p.Gorm.Save(&subCategory)
	if result.Error != nil {
		return model.SubCategory{}, result.Error
	}
	return subCategory, nil
}
