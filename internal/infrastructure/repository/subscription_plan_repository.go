package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"personal-finance/internal/domain"

	"gorm.io/gorm"
)

var ErrSubscriptionPlanNotFound = errors.New("subscription plan not found")

type SubscriptionPlanRepository struct {
	db *gorm.DB
}

func NewSubscriptionPlanRepository(db *gorm.DB) *SubscriptionPlanRepository {
	return &SubscriptionPlanRepository{db: db}
}

func (r *SubscriptionPlanRepository) Create(ctx context.Context, plan domain.SubscriptionPlan) error {
	var mpPlanID *string
	if plan.MPPreapprovalPlanID != "" {
		id := plan.MPPreapprovalPlanID
		mpPlanID = &id
	}
	row := SubscriptionPlanDB{
		ID:                  plan.ID,
		Name:                plan.Name,
		Price:               plan.Price,
		Currency:            plan.Currency,
		Frequency:           plan.Frequency,
		FrequencyType:       plan.FrequencyType,
		IsActive:            plan.IsActive,
		IsPublic:            plan.IsPublic,
		MPPreapprovalPlanID: mpPlanID,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}
	err := r.db.WithContext(ctx).Create(&row).Error
	if err != nil {
		return fmt.Errorf("error creating plan: %w: %s", ErrDatabaseError, err.Error())
	}
	return nil
}

// FindActive returns only public plans (the storefront). Promotional plans
// (is_public = false) are resolved by id via FindActiveByID, not listed here.
func (r *SubscriptionPlanRepository) FindActive(ctx context.Context) ([]domain.SubscriptionPlan, error) {
	var rows []SubscriptionPlanDB
	err := r.db.WithContext(ctx).Where("is_active = true AND is_public = true").Order("created_at asc").Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("error listing active plans: %w: %s", ErrDatabaseError, err.Error())
	}
	plans := make([]domain.SubscriptionPlan, len(rows))
	for i, row := range rows {
		plans[i] = row.ToDomain()
	}
	return plans, nil
}

// UpdateMPPlanID stores the Mercado Pago preapproval_plan id created for a plan.
func (r *SubscriptionPlanRepository) UpdateMPPlanID(ctx context.Context, planID, mpPlanID string) error {
	res := r.db.WithContext(ctx).
		Model(&SubscriptionPlanDB{}).
		Where("id = ?", planID).
		Updates(map[string]interface{}{
			"mp_preapproval_plan_id": mpPlanID,
			"updated_at":             time.Now(),
		})
	if res.Error != nil {
		return fmt.Errorf("error updating mp plan id: %w: %s", ErrDatabaseError, res.Error.Error())
	}
	if res.RowsAffected == 0 {
		return ErrSubscriptionPlanNotFound
	}
	return nil
}

func (r *SubscriptionPlanRepository) FindActiveByID(ctx context.Context, id string) (domain.SubscriptionPlan, error) {
	var row SubscriptionPlanDB
	err := r.db.WithContext(ctx).Where("id = ? AND is_active = true", id).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.SubscriptionPlan{}, ErrSubscriptionPlanNotFound
		}
		return domain.SubscriptionPlan{}, fmt.Errorf("error finding plan %s: %w: %s", id, ErrDatabaseError, err.Error())
	}
	return row.ToDomain(), nil
}

func (r *SubscriptionPlanRepository) FindIDByStoreProduct(ctx context.Context, store, productID string) (string, error) {
	if productID == "" {
		return "", nil
	}
	var column string
	switch store {
	case "APP_STORE":
		column = "apple_product_id"
	case "PLAY_STORE":
		column = "google_product_id"
	default:
		return "", nil
	}
	var row SubscriptionPlanDB
	err := r.db.WithContext(ctx).
		Where(column+" = ?", productID).
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", fmt.Errorf("error finding plan by %s=%s: %w: %s", column, productID, ErrDatabaseError, err.Error())
	}
	return row.ID, nil
}
