package usecase

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/infrastructure/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type (
	CouponRepository interface {
		Create(ctx context.Context, coupon domain.Coupon) error
		Update(ctx context.Context, coupon domain.Coupon) error
		FindByID(ctx context.Context, id string) (domain.Coupon, error)
		FindActiveByCode(ctx context.Context, code string) (domain.Coupon, error)
		List(ctx context.Context, filter repository.CouponListFilter) ([]domain.Coupon, error)
		IncrementRedemptionCount(ctx context.Context, tx *gorm.DB, couponID string) error
	}

	CouponRedemptionRepository interface {
		Create(ctx context.Context, redemption domain.CouponRedemption) (domain.CouponRedemption, error)
		FindByID(ctx context.Context, id uuid.UUID) (domain.CouponRedemption, error)
		FindBySubscription(ctx context.Context, subscriptionID uuid.UUID) (domain.CouponRedemption, error)
		FindByUserCoupon(ctx context.Context, userID, couponID string) (domain.CouponRedemption, error)
		RefreshPending(ctx context.Context, userID, couponID string, originalPrice, lockedPrice float64) (domain.CouponRedemption, error)
		MarkActive(ctx context.Context, tx *gorm.DB, redemptionID, subscriptionID uuid.UUID) error
		MarkCancelledBySubscription(ctx context.Context, subscriptionID uuid.UUID) error
	}
)

type Coupon struct {
	couponRepo     CouponRepository
	redemptionRepo CouponRedemptionRepository
	planRepo       SubscriptionPlanRepository
	db             *gorm.DB
}

func NewCoupon(
	couponRepo CouponRepository,
	redemptionRepo CouponRedemptionRepository,
	planRepo SubscriptionPlanRepository,
	db *gorm.DB,
) *Coupon {
	return &Coupon{
		couponRepo:     couponRepo,
		redemptionRepo: redemptionRepo,
		planRepo:       planRepo,
		db:             db,
	}
}

type CouponUpdateFields struct {
	Description       *string
	DiscountType      *domain.CouponDiscountType
	DiscountValue     *float64
	ValidFrom         *time.Time
	ValidUntil        *time.Time
	MaxRedemptions    *int
	ApplicablePlanIDs []string
	IsActive          *bool
}

type CouponPreview struct {
	Valid           bool    `json:"valid"`
	Reason          string  `json:"reason,omitempty"`
	OriginalPrice   float64 `json:"original_price,omitempty"`
	DiscountedPrice float64 `json:"discounted_price,omitempty"`
	Currency        string  `json:"currency,omitempty"`
}

func (s *Coupon) Preview(ctx context.Context, userID, planID, code string) (CouponPreview, error) {
	plan, err := s.planRepo.FindActiveByID(ctx, planID)
	if err != nil {
		return CouponPreview{Valid: false, Reason: "plan not found"}, nil
	}

	coupon, err := s.couponRepo.FindActiveByCode(ctx, code)
	if err != nil {
		if errors.Is(err, domain.ErrCouponNotFound) {
			return CouponPreview{Valid: false, Reason: "coupon not found"}, nil
		}
		return CouponPreview{}, err
	}

	if reason := s.validateCoupon(ctx, coupon, plan, userID); reason != "" {
		return CouponPreview{Valid: false, Reason: reason, OriginalPrice: plan.Price, Currency: plan.Currency}, nil
	}

	locked, err := computeLockedPrice(coupon, plan.Price)
	if err != nil {
		return CouponPreview{Valid: false, Reason: err.Error(), OriginalPrice: plan.Price, Currency: plan.Currency}, nil
	}

	return CouponPreview{
		Valid:           true,
		OriginalPrice:   plan.Price,
		DiscountedPrice: locked,
		Currency:        plan.Currency,
	}, nil
}

