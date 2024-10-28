package repository

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"personal-finance/internal/domain/estimate"
	"personal-finance/internal/domain/subcategory/repository"
	"personal-finance/internal/model"
)

type Repository interface {
	AddEstimate(ctx context.Context, category model.EstimateCategories, userID string) (model.EstimateCategories, error)
	AddSubEstimate(ctx context.Context, subCategory model.EstimateSubCategories, userID string) (model.EstimateSubCategories, error)
	FindCategoriesByMonth(ctx context.Context, month int, year int, userID string) ([]model.EstimateCategories, error)
	FindSubcategoriesByMonth(ctx context.Context, month int, year int, userID string) ([]model.EstimateSubCategories, error)
	Update(ctx context.Context, id int, category model.EstimateCategories, userID string) (model.EstimateCategories, error)
}

type PgRepository struct {
	gorm            *gorm.DB
	subCategoryRepo repository.Repository
}

func NewPgRepository(gorm *gorm.DB, subCategoryRepo repository.Repository) Repository {
	return PgRepository{
		gorm:            gorm,
		subCategoryRepo: subCategoryRepo,
	}
}

func (p PgRepository) FindEstimateByCategoryByMonth(
	_ context.Context,
	categoryID *uuid.UUID,
	month int,
	year int,
	userID string,
) (model.EstimateCategories, error) {
	var estimates model.EstimateCategories
	resultCategories := p.gorm.
		Where("estimate_categories.user_id = ?", userID).
		Where("estimate_categories.month = ? AND estimate_categories.year = ?", month, year).
		Where("estimate_categories.category_id = ?", categoryID).
		First(&estimates)
	if err := resultCategories.Error; err != nil {
		return model.EstimateCategories{}, err
	}
	return estimates, nil
}

func (p PgRepository) AddEstimate(
	ctx context.Context,
	category model.EstimateCategories,
	userID string,
) (model.EstimateCategories, error) {
	cat, err := p.FindEstimateByCategoryByMonth(ctx, category.CategoryID, int(category.Month), category.Year, userID)
	if err == nil && cat.ID != nil {
		log.Printf(estimate.ErrMonthCategoryEstimateExists.Error())
		return cat, estimate.ErrMonthCategoryEstimateExists
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return model.EstimateCategories{}, err
	}

	id := uuid.New()
	category.ID = &id
	category.UserID = userID

	result := p.gorm.
		Select([]string{
			"id",
			"category_id",
			"month",
			"year",
			"amount",
			"user_id",
		}).
		Create(&category)
	if err := result.Error; err != nil {
		return model.EstimateCategories{}, err
	}

	return category, nil
}

func (p PgRepository) FindEstimateByID(estimateID *uuid.UUID) (model.EstimateCategories, error) {
	var category model.EstimateCategories
	result := p.gorm.
		Where("id = ?", estimateID).
		First(&category)
	if err := result.Error; err != nil {
		return model.EstimateCategories{}, err
	}
	return category, nil
}

func (p PgRepository) FindSubEstimatesByEstimateByMonth(
	_ context.Context,
	month int,
	year int,
	estimateID *uuid.UUID,
	userID string,
) ([]model.EstimateSubCategories, error) {
	var subEstimates []model.EstimateSubCategories
	resultSubCategories := p.gorm.
		Where("estimate_sub_categories.user_id = ?", userID).
		Where("estimate_sub_categories.month = ? AND estimate_sub_categories.year = ?", month, year).
		Where("estimate_sub_categories.estimate_category_id = ?", estimateID).
		Find(&subEstimates)
	if err := resultSubCategories.Error; err != nil {
		return []model.EstimateSubCategories{}, err
	}
	return subEstimates, nil
}

func (p PgRepository) FindSubEstimateBySubCategoryByMonth(
	_ context.Context,
	subCategoryID *uuid.UUID,
	month int,
	year int,
	userID string,
) (model.EstimateSubCategories, error) {
	var estimates model.EstimateSubCategories
	resultSubCategories := p.gorm.
		Where("estimate_sub_categories.user_id = ?", userID).
		Where("estimate_sub_categories.month = ? AND estimate_sub_categories.year = ?", month, year).
		Where("estimate_sub_categories.sub_category_id = ?", subCategoryID).
		First(&estimates)
	if err := resultSubCategories.Error; err != nil {
		return model.EstimateSubCategories{}, err
	}
	return estimates, nil
}

