package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"personal-finance/internal/domain"

	"gorm.io/gorm"
)

type CouponRepository struct {
	db *gorm.DB
}

func NewCouponRepository(db *gorm.DB) *CouponRepository {
	return &CouponRepository{db: db}
}

func (r *CouponRepository) Create(ctx context.Context, coupon domain.Coupon) error {
	now := time.Now()
	if coupon.CreatedAt.IsZero() {
		coupon.CreatedAt = now
	}
	coupon.UpdatedAt = now

	row := FromCouponDomain(coupon)
	err := r.db.WithContext(ctx).Create(&row).Error
	if err != nil {
		return fmt.Errorf("error creating coupon: %w: %s", ErrDatabaseError, err.Error())
	}
	return nil
}

func (r *CouponRepository) Update(ctx context.Context, coupon domain.Coupon) error {
	coupon.UpdatedAt = time.Now()
	row := FromCouponDomain(coupon)

	res := r.db.WithContext(ctx).
		Model(&CouponDB{}).
		Where("id = ?", coupon.ID).
		Updates(map[string]interface{}{
			"description":         row.Description,
			"discount_type":       row.DiscountType,
			"discount_value":      row.DiscountValue,
			"valid_from":          row.ValidFrom,
			"valid_until":         row.ValidUntil,
			"max_redemptions":     row.MaxRedemptions,
			"applicable_plan_ids": row.ApplicablePlanIDs,
			"target_plan_id":      row.TargetPlanID,
			"is_active":           row.IsActive,
			"updated_at":          row.UpdatedAt,
		})
	if res.Error != nil {
		return fmt.Errorf("error updating coupon: %w: %s", ErrDatabaseError, res.Error.Error())
	}
	if res.RowsAffected == 0 {
		return domain.ErrCouponNotFound
	}
	return nil
}

func (r *CouponRepository) FindByID(ctx context.Context, id string) (domain.Coupon, error) {
	var row CouponDB
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Coupon{}, domain.ErrCouponNotFound
		}
		return domain.Coupon{}, fmt.Errorf("error finding coupon: %w: %s", ErrDatabaseError, err.Error())
	}
	return row.ToDomain(), nil
}

func (r *CouponRepository) FindActiveByCode(ctx context.Context, code string) (domain.Coupon, error) {
	var row CouponDB
	err := r.db.WithContext(ctx).
		Where("code = ? AND is_active = ?", code, true).
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Coupon{}, domain.ErrCouponNotFound
		}
		return domain.Coupon{}, fmt.Errorf("error finding coupon: %w: %s", ErrDatabaseError, err.Error())
	}
	return row.ToDomain(), nil
}

type CouponListFilter struct {
	OnlyActive bool
}

func (r *CouponRepository) List(ctx context.Context, filter CouponListFilter) ([]domain.Coupon, error) {
	query := r.db.WithContext(ctx).Model(&CouponDB{})
	if filter.OnlyActive {
		query = query.Where("is_active = ?", true)
	}
	var rows []CouponDB
	err := query.Order("created_at desc").Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("error listing coupons: %w: %s", ErrDatabaseError, err.Error())
	}
	out := make([]domain.Coupon, len(rows))
	for i, row := range rows {
		out[i] = row.ToDomain()
	}
	return out, nil
}

// IncrementRedemptionCount atomically increases redemption_count if the cap has
// not been reached. Returns ErrCouponMaxReached when no row was updated due to
// the cap. Callers should run this inside the same transaction that marks the
// redemption active so the counter and the redemption move together.
func (r *CouponRepository) IncrementRedemptionCount(ctx context.Context, tx *gorm.DB, couponID string) error {
	db := tx
	if db == nil {
		db = r.db
	}

	res := db.WithContext(ctx).
		Model(&CouponDB{}).
		Where("id = ? AND (max_redemptions IS NULL OR redemption_count < max_redemptions)", couponID).
		Updates(map[string]interface{}{
			"redemption_count": gorm.Expr("redemption_count + 1"),
			"updated_at":       time.Now(),
		})
	if res.Error != nil {
		return fmt.Errorf("error incrementing redemption count: %w: %s", ErrDatabaseError, res.Error.Error())
	}
	if res.RowsAffected == 0 {
		return domain.ErrCouponMaxReached
	}
	return nil
}
