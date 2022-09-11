package categories

import (
	"context"
	"gorm.io/gorm"
	"personal-finance/internal/business/model"
	"time"
)

type Repository interface {
	Add(ctx context.Context, category model.Category) (model.Category, error)
	FindAll(ctx context.Context) ([]model.Category, error)
	FindByID(ctx context.Context, ID int) (model.Category, error)
	Update(ctx context.Context, ID int, category model.Category) (model.Category, error)
	Delete(ctx context.Context, ID int) error
}

type PgRepository struct {
	Gorm *gorm.DB
}

func (p PgRepository) Add(ctx context.Context, category model.Category) (model.Category, error) {
	result := p.Gorm.Create(&category)
	if err := result.Error; err != nil {
		return model.Category{}, err
	}
	return category, nil
}

func (p PgRepository) FindAll(ctx context.Context) ([]model.Category, error) {
	var categories []model.Category
	result := p.Gorm.Find(&categories)
	if err := result.Error; err != nil {
		return []model.Category{}, err
	}
	return categories, nil
}

func (p PgRepository) FindByID(ctx context.Context, id int) (model.Category, error) {
	var category model.Category
	result := p.Gorm.First(&category, id)
	if err := result.Error; err != nil {
		return model.Category{}, err
	}
	return category, nil
}

func (p PgRepository) Update(ctx context.Context, id int, category model.Category) (model.Category, error) {
	cat, err := p.FindByID(context.Background(), id)
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

func (p PgRepository) Delete(ctx context.Context, id int) error {
	if err := p.Gorm.Delete(&model.Category{}, id).Error; err != nil {
		return err
	}
	return nil
}