func (s *Coupon) ApplyWebCheckout(ctx context.Context, userID string, plan domain.SubscriptionPlan, code string) (redemptionID uuid.UUID, err error) {
	coupon, err := s.couponRepo.FindActiveByCode(ctx, code)
	if err != nil {
		if errors.Is(err, domain.ErrCouponNotFound) {
			return uuid.Nil, domain.ErrCouponNotFound
		}
		return uuid.Nil, err
	}

	if reason := s.validateCoupon(ctx, coupon, plan, userID); reason != "" {
		return uuid.Nil, mapValidationReason(reason)
	}

	locked, err := computeLockedPrice(coupon, plan.Price)
	if err != nil {
		return uuid.Nil, err
	}

	if existing, err := s.redemptionRepo.FindByUserCoupon(ctx, userID, coupon.ID); err == nil &&
		existing.Status == domain.CouponRedemptionPending {
		refreshed, err := s.redemptionRepo.RefreshPending(ctx, userID, coupon.ID, plan.Price, locked)
		if err != nil {
			return uuid.Nil, err
		}
		return refreshed.ID, nil
	}

	created, err := s.redemptionRepo.Create(ctx, domain.CouponRedemption{
		UserID:        userID,
		CouponID:      coupon.ID,
		PlanID:        plan.ID,
		OriginalPrice: plan.Price,
		LockedPrice:   locked,
		Status:        domain.CouponRedemptionPending,
		RedeemedAt:    time.Now(),
	})
	if err != nil {
		return uuid.Nil, err
	}
	return created.ID, nil
}

func (s *Coupon) ApplyAtCheckout(ctx context.Context, userID string, plan domain.SubscriptionPlan, code string) (lockedPrice float64, redemptionID uuid.UUID, err error) {
	coupon, err := s.couponRepo.FindActiveByCode(ctx, code)
	if err != nil {
		if errors.Is(err, domain.ErrCouponNotFound) {
			return 0, uuid.Nil, domain.ErrCouponNotFound
		}
		return 0, uuid.Nil, err
	}

	if reason := s.validateCoupon(ctx, coupon, plan, userID); reason != "" {
		return 0, uuid.Nil, mapValidationReason(reason)
	}

	locked, err := computeLockedPrice(coupon, plan.Price)
	if err != nil {
		return 0, uuid.Nil, err
	}

	if existing, err := s.redemptionRepo.FindByUserCoupon(ctx, userID, coupon.ID); err == nil &&
		existing.Status == domain.CouponRedemptionPending {
		refreshed, err := s.redemptionRepo.RefreshPending(ctx, userID, coupon.ID, plan.Price, locked)
		if err != nil {
			return 0, uuid.Nil, err
		}
		return refreshed.LockedPrice, refreshed.ID, nil
	}

	created, err := s.redemptionRepo.Create(ctx, domain.CouponRedemption{
		UserID:        userID,
		CouponID:      coupon.ID,
		PlanID:        plan.ID,
		OriginalPrice: plan.Price,
		LockedPrice:   locked,
		Status:        domain.CouponRedemptionPending,
		RedeemedAt:    time.Now(),
	})
	if err != nil {
		return 0, uuid.Nil, err
	}

	return locked, created.ID, nil
}

func (s *Coupon) Confirm(ctx context.Context, redemptionID, subscriptionID uuid.UUID) error {
	redemption, err := s.redemptionRepo.FindByID(ctx, redemptionID)
	if err != nil {
		return err
	}
	if redemption.Status == domain.CouponRedemptionActive {
		return nil
	}
	if redemption.Status != domain.CouponRedemptionPending {
		return nil
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.couponRepo.IncrementRedemptionCount(ctx, tx, redemption.CouponID); err != nil {
			return err
		}
		return s.redemptionRepo.MarkActive(ctx, tx, redemptionID, subscriptionID)
	})
}

func (s *Coupon) MarkCancelledBySubscription(ctx context.Context, subscriptionID uuid.UUID) error {
	return s.redemptionRepo.MarkCancelledBySubscription(ctx, subscriptionID)
}

func (s *Coupon) GetByID(ctx context.Context, id string) (domain.Coupon, error) {
	return s.couponRepo.FindByID(ctx, id)
}

func (s *Coupon) List(ctx context.Context, onlyActive bool) ([]domain.Coupon, error) {
	return s.couponRepo.List(ctx, repository.CouponListFilter{OnlyActive: onlyActive})
}

func (s *Coupon) Create(ctx context.Context, coupon domain.Coupon) error {
	if err := validateCouponInput(coupon); err != nil {
		return err
	}
	return s.couponRepo.Create(ctx, coupon)
}

