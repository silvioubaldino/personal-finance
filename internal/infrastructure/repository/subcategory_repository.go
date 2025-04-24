package repository

import (
	"context"
	"fmt"

	"personal-finance/internal/plataform/authentication"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SubCategoryRepository struct {
	db *gorm.DB
}

func NewSubCategoryRepository(db *gorm.DB) *SubCategoryRepository {
	return &SubCategoryRepository{
		db: db,
	}
}

func (r *SubCategoryRepository) IsSubCategoryBelongsToCategory(ctx context.Context, subCategoryID uuid.UUID, categoryID uuid.UUID) (bool, error) {
	userID := ctx.Value(authentication.UserID).(string)

	var count int64
	err := r.db.Model(&SubCategoryDB{}).
		Where("id = ? AND category_id = ? AND user_id = ?", subCategoryID, categoryID, userID).
		Count(&count).
		Error

	if err != nil {
		return false, fmt.Errorf("error checking if subcategory belongs to category: %w", err)
	}

	return count > 0, nil
}
