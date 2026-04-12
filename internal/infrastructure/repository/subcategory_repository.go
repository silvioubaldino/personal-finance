package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

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
	userID := ctx.Value(authentication.UserID).(string)
	now := time.Now()
	id := uuid.New()

	dbModel := FromSubCategoryDomain(subcategory)
	dbModel.ID = &id
	dbModel.UserID = userID
	dbModel.DateCreate = now
	dbModel.DateUpdate = now

	if err := r.db.WithContext(ctx).Create(&dbModel).Error; err != nil {
		return domain.SubCategory{}, domain.WrapInternalError(err, "error creating subcategory")
	}

	return dbModel.ToDomain(), nil
}

func (r *SubCategoryRepository) FindAll(ctx context.Context) (domain.SubCategoryList, error) {
	userID := ctx.Value(authentication.UserID).(string)

	var dbModels []SubCategoryDB
	err := r.db.WithContext(ctx).
		Where("user_id = ? OR user_id = ?", userID, repository.DefaultIDCategory).
		Order("description").
		Find(&dbModels).Error
	if err != nil {
		return nil, fmt.Errorf("error finding subcategories: %w", err)
	}

	subCategories := make(domain.SubCategoryList, len(dbModels))
	for i, m := range dbModels {
		subCategories[i] = m.ToDomain()
	}

	return subCategories, nil
}

func (r *SubCategoryRepository) FindByID(ctx context.Context, id uuid.UUID) (domain.SubCategory, error) {
	userID := ctx.Value(authentication.UserID).(string)

	var dbModel SubCategoryDB
	err := r.db.WithContext(ctx).
		Where("id = ? AND user_id IN (?,?)", id, userID, repository.DefaultIDCategory).
		First(&dbModel).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.SubCategory{}, domain.WrapNotFound(ErrSubCategoryNotFound, "subcategory")
		}
		return domain.SubCategory{}, domain.WrapInternalError(err, "error finding subcategory")
	}

	return dbModel.ToDomain(), nil
}

func (r *SubCategoryRepository) FindByCategoryID(ctx context.Context, categoryID uuid.UUID) (domain.SubCategoryList, error) {
	userID := ctx.Value(authentication.UserID).(string)

	var dbModels []SubCategoryDB
	err := r.db.WithContext(ctx).
		Where("category_id = ? AND user_id IN (?,?)", categoryID, userID, repository.DefaultIDCategory).
		Order("description").
		Find(&dbModels).Error
	if err != nil {
		return nil, domain.WrapInternalError(err, "error finding subcategories by category")
	}

	subCategories := make(domain.SubCategoryList, len(dbModels))
	for i, m := range dbModels {
		subCategories[i] = m.ToDomain()
	}

	return subCategories, nil
}

func (r *SubCategoryRepository) Update(ctx context.Context, id uuid.UUID, subcategory domain.SubCategory) (domain.SubCategory, error) {
	userID := ctx.Value(authentication.UserID).(string)

	var dbModel SubCategoryDB
	err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		First(&dbModel).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.SubCategory{}, domain.WrapNotFound(ErrSubCategoryNotFound, "subcategory")
		}
		return domain.SubCategory{}, domain.WrapInternalError(err, "error finding subcategory")
	}

	if subcategory.Description != "" {
		dbModel.Description = subcategory.Description
	}
	dbModel.DateUpdate = time.Now()

	if err := r.db.WithContext(ctx).Save(&dbModel).Error; err != nil {
		return domain.SubCategory{}, domain.WrapInternalError(err, "error updating subcategory")
	}

	return dbModel.ToDomain(), nil
}

func (r *SubCategoryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	userID := ctx.Value(authentication.UserID).(string)

	result := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&SubCategoryDB{})

	if result.Error != nil {
		return domain.WrapInternalError(result.Error, "error deleting subcategory")
	}

	if result.RowsAffected == 0 {
		return domain.WrapNotFound(ErrSubCategoryNotFound, "subcategory")
	}

	return nil
}

func (r *SubCategoryRepository) DeleteAllByUserID(ctx context.Context, tx *gorm.DB, userID string) error {
	db := r.db
	if tx != nil {
		db = tx
	}

	err := db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&SubCategoryDB{}).Error
	if err != nil {
		return fmt.Errorf("error deleting subcategories: %w", err)
	}

	return nil
}
