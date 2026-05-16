package repository

import (
	"context"
	"testing"
	"time"

	"personal-finance/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCouponRedemptionRepository_CreateAndMarkActive(t *testing.T) {
	ctx := context.Background()
	db := setupCouponTestDB(t)
	redRepo := NewCouponRedemptionRepository(db)

	created, err := redRepo.Create(ctx, domain.CouponRedemption{
		UserID:        "user-1",
		CouponID:      "c1",
		PlanID:        "plus_monthly",
		OriginalPrice: 9.90,
		LockedPrice:   6.93,
		Status:        domain.CouponRedemptionPending,
		RedeemedAt:    time.Now(),
	})
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, created.ID)

	subID := uuid.New()
	assert.NoError(t, redRepo.MarkActive(ctx, nil, created.ID, subID))

	found, err := redRepo.FindByID(ctx, created.ID)
	assert.NoError(t, err)
	assert.Equal(t, domain.CouponRedemptionActive, found.Status)
	assert.NotNil(t, found.SubscriptionID)
	assert.Equal(t, subID, *found.SubscriptionID)
}

func TestCouponRedemptionRepository_UniqueUserCoupon(t *testing.T) {
	ctx := context.Background()
	db := setupCouponTestDB(t)
	redRepo := NewCouponRedemptionRepository(db)

	_, err := redRepo.Create(ctx, domain.CouponRedemption{
		UserID: "user-1", CouponID: "c1", PlanID: "p1",
		OriginalPrice: 9.90, LockedPrice: 5.00,
		Status:     domain.CouponRedemptionPending,
		RedeemedAt: time.Now(),
	})
	assert.NoError(t, err)

	_, err = redRepo.Create(ctx, domain.CouponRedemption{
		UserID: "user-1", CouponID: "c1", PlanID: "p1",
		OriginalPrice: 9.90, LockedPrice: 5.00,
		Status:     domain.CouponRedemptionPending,
		RedeemedAt: time.Now(),
	})
	assert.Error(t, err)
}

func TestCouponRedemptionRepository_MarkCancelledBySubscription(t *testing.T) {
	ctx := context.Background()
	db := setupCouponTestDB(t)
	redRepo := NewCouponRedemptionRepository(db)

	created, err := redRepo.Create(ctx, domain.CouponRedemption{
		UserID: "u1", CouponID: "c1", PlanID: "p1",
		OriginalPrice: 9.90, LockedPrice: 5.00,
		Status: domain.CouponRedemptionPending, RedeemedAt: time.Now(),
	})
	assert.NoError(t, err)

	subID := uuid.New()
	assert.NoError(t, redRepo.MarkActive(ctx, nil, created.ID, subID))
	assert.NoError(t, redRepo.MarkCancelledBySubscription(ctx, subID))

	found, _ := redRepo.FindByID(ctx, created.ID)
	assert.Equal(t, domain.CouponRedemptionCancelled, found.Status)
	assert.NotNil(t, found.CancelledAt)
}
