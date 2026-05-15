package repository

import (
	"context"
	"errors"
	"fmt"

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

func (r *SubscriptionPlanRepository) FindActive(ctx context.Context) ([]domain.SubscriptionPlan, error) {
	var rows []SubscriptionPlanDB
	err := r.db.WithContext(ctx).Where("is_active = true").Order("created_at asc").Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("error listing active plans: %w: %s", ErrDatabaseError, err.Error())
	}
	plans := make([]domain.SubscriptionPlan, len(rows))
	for i, row := range rows {
		plans[i] = row.ToDomain()
	}
	return plans, nil
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
