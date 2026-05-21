package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"personal-finance/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CouponRedemptionRepository struct {
	db *gorm.DB
}

func NewCouponRedemptionRepository(db *gorm.DB) *CouponRedemptionRepository {
	return &CouponRedemptionRepository{db: db}
}

func (r *CouponRedemptionRepository) DB() *gorm.DB {
	return r.db
}

func (r *CouponRedemptionRepository) Create(ctx context.Context, redemption domain.CouponRedemption) (domain.CouponRedemption, error) {
	if redemption.ID == uuid.Nil {
		redemption.ID = uuid.New()
	}
	if redemption.RedeemedAt.IsZero() {
		redemption.RedeemedAt = time.Now()
	}
	if redemption.Status == "" {
		redemption.Status = domain.CouponRedemptionPending
	}

	row := FromCouponRedemptionDomain(redemption)
	err := r.db.WithContext(ctx).Create(&row).Error
	if err != nil {
		return domain.CouponRedemption{}, fmt.Errorf("error creating coupon redemption: %w: %s", ErrDatabaseError, err.Error())
	}
	return row.ToDomain(), nil
}

func (r *CouponRedemptionRepository) FindByID(ctx context.Context, id uuid.UUID) (domain.CouponRedemption, error) {
	var row CouponRedemptionDB
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.CouponRedemption{}, domain.ErrCouponRedemptionMissing
		}
		return domain.CouponRedemption{}, fmt.Errorf("error finding redemption: %w: %s", ErrDatabaseError, err.Error())
	}
	return row.ToDomain(), nil
}

func (r *CouponRedemptionRepository) FindBySubscription(ctx context.Context, subscriptionID uuid.UUID) (domain.CouponRedemption, error) {
	var row CouponRedemptionDB
	err := r.db.WithContext(ctx).Where("subscription_id = ?", subscriptionID).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.CouponRedemption{}, domain.ErrCouponRedemptionMissing
		}
		return domain.CouponRedemption{}, fmt.Errorf("error finding redemption: %w: %s", ErrDatabaseError, err.Error())
	}
	return row.ToDomain(), nil
}

func (r *CouponRedemptionRepository) FindByUserCoupon(ctx context.Context, userID, couponID string) (domain.CouponRedemption, error) {
	var row CouponRedemptionDB
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND coupon_id = ?", userID, couponID).
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.CouponRedemption{}, domain.ErrCouponRedemptionMissing
		}
		return domain.CouponRedemption{}, fmt.Errorf("error finding redemption: %w: %s", ErrDatabaseError, err.Error())
	}
	return row.ToDomain(), nil
}

// MarkActive flips a pending redemption to active and links the subscription.
// Idempotent: re-calling on an already-active redemption with the same
// subscription_id is a no-op.
func (r *CouponRedemptionRepository) MarkActive(ctx context.Context, tx *gorm.DB, redemptionID, subscriptionID uuid.UUID) error {
	db := tx
	if db == nil {
		db = r.db
	}
	res := db.WithContext(ctx).
		Model(&CouponRedemptionDB{}).
		Where("id = ? AND status = ?", redemptionID, string(domain.CouponRedemptionPending)).
		Updates(map[string]interface{}{
			"status":          string(domain.CouponRedemptionActive),
			"subscription_id": subscriptionID,
		})
	if res.Error != nil {
		return fmt.Errorf("error marking redemption active: %w: %s", ErrDatabaseError, res.Error.Error())
	}
	return nil
}

func (r *CouponRedemptionRepository) MarkCancelledBySubscription(ctx context.Context, subscriptionID uuid.UUID) error {
	now := time.Now()
	res := r.db.WithContext(ctx).
		Model(&CouponRedemptionDB{}).
		Where("subscription_id = ? AND status = ?", subscriptionID, string(domain.CouponRedemptionActive)).
		Updates(map[string]interface{}{
			"status":       string(domain.CouponRedemptionCancelled),
			"cancelled_at": now,
		})
	if res.Error != nil {
		return fmt.Errorf("error cancelling redemption: %w: %s", ErrDatabaseError, res.Error.Error())
	}
	return nil
}