func (s *Coupon) Update(ctx context.Context, id string, fields CouponUpdateFields) error {
	current, err := s.couponRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if fields.Description != nil {
		current.Description = *fields.Description
	}
	if fields.DiscountType != nil {
		current.DiscountType = *fields.DiscountType
	}
	if fields.DiscountValue != nil {
		current.DiscountValue = *fields.DiscountValue
	}
	if fields.ValidFrom != nil {
		current.ValidFrom = *fields.ValidFrom
	}
	if fields.ValidUntil != nil {
		current.ValidUntil = *fields.ValidUntil
	}
	if fields.MaxRedemptions != nil {
		current.MaxRedemptions = fields.MaxRedemptions
	}
	if fields.ApplicablePlanIDs != nil {
		current.ApplicablePlanIDs = fields.ApplicablePlanIDs
	}
	if fields.IsActive != nil {
		current.IsActive = *fields.IsActive
	}

	if err := validateCouponInput(current); err != nil {
		return err
	}
	return s.couponRepo.Update(ctx, current)
}

func (s *Coupon) Deactivate(ctx context.Context, id string) error {
	current, err := s.couponRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	current.IsActive = false
	return s.couponRepo.Update(ctx, current)
}

func validateCouponInput(c domain.Coupon) error {
	if c.ID == "" {
		return domain.WrapInvalidInput(domain.New("id is required"), "coupon")
	}
	if c.Code == "" {
		return domain.WrapInvalidInput(domain.New("code is required"), "coupon")
	}
	if c.DiscountType != domain.CouponDiscountPercentage && c.DiscountType != domain.CouponDiscountFixedAmount {
		return domain.WrapInvalidInput(domain.New("discount_type must be percentage or fixed_amount"), "coupon")
	}
	if c.DiscountValue <= 0 {
		return domain.WrapInvalidInput(domain.New("discount_value must be positive"), "coupon")
	}
	if c.DiscountType == domain.CouponDiscountPercentage && c.DiscountValue >= 100 {
		return domain.WrapInvalidInput(domain.New("percentage discount must be < 100"), "coupon")
	}
	if c.ValidFrom.IsZero() || c.ValidUntil.IsZero() {
		return domain.WrapInvalidInput(domain.New("valid_from and valid_until are required"), "coupon")
	}
	if !c.ValidUntil.After(c.ValidFrom) {
		return domain.WrapInvalidInput(domain.New("valid_until must be after valid_from"), "coupon")
	}
	if c.MaxRedemptions != nil && *c.MaxRedemptions <= 0 {
		return domain.WrapInvalidInput(domain.New("max_redemptions must be positive"), "coupon")
	}
	return nil
}

func (s *Coupon) validateCoupon(ctx context.Context, coupon domain.Coupon, plan domain.SubscriptionPlan, userID string) string {
	if !coupon.IsActive {
		return "coupon is not active"
	}
	now := time.Now()
	if now.Before(coupon.ValidFrom) || now.After(coupon.ValidUntil) {
		return "coupon outside validity window"
	}
	if coupon.MaxRedemptions != nil && coupon.RedemptionCount >= *coupon.MaxRedemptions {
		return "coupon redemption limit reached"
	}
	if len(coupon.ApplicablePlanIDs) > 0 && !contains(coupon.ApplicablePlanIDs, plan.ID) {
		return "coupon does not apply to this plan"
	}

	if existing, err := s.redemptionRepo.FindByUserCoupon(ctx, userID, coupon.ID); err == nil {
		if existing.Status != domain.CouponRedemptionPending {
			return "coupon already redeemed by this user"
		}
	}
	return ""
}

func mapValidationReason(reason string) error {
	switch reason {
	case "coupon is not active":
		return domain.ErrCouponInactive
	case "coupon outside validity window":
		return domain.ErrCouponExpired
	case "coupon redemption limit reached":
		return domain.ErrCouponMaxReached
	case "coupon does not apply to this plan":
		return domain.ErrCouponPlanNotApplicable
	case "coupon already redeemed by this user":
		return domain.ErrCouponAlreadyRedeemed
	default:
		return fmt.Errorf("%w: %s", domain.ErrInvalidInput, reason)
	}
}

func computeLockedPrice(coupon domain.Coupon, originalPrice float64) (float64, error) {
	var locked float64
	switch coupon.DiscountType {
	case domain.CouponDiscountPercentage:
		locked = originalPrice * (1 - coupon.DiscountValue/100)
	case domain.CouponDiscountFixedAmount:
		locked = originalPrice - coupon.DiscountValue
	default:
		return 0, domain.ErrCouponInvalidPrice
	}
	locked = math.Round(locked*100) / 100
	if locked <= 0 {
		return 0, domain.ErrCouponInvalidPrice
	}
	return locked, nil
}

func contains(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}
