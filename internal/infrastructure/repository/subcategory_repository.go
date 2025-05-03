package repository

import (
	"context"
	"errors"
	"fmt"

	"personal-finance/internal/domain"
	"personal-finance/internal/domain/category/repository"
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
	err := r.db.WithContext(ctx).
		Model(&SubCategoryDB{}).
		Where("id = ? AND category_id = ? AND user_id IN (?,?)", subCategoryID, categoryID, userID, repository.DefaultIDCategory).
		Count(&count).
		Error
	if err != nil {
		return false, fmt.Errorf("error checking if subcategory belongs to category: %w", err)
	}

	return count > 0, nil
}

func (r *SubCategoryRepository) Add(ctx context.Context, subcategory domain.SubCategory) (domain.SubCategory, error) {
	return domain.SubCategory{}, errors.New("method Add not implemented")
}

func (r *SubCategoryRepository) FindAll(ctx context.Context) (domain.SubCategoryList, error) {
	return domain.SubCategoryList{}, errors.New("method FindAll not implemented")
}

func (r *SubCategoryRepository) FindByID(ctx context.Context, ID uuid.UUID) (domain.SubCategory, error) {
	return domain.SubCategory{}, errors.New("method FindByID not implemented")
}

func (r *SubCategoryRepository) FindByCategoryID(ctx context.Context, categoryID uuid.UUID) (domain.SubCategoryList, error) {
	return domain.SubCategoryList{}, errors.New("method FindByCategoryID not implemented")
}

func (r *SubCategoryRepository) Update(ctx context.Context, subcategory domain.SubCategory) (domain.SubCategory, error) {
	return domain.SubCategory{}, errors.New("method Update not implemented")
}

func (r *SubCategoryRepository) Delete(ctx context.Context, ID uuid.UUID) error {
	return errors.New("method Delete not implemented")
}
