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

func (r *EstimateRepository) FindCategoriesByMonth(ctx context.Context, month int, year int) ([]domain.EstimateCategories, error) {
	userID := ctx.Value(authentication.UserID).(string)

	var dbModels []EstimateCategoryDB
	err := r.db.WithContext(ctx).
		Table("estimate_categories").
		Where("estimate_categories.user_id = ?", userID).
		Where("estimate_categories.month = ? AND estimate_categories.year = ?", month, year).
		Joins("LEFT JOIN categories c ON estimate_categories.category_id = c.id").
		Select("estimate_categories.*, c.description as category_name, c.is_income as is_category_income").
		Order("c.description").
		Find(&dbModels).Error
	if err != nil {
		return nil, fmt.Errorf("error finding estimate categories by month: %w", err)
	}

	result := make([]domain.EstimateCategories, len(dbModels))
	for i, m := range dbModels {
		result[i] = toEstimateCategoryDomain(m)
	}
	return result, nil
}

func (r *EstimateRepository) FindSubcategoriesByMonth(ctx context.Context, month int, year int) ([]domain.EstimateSubCategories, error) {
	userID := ctx.Value(authentication.UserID).(string)

	var dbModels []EstimateSubCategoryDB
	err := r.db.WithContext(ctx).
		Table("estimate_sub_categories").
		Where("estimate_sub_categories.user_id = ?", userID).
		Where("estimate_sub_categories.month = ? AND estimate_sub_categories.year = ?", month, year).
		Joins("LEFT JOIN sub_categories sc ON estimate_sub_categories.sub_category_id = sc.id").
		Select("estimate_sub_categories.*, sc.description as sub_category_name").
		Order("sc.description").
		Find(&dbModels).Error
	if err != nil {
		return nil, fmt.Errorf("error finding estimate sub categories by month: %w", err)
	}

	result := make([]domain.EstimateSubCategories, len(dbModels))
	for i, m := range dbModels {
		result[i] = toEstimateSubCategoryDomain(m)
	}
	return result, nil
}

func (r *EstimateRepository) AddEstimateCategory(ctx context.Context, category domain.EstimateCategories) (domain.EstimateCategories, error) {
	userID := ctx.Value(authentication.UserID).(string)
	id := uuid.New()
	idStr := id.String()

	var catIDStr *string
	if category.CategoryID != nil {
		s := category.CategoryID.String()
		catIDStr = &s
	}

	dbModel := EstimateCategoryDB{
		ID:         &idStr,
		CategoryID: catIDStr,
		Month:      int(category.Month),
		Year:       category.Year,
		Amount:     category.Amount,
		UserID:     userID,
	}

	result := r.db.WithContext(ctx).
		Select([]string{"id", "category_id", "month", "year", "amount", "user_id"}).
		Create(&dbModel)
	if err := result.Error; err != nil {
		return domain.EstimateCategories{}, fmt.Errorf("error creating estimate category: %w", err)
	}

	return toEstimateCategoryDomain(dbModel), nil
}

func (r *EstimateRepository) AddEstimateSubCategory(ctx context.Context, subEstimate domain.EstimateSubCategories) (domain.EstimateSubCategories, error) {
	userID := ctx.Value(authentication.UserID).(string)
	id := uuid.New()
	idStr := id.String()

	var subCatIDStr, estCatIDStr *string
	if subEstimate.SubCategoryID != nil {
		s := subEstimate.SubCategoryID.String()
		subCatIDStr = &s
	}
	if subEstimate.EstimateCategoryID != nil {
		s := subEstimate.EstimateCategoryID.String()
		estCatIDStr = &s
	}

	dbModel := EstimateSubCategoryDB{
		ID:                 &idStr,
		SubCategoryID:      subCatIDStr,
		EstimateCategoryID: estCatIDStr,
		Month:              int(subEstimate.Month),
		Year:               subEstimate.Year,
		Amount:             subEstimate.Amount,
		UserID:             userID,
	}

	result := r.db.WithContext(ctx).
		Select([]string{"id", "sub_category_id", "estimate_category_id", "month", "year", "amount", "user_id"}).
		Create(&dbModel)
	if err := result.Error; err != nil {
		return domain.EstimateSubCategories{}, fmt.Errorf("error creating estimate sub category: %w", err)
	}

	return toEstimateSubCategoryDomain(dbModel), nil
}

func (r *EstimateRepository) UpdateEstimateCategoryAmount(ctx context.Context, id *uuid.UUID, amount float64) (domain.EstimateCategories, error) {
	userID := ctx.Value(authentication.UserID).(string)
	idStr := id.String()

	result := r.db.WithContext(ctx).
		Model(&EstimateCategoryDB{}).
		Where("id = ? AND user_id = ?", idStr, userID).
		Update("amount", amount)
	if err := result.Error; err != nil {
		return domain.EstimateCategories{}, fmt.Errorf("error updating estimate category amount: %w", err)
	}

	var dbModel EstimateCategoryDB
	err := r.db.WithContext(ctx).
		Table("estimate_categories").
		Where("estimate_categories.id = ?", idStr).
		Joins("LEFT JOIN categories c ON estimate_categories.category_id = c.id").
		Select("estimate_categories.*, c.description as category_name, c.is_income as is_category_income").
		First(&dbModel).Error
	if err != nil {
		return domain.EstimateCategories{}, fmt.Errorf("error fetching updated estimate category: %w", err)
	}

	return toEstimateCategoryDomain(dbModel), nil
}

func (r *EstimateRepository) UpdateEstimateSubCategoryAmount(ctx context.Context, id *uuid.UUID, amount float64) (domain.EstimateSubCategories, error) {
	userID := ctx.Value(authentication.UserID).(string)
	idStr := id.String()

	result := r.db.WithContext(ctx).
		Model(&EstimateSubCategoryDB{}).
		Where("id = ? AND user_id = ?", idStr, userID).
		Update("amount", amount)
	if err := result.Error; err != nil {
		return domain.EstimateSubCategories{}, fmt.Errorf("error updating estimate sub category amount: %w", err)
	}

	var dbModel EstimateSubCategoryDB
	err := r.db.WithContext(ctx).
		Table("estimate_sub_categories").
		Where("estimate_sub_categories.id = ?", idStr).
		Joins("LEFT JOIN sub_categories sc ON estimate_sub_categories.sub_category_id = sc.id").
		Select("estimate_sub_categories.*, sc.description as sub_category_name").
		First(&dbModel).Error
	if err != nil {
		return domain.EstimateSubCategories{}, fmt.Errorf("error fetching updated estimate sub category: %w", err)
	}

	return toEstimateSubCategoryDomain(dbModel), nil
}

func (r *EstimateRepository) DeleteEstimateCategory(ctx context.Context, id *uuid.UUID) error {
	userID := ctx.Value(authentication.UserID).(string)
	idStr := id.String()

	result := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", idStr, userID).
		Delete(&EstimateCategoryDB{})

	if result.Error != nil {
		return fmt.Errorf("error deleting estimate category: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("estimate category not found")
	}

	return nil
}

func (r *EstimateRepository) DeleteEstimateSubCategory(ctx context.Context, id *uuid.UUID) error {
	userID := ctx.Value(authentication.UserID).(string)
	idStr := id.String()

	result := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", idStr, userID).
		Delete(&EstimateSubCategoryDB{})

	if result.Error != nil {
		return fmt.Errorf("error deleting estimate sub category: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("estimate sub category not found")
	}

	return nil
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
