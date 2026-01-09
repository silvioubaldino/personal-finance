package repository

import (
	"context"
	"fmt"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/plataform/authentication"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EstimateRepository struct {
	db *gorm.DB
}

func NewEstimateRepository(db *gorm.DB) *EstimateRepository {
	return &EstimateRepository{
		db: db,
	}
}

type EstimateCategoryDB struct {
	ID               *string `gorm:"primaryKey;column:id"`
	CategoryID       *string `gorm:"column:category_id"`
	CategoryName     string  `gorm:"column:category_name"`
	IsCategoryIncome bool    `gorm:"column:is_category_income"`
	Month            int     `gorm:"column:month"`
	Year             int     `gorm:"column:year"`
	Amount           float64 `gorm:"column:amount"`
	UserID           string  `gorm:"column:user_id"`
}

func (EstimateCategoryDB) TableName() string {
	return "estimate_categories"
}

type EstimateSubCategoryDB struct {
	ID                 *string `gorm:"primaryKey;column:id"`
	SubCategoryID      *string `gorm:"column:sub_category_id"`
	SubCategoryName    string  `gorm:"column:sub_category_name"`
	EstimateCategoryID *string `gorm:"column:estimate_category_id"`
	Month              int     `gorm:"column:month"`
	Year               int     `gorm:"column:year"`
	Amount             float64 `gorm:"column:amount"`
	UserID             string  `gorm:"column:user_id"`
}

func (EstimateSubCategoryDB) TableName() string {
	return "estimate_sub_categories"
}

func (r *EstimateRepository) FindAllCategoriesByUserID(ctx context.Context) ([]domain.EstimateCategories, error) {
	userID := ctx.Value(authentication.UserID).(string)

	var dbModels []EstimateCategoryDB
	err := r.db.WithContext(ctx).
		Table("estimate_categories").
		Where("estimate_categories.user_id = ?", userID).
		Joins("LEFT JOIN categories c ON estimate_categories.category_id = c.id").
		Select("estimate_categories.*, c.description as category_name, c.is_income as is_category_income").
		Order("year DESC, month DESC").
		Find(&dbModels).Error
	if err != nil {
		return nil, fmt.Errorf("error finding estimate categories: %w: %s", ErrDatabaseError, err.Error())
	}

	result := make([]domain.EstimateCategories, len(dbModels))
	for i, m := range dbModels {
		result[i] = toEstimateCategoryDomain(m)
	}

	return result, nil
}

func (r *EstimateRepository) FindAllSubCategoriesByUserID(ctx context.Context) ([]domain.EstimateSubCategories, error) {
	userID := ctx.Value(authentication.UserID).(string)

	var dbModels []EstimateSubCategoryDB
	err := r.db.WithContext(ctx).
		Table("estimate_sub_categories").
		Where("estimate_sub_categories.user_id = ?", userID).
		Joins("LEFT JOIN sub_categories sc ON estimate_sub_categories.sub_category_id = sc.id").
		Select("estimate_sub_categories.*, sc.description as sub_category_name").
		Order("year DESC, month DESC").
		Find(&dbModels).Error
	if err != nil {
		return nil, fmt.Errorf("error finding estimate sub categories: %w: %s", ErrDatabaseError, err.Error())
	}

	result := make([]domain.EstimateSubCategories, len(dbModels))
	for i, m := range dbModels {
		result[i] = toEstimateSubCategoryDomain(m)
	}

	return result, nil
}

func toEstimateCategoryDomain(m EstimateCategoryDB) domain.EstimateCategories {
	var catID, id *uuid.UUID
	if m.CategoryID != nil {
		parsed, _ := uuid.Parse(*m.CategoryID)
		catID = &parsed
	}
	if m.ID != nil {
		parsed, _ := uuid.Parse(*m.ID)
		id = &parsed
	}
	return domain.EstimateCategories{
		ID:               id,
		CategoryID:       catID,
		CategoryName:     m.CategoryName,
		IsCategoryIncome: m.IsCategoryIncome,
		Month:            time.Month(m.Month),
		Year:             m.Year,
		Amount:           m.Amount,
		UserID:           m.UserID,
	}
}

func toEstimateSubCategoryDomain(m EstimateSubCategoryDB) domain.EstimateSubCategories {
	var subCatID, id, estCatID *uuid.UUID
	if m.SubCategoryID != nil {
		parsed, _ := uuid.Parse(*m.SubCategoryID)
		subCatID = &parsed
	}
	if m.ID != nil {
		parsed, _ := uuid.Parse(*m.ID)
		id = &parsed
	}
	if m.EstimateCategoryID != nil {
		parsed, _ := uuid.Parse(*m.EstimateCategoryID)
		estCatID = &parsed
	}
	return domain.EstimateSubCategories{
		ID:                 id,
		SubCategoryID:      subCatID,
		SubCategoryName:    m.SubCategoryName,
		EstimateCategoryID: estCatID,
		Month:              time.Month(m.Month),
		Year:               m.Year,
		Amount:             m.Amount,
		UserID:             m.UserID,
	}
}

func (r *EstimateRepository) DeleteAllByUserID(ctx context.Context, tx *gorm.DB, userID string) error {
	db := r.db
	if tx != nil {
		db = tx
	}

	err := db.WithContext(ctx).
		Table("estimate_sub_categories").
		Where("user_id = ?", userID).
		Delete(&EstimateSubCategoryDB{}).Error
	if err != nil {
		return fmt.Errorf("error deleting estimate sub categories: %w: %s", ErrDatabaseError, err.Error())
	}

	err = db.WithContext(ctx).
		Table("estimate_categories").
		Where("user_id = ?", userID).
		Delete(&EstimateCategoryDB{}).Error
	if err != nil {
		return fmt.Errorf("error deleting estimate categories: %w: %s", ErrDatabaseError, err.Error())
	}

	return nil
}
