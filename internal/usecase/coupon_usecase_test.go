package usecase

import (
	"context"
	"testing"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/infrastructure/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newCouponTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	assert.NoError(t, db.AutoMigrate(&repository.CouponDB{}, &repository.CouponRedemptionDB{}))
	return db
}

func newCouponUseCase(t *testing.T) (*Coupon, *repository.CouponRepository, *repository.CouponRedemptionRepository, *fakePlanRepo) {
	t.Helper()
	db := newCouponTestDB(t)
	couponRepo := repository.NewCouponRepository(db)
	redRepo := repository.NewCouponRedemptionRepository(db)
	planRepo := &fakePlanRepo{plans: map[string]domain.SubscriptionPlan{
		"plus_monthly": {ID: "plus_monthly", Name: "Plus Mensal", Price: 10.00, Currency: "BRL", IsActive: true},
	}}
	return NewCoupon(couponRepo, redRepo, planRepo, db), couponRepo, redRepo, planRepo
}

type fakePlanRepo struct {
	plans map[string]domain.SubscriptionPlan
}

func (f *fakePlanRepo) Create(_ context.Context, plan domain.SubscriptionPlan) error {
	f.plans[plan.ID] = plan
	return nil
}

func (f *fakePlanRepo) FindActive(_ context.Context) ([]domain.SubscriptionPlan, error) {
	out := make([]domain.SubscriptionPlan, 0, len(f.plans))
	for _, p := range f.plans {
		out = append(out, p)
	}
	return out, nil
}

func (f *fakePlanRepo) FindActiveByID(_ context.Context, id string) (domain.SubscriptionPlan, error) {
	if p, ok := f.plans[id]; ok {
		return p, nil
	}
	return domain.SubscriptionPlan{}, repository.ErrSubscriptionPlanNotFound
}

func (f *fakePlanRepo) FindIDByStoreProduct(_ context.Context, _, _ string) (string, error) {
	return "", nil
}

func validCoupon(id, code string) domain.Coupon {
	now := time.Now().UTC()
	return domain.Coupon{
		ID:            id,
		Code:          code,
		DiscountType:  domain.CouponDiscountPercentage,
		DiscountValue: 30,
		ValidFrom:     now.Add(-time.Hour),
		ValidUntil:    now.Add(24 * time.Hour),
		IsActive:      true,
	}
}

func TestCoupon_Preview_HappyPath(t *testing.T) {
	uc, couponRepo, _, _ := newCouponUseCase(t)
	assert.NoError(t, couponRepo.Create(context.Background(), validCoupon("c1", "PROMO")))

	preview, err := uc.Preview(context.Background(), "user-1", "plus_monthly", "PROMO")
	assert.NoError(t, err)
	assert.True(t, preview.Valid)
	assert.Equal(t, 10.0, preview.OriginalPrice)
	assert.Equal(t, 7.0, preview.DiscountedPrice)
}

func TestCoupon_Preview_RejectsZeroPrice(t *testing.T) {
	uc, couponRepo, _, _ := newCouponUseCase(t)
	c := validCoupon("c1", "FULL")
	c.DiscountType = domain.CouponDiscountFixedAmount
	c.DiscountValue = 20 // > plan price of 10
	assert.NoError(t, couponRepo.Create(context.Background(), c))

	preview, err := uc.Preview(context.Background(), "user-1", "plus_monthly", "FULL")
	assert.NoError(t, err)
	assert.False(t, preview.Valid)
}

func TestCoupon_Preview_OutsideWindow(t *testing.T) {
	uc, couponRepo, _, _ := newCouponUseCase(t)
	c := validCoupon("c1", "EXPIRED")
	c.ValidUntil = time.Now().Add(-time.Hour)
	c.ValidFrom = c.ValidUntil.Add(-time.Hour)
	assert.NoError(t, couponRepo.Create(context.Background(), c))

	preview, err := uc.Preview(context.Background(), "user-1", "plus_monthly", "EXPIRED")
	assert.NoError(t, err)
	assert.False(t, preview.Valid)
}

func TestCoupon_Preview_PlanNotApplicable(t *testing.T) {
	uc, couponRepo, _, _ := newCouponUseCase(t)
	c := validCoupon("c1", "ANNUALONLY")
	c.ApplicablePlanIDs = []string{"plus_annual"}
	assert.NoError(t, couponRepo.Create(context.Background(), c))

	preview, err := uc.Preview(context.Background(), "user-1", "plus_monthly", "ANNUALONLY")
	assert.NoError(t, err)
	assert.False(t, preview.Valid)
}

