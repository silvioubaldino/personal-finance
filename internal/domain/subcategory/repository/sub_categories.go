package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"personal-finance/internal/model"
)

type Repository interface {
	FindByID(ctx context.Context, id int, userID string) (model.SubCategory, error)
	Add(ctx context.Context, subCategory model.SubCategory, userID string) (model.SubCategory, error)
	Update(ctx context.Context, id int, subCategory model.SubCategory, userID string) (model.SubCategory, error)
}

type PgRepository struct {
	Gorm *gorm.DB
}

func NewPgRepository(gorm *gorm.DB) Repository {
	return PgRepository{Gorm: gorm}
}

func (p PgRepository) FindByID(_ context.Context, id int, userID string) (model.SubCategory, error) {
	var subCategory model.SubCategory
	result := p.Gorm.Where("user_id=?", userID).First(&subCategory, id)
	if err := result.Error; err != nil {
		return model.SubCategory{}, err
	}
	return subCategory, nil
}

func (p PgRepository) Add(_ context.Context, subCategory model.SubCategory, userID string) (model.SubCategory, error) {
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

func (p PgRepository) Update(_ context.Context, id int, category model.SubCategory, userID string) (model.SubCategory, error) {
	subCategory, err := p.FindByID(context.Background(), id, userID)
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
