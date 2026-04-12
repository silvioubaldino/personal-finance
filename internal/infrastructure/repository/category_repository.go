package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/plataform/authentication"

	"github.com/google/uuid"
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

func (r *CategoryRepository) Add(ctx context.Context, category domain.Category) (domain.Category, error) {
	userID := ctx.Value(authentication.UserID).(string)
	now := time.Now()
	id := uuid.New()

	dbModel := FromCategoryDomain(category)
	dbModel.ID = &id
	dbModel.UserID = userID
	dbModel.DateCreate = now
	dbModel.DateUpdate = now

	if err := r.db.WithContext(ctx).Create(&dbModel).Error; err != nil {
		return domain.Category{}, domain.WrapInternalError(err, "error creating category")
	}

	return dbModel.ToDomain(), nil
}

func (r *CategoryRepository) FindAll(ctx context.Context) ([]domain.Category, error) {
	userID := ctx.Value(authentication.UserID).(string)

	var dbModels []CategoryDB
	err := r.db.WithContext(ctx).
		Where("user_id = ? OR user_id = ?", userID, DefaultCategoryUserID).
		Preload("SubCategories", r.db.Where("user_id IN(?,?)", userID, DefaultCategoryUserID)).
		Order("description").
		Find(&dbModels).Error
	if err != nil {
		return nil, fmt.Errorf("error finding categories: %w: %s", ErrDatabaseError, err.Error())
	}

	categories := make([]domain.Category, len(dbModels))
	for i, m := range dbModels {
		cat := m.ToDomain()
		cat.SubCategories = make(domain.SubCategoryList, len(m.SubCategories))
		for j, sub := range m.SubCategories {
			cat.SubCategories[j] = sub.ToDomain()
		}
		categories[i] = cat
	}

	return categories, nil
}

func (r *CategoryRepository) FindByID(ctx context.Context, id uuid.UUID) (domain.Category, error) {
	userID := ctx.Value(authentication.UserID).(string)

	var dbModel CategoryDB
	err := r.db.WithContext(ctx).
		Where("id = ? AND (user_id = ? OR user_id = ?)", id, userID, DefaultCategoryUserID).
		Preload("SubCategories", r.db.Where("user_id IN(?,?)", userID, DefaultCategoryUserID)).
		First(&dbModel).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Category{}, domain.WrapNotFound(ErrCategoryNotFound, "category")
		}
		return domain.Category{}, domain.WrapInternalError(err, "error finding category")
	}

	cat := dbModel.ToDomain()
	cat.SubCategories = make(domain.SubCategoryList, len(dbModel.SubCategories))
	for j, sub := range dbModel.SubCategories {
		cat.SubCategories[j] = sub.ToDomain()
	}
	return cat, nil
}

func (r *CategoryRepository) Update(ctx context.Context, id uuid.UUID, category domain.Category) (domain.Category, error) {
	userID := ctx.Value(authentication.UserID).(string)

	var dbModel CategoryDB
	err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		First(&dbModel).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Category{}, domain.WrapNotFound(ErrCategoryNotFound, "category")
		}
		return domain.Category{}, domain.WrapInternalError(err, "error finding category")
	}

	if category.Description != "" {
		dbModel.Description = category.Description
	}
	if category.Color != "" {
		dbModel.Color = category.Color
	}
	dbModel.DateUpdate = time.Now()

	if err := r.db.WithContext(ctx).Save(&dbModel).Error; err != nil {
		return domain.Category{}, domain.WrapInternalError(err, "error updating category")
	}

	return dbModel.ToDomain(), nil
}

func (r *CategoryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	userID := ctx.Value(authentication.UserID).(string)

	result := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&CategoryDB{})

	if result.Error != nil {
		return domain.WrapInternalError(result.Error, "error deleting category")
	}

	if result.RowsAffected == 0 {
		return domain.WrapNotFound(ErrCategoryNotFound, "category")
	}

	return nil
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
