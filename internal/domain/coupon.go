package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type CouponDiscountType string

const (
	CouponDiscountPercentage  CouponDiscountType = "percentage"
	CouponDiscountFixedAmount CouponDiscountType = "fixed_amount"
)

type CouponRedemptionStatus string

const (
	CouponRedemptionPending   CouponRedemptionStatus = "pending"
	CouponRedemptionActive    CouponRedemptionStatus = "active"
	CouponRedemptionCancelled CouponRedemptionStatus = "cancelled"
)

type Coupon struct {
	ID                string
	Code              string
	Description       string
	DiscountType      CouponDiscountType
	DiscountValue     float64
	ValidFrom         time.Time
	ValidUntil        time.Time
	MaxRedemptions    *int
	RedemptionCount   int
	ApplicablePlanIDs []string
	IsActive          bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type CouponRedemption struct {
	ID             uuid.UUID
	UserID         string
	CouponID       string
	PlanID         string
	SubscriptionID *uuid.UUID
	OriginalPrice  float64
	LockedPrice    float64
	Status         CouponRedemptionStatus
	RedeemedAt     time.Time
	CancelledAt    *time.Time
}

var (
	ErrCouponNotFound          = errors.New("coupon not found")
	ErrCouponInactive          = errors.New("coupon is not active")
	ErrCouponExpired           = errors.New("coupon outside validity window")
	ErrCouponMaxReached        = errors.New("coupon redemption limit reached")
	ErrCouponAlreadyRedeemed   = errors.New("user already redeemed this coupon")
	ErrCouponPlanNotApplicable = errors.New("coupon does not apply to this plan")
	ErrCouponInvalidPrice      = errors.New("coupon would result in non-positive price")
	ErrCouponRedemptionMissing = errors.New("coupon redemption not found")
)
