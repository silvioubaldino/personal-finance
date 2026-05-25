package repository

import (
	"context"
	"testing"
	"time"

	"personal-finance/internal/domain"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupCouponTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	assert.NoError(t, db.AutoMigrate(&CouponDB{}, &CouponRedemptionDB{}))
	return db
}

func sampleCoupon(id, code string, max *int) domain.Coupon {
	now := time.Now().UTC()
	return domain.Coupon{
		ID:             id,
		Code:           code,
		DiscountType:   domain.CouponDiscountPercentage,
		DiscountValue:  30,
		ValidFrom:      now.Add(-time.Hour),
		ValidUntil:     now.Add(24 * time.Hour),
		MaxRedemptions: max,
		IsActive:       true,
	}
}

func TestCouponRepository_CreateAndFind(t *testing.T) {
	ctx := context.Background()
	repo := NewCouponRepository(setupCouponTestDB(t))

	c := sampleCoupon("c1", "BLACKFRIDAY", nil)
	c.ApplicablePlanIDs = []string{"plus_monthly"}
	assert.NoError(t, repo.Create(ctx, c))

	found, err := repo.FindActiveByCode(ctx, "BLACKFRIDAY")
	assert.NoError(t, err)
	assert.Equal(t, "c1", found.ID)
	assert.Equal(t, []string{"plus_monthly"}, found.ApplicablePlanIDs)

	_, err = repo.FindActiveByCode(ctx, "MISSING")
	assert.ErrorIs(t, err, domain.ErrCouponNotFound)
}

func TestCouponRepository_IncrementRespectsCap(t *testing.T) {
	ctx := context.Background()
	repo := NewCouponRepository(setupCouponTestDB(t))

	max := 2
	assert.NoError(t, repo.Create(ctx, sampleCoupon("c1", "X", &max)))

	assert.NoError(t, repo.IncrementRedemptionCount(ctx, nil, "c1"))
	assert.NoError(t, repo.IncrementRedemptionCount(ctx, nil, "c1"))

	err := repo.IncrementRedemptionCount(ctx, nil, "c1")
	assert.ErrorIs(t, err, domain.ErrCouponMaxReached)

	found, _ := repo.FindByID(ctx, "c1")
	assert.Equal(t, 2, found.RedemptionCount)
}

func TestCouponRepository_IncrementUnlimited(t *testing.T) {
	ctx := context.Background()
	repo := NewCouponRepository(setupCouponTestDB(t))

	assert.NoError(t, repo.Create(ctx, sampleCoupon("c1", "X", nil)))

	for i := 0; i < 5; i++ {
		assert.NoError(t, repo.IncrementRedemptionCount(ctx, nil, "c1"))
	}
	found, _ := repo.FindByID(ctx, "c1")
	assert.Equal(t, 5, found.RedemptionCount)
}

// TestCouponRepository_IncrementCapEnforced asserts the UPDATE-with-cap clause
// never lets the counter overshoot max_redemptions even when callers keep
// trying. The Postgres-side guarantee comes from row-level locking; here we
// only validate the SQL itself rejects post-cap attempts.
func TestCouponRepository_IncrementCapEnforced(t *testing.T) {
	ctx := context.Background()
	repo := NewCouponRepository(setupCouponTestDB(t))

	max := 3
	assert.NoError(t, repo.Create(ctx, sampleCoupon("c1", "X", &max)))

	for i := 0; i < 10; i++ {
		_ = repo.IncrementRedemptionCount(ctx, nil, "c1")
	}

	found, _ := repo.FindByID(ctx, "c1")
	assert.Equal(t, max, found.RedemptionCount)
}