func (p PgRepository) ShouldUpdateParentEstimate(
	ctx context.Context,
	subEstimate model.EstimateSubCategories,
	estimate model.EstimateCategories,
	userID string,
) (bool, float64, error) {
	subEstimatesByEstimate, err := p.FindSubEstimatesByEstimateByMonth(
		ctx, int(subEstimate.Month),
		subEstimate.Year,
		subEstimate.EstimateCategoryID,
		userID)
	if err != nil {
		log.Printf(err.Error())
		return false, 0, err
	}

	allSubEstimatesSum := subEstimate.Amount
	for _, sub := range subEstimatesByEstimate {
		allSubEstimatesSum += sub.Amount
	}

	if allSubEstimatesSum > estimate.Amount {
		return true, allSubEstimatesSum, nil
	}
	return false, 0, nil
}

func (p PgRepository) AddSubEstimateConsistent(
	_ context.Context,
	subEstimate model.EstimateSubCategories,
	userID string,
	tx *gorm.DB,
) (model.EstimateSubCategories, error) {
	id := uuid.New()
	subEstimate.ID = &id
	subEstimate.UserID = userID

	result := tx.
		Select([]string{
			"id",
			"sub_category_id",
			"month",
			"year",
			"amount",
			"user_id",
			"estimate_category_id",
		}).
		Create(&subEstimate)
	if err := result.Error; err != nil {
		return model.EstimateSubCategories{}, err
	}

	return subEstimate, nil
}

func (p PgRepository) AddSubEstimate(
	ctx context.Context,
	subEstimate model.EstimateSubCategories,
	userID string,
) (model.EstimateSubCategories, error) {
	estSubCat, err := p.FindSubEstimateBySubCategoryByMonth(
		ctx,
		subEstimate.SubCategoryID,
		int(subEstimate.Month),
		subEstimate.Year,
		userID)
	if err == nil && estSubCat.ID != nil {
		log.Printf(estimate.ErrMonthSubCategoryEstimateExists.Error())
		return estSubCat, estimate.ErrMonthSubCategoryEstimateExists
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return model.EstimateSubCategories{}, err
	}

	estimateCat, err := p.FindEstimateByID(subEstimate.EstimateCategoryID)

	subCategory, err := p.subCategoryRepo.FindByID(ctx, *subEstimate.SubCategoryID, userID)
	if err != nil {
		return model.EstimateSubCategories{}, err
	}

	if *subCategory.CategoryID != *estimateCat.CategoryID {
		return model.EstimateSubCategories{}, estimate.ErrSubCategoryNotInCategory
	}

	gormTransactionErr := p.gorm.Transaction(func(tx *gorm.DB) error {
		should, amount, err := p.ShouldUpdateParentEstimate(ctx, subEstimate, estimateCat, userID)
		if err != nil {
			return err
		}
		if should {
			// update category amount
			log.Printf("update category amount: %f\n", amount)
			return fmt.Errorf("sum of sub categories is greater than category amount, " +
				"automatic estimate category update will be implemented")
		}
		_, err = p.AddSubEstimateConsistent(ctx, subEstimate, userID, tx)
		if err != nil {
			return err
		}
		return nil
	})

	if gormTransactionErr != nil {
		return model.EstimateSubCategories{}, gormTransactionErr
	}

	return subEstimate, nil
}

func (p PgRepository) FindCategoriesByMonth(
	_ context.Context,
	month int,
	year int,
	userID string,
) ([]model.EstimateCategories, error) {
	var estimates []model.EstimateCategories
	resultCategories := p.gorm.
		Where("estimate_categories.user_id = ?", userID).
		Where("estimate_categories.month = ? AND estimate_categories.year = ?", month, year).
		Joins("LEFT JOIN categories c ON estimate_categories.category_id = c.id").
		Select([]string{
			"estimate_categories.*",
			"c.description as category_name",
		}).
		Find(&estimates)
	if err := resultCategories.Error; err != nil {
		return []model.EstimateCategories{}, err
	}
	return estimates, nil
}

func (p PgRepository) FindSubcategoriesByMonth(
	_ context.Context,
	month int,
	year int,
	userID string,
) ([]model.EstimateSubCategories, error) {
	var estimateSubcategories []model.EstimateSubCategories
	resultSubCategories := p.gorm.
		Where("estimate_sub_categories.user_id = ?", userID).
		Where("estimate_sub_categories.month = ? AND estimate_sub_categories.year = ?", month, year).
		Joins("LEFT JOIN sub_categories sc ON estimate_sub_categories.sub_category_id = sc.id").
		Select([]string{
			"estimate_sub_categories.*",
			"sc.description as sub_category_name",
		}).
		Find(&estimateSubcategories)
	if err := resultSubCategories.Error; err != nil {
		return []model.EstimateSubCategories{}, err
	}

	return estimateSubcategories, nil
}

func (p PgRepository) Update(_ context.Context,
	id int,
	category model.EstimateCategories,
	userID string,
) (model.EstimateCategories, error) {
	//TODO implement me
	panic("implement me")
}