func TestCoupon_ApplyAtCheckout_CreatesPendingRedemption(t *testing.T) {
	uc, couponRepo, redRepo, planRepo := newCouponUseCase(t)
	assert.NoError(t, couponRepo.Create(context.Background(), validCoupon("c1", "PROMO")))
	plan, _ := planRepo.FindActiveByID(context.Background(), "plus_monthly")

	locked, redemptionID, err := uc.ApplyAtCheckout(context.Background(), "user-1", plan, "PROMO")
	assert.NoError(t, err)
	assert.Equal(t, 7.0, locked)
	assert.NotEqual(t, uuid.Nil, redemptionID)

	red, err := redRepo.FindByID(context.Background(), redemptionID)
	assert.NoError(t, err)
	assert.Equal(t, domain.CouponRedemptionPending, red.Status)
	assert.Equal(t, "user-1", red.UserID)
	assert.Equal(t, 10.0, red.OriginalPrice)
}

func TestCoupon_ApplyAtCheckout_RefreshesPendingOnRetry(t *testing.T) {
	uc, couponRepo, redRepo, planRepo := newCouponUseCase(t)
	assert.NoError(t, couponRepo.Create(context.Background(), validCoupon("c1", "PROMO")))
	plan, _ := planRepo.FindActiveByID(context.Background(), "plus_monthly")

	_, firstID, err := uc.ApplyAtCheckout(context.Background(), "user-1", plan, "PROMO")
	assert.NoError(t, err)

	// Second apply on abandoned PENDING must succeed and reuse the same redemption ID.
	locked, secondID, err := uc.ApplyAtCheckout(context.Background(), "user-1", plan, "PROMO")
	assert.NoError(t, err)
	assert.Equal(t, firstID, secondID, "should reuse the same redemption record")
	assert.Equal(t, 7.0, locked)

	red, err := redRepo.FindByID(context.Background(), secondID)
	assert.NoError(t, err)
	assert.Equal(t, domain.CouponRedemptionPending, red.Status)
}

func TestCoupon_ApplyAtCheckout_BlocksWhenAlreadyActive(t *testing.T) {
	uc, couponRepo, _, planRepo := newCouponUseCase(t)
	assert.NoError(t, couponRepo.Create(context.Background(), validCoupon("c1", "PROMO")))
	plan, _ := planRepo.FindActiveByID(context.Background(), "plus_monthly")

	_, redemptionID, err := uc.ApplyAtCheckout(context.Background(), "user-1", plan, "PROMO")
	assert.NoError(t, err)

	// Simulate payment confirmed.
	assert.NoError(t, uc.Confirm(context.Background(), redemptionID, uuid.New()))

	_, _, err = uc.ApplyAtCheckout(context.Background(), "user-1", plan, "PROMO")
	assert.ErrorIs(t, err, domain.ErrCouponAlreadyRedeemed)
}

func TestCoupon_Confirm_IsIdempotent(t *testing.T) {
	uc, couponRepo, redRepo, planRepo := newCouponUseCase(t)
	assert.NoError(t, couponRepo.Create(context.Background(), validCoupon("c1", "PROMO")))
	plan, _ := planRepo.FindActiveByID(context.Background(), "plus_monthly")

	_, redemptionID, err := uc.ApplyAtCheckout(context.Background(), "user-1", plan, "PROMO")
	assert.NoError(t, err)

	subID := uuid.New()
	assert.NoError(t, uc.Confirm(context.Background(), redemptionID, subID))
	// Second call must be a no-op — counter still 1, status still active.
	assert.NoError(t, uc.Confirm(context.Background(), redemptionID, subID))

	red, _ := redRepo.FindByID(context.Background(), redemptionID)
	assert.Equal(t, domain.CouponRedemptionActive, red.Status)

	coupon, _ := couponRepo.FindByID(context.Background(), "c1")
	assert.Equal(t, 1, coupon.RedemptionCount)
}

func TestCoupon_Confirm_CancelFlowMarksRedemption(t *testing.T) {
	uc, couponRepo, redRepo, planRepo := newCouponUseCase(t)
	assert.NoError(t, couponRepo.Create(context.Background(), validCoupon("c1", "PROMO")))
	plan, _ := planRepo.FindActiveByID(context.Background(), "plus_monthly")

	_, redemptionID, _ := uc.ApplyAtCheckout(context.Background(), "user-1", plan, "PROMO")
	subID := uuid.New()
	assert.NoError(t, uc.Confirm(context.Background(), redemptionID, subID))

	assert.NoError(t, uc.MarkCancelledBySubscription(context.Background(), subID))

	red, _ := redRepo.FindByID(context.Background(), redemptionID)
	assert.Equal(t, domain.CouponRedemptionCancelled, red.Status)
}
