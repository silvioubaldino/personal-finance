package repository

import (
	"context"
	"fmt"

	"personal-finance/internal/domain"
	"personal-finance/internal/plataform/authentication"

	"gorm.io/gorm"
)

const DefaultCategoryUserID = "default_category_id"

type CategoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) *CategoryRepository {
	return &CategoryRepository{
		db: db,
	}
}

func (r *CategoryRepository) FindAll(ctx context.Context) ([]domain.Category, error) {
	userID := ctx.Value(authentication.UserID).(string)

	var dbModels []CategoryDB
	err := r.db.WithContext(ctx).
		Where("user_id = ? OR user_id = ?", userID, DefaultCategoryUserID).
		Order("description").
		Find(&dbModels).Error
	if err != nil {
		return nil, fmt.Errorf("error finding categories: %w: %s", ErrDatabaseError, err.Error())
	}

	categories := make([]domain.Category, len(dbModels))
	for i, m := range dbModels {
		categories[i] = m.ToDomain()
	}

	return categories, nil
}

func (r *CategoryRepository) DeleteAllByUserID(ctx context.Context, tx *gorm.DB, userID string) error {
	db := r.db
	if tx != nil {
		db = tx
	}

	err := db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&CategoryDB{}).Error
	if err != nil {
		return fmt.Errorf("error deleting categories: %w: %s", ErrDatabaseError, err.Error())
	}

	return nil
}
