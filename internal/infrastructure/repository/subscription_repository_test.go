package repository

import (
	"context"
	"testing"
	"time"

	"personal-finance/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupSubscriptionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	assert.NoError(t, db.AutoMigrate(&SubscriptionDB{}))
	return db
}

func TestSubscriptionRepository_Upsert(t *testing.T) {
	ctx := context.Background()
	start := time.Now().UTC().Truncate(time.Second)

	t.Run("inserts a new subscription and reads it back", func(t *testing.T) {
		repo := NewSubscriptionRepository(setupSubscriptionTestDB(t))

		sub := domain.Subscription{
			UserID:       "user-1",
			Source:       domain.SubscriptionSourceMercadoPago,
			ExternalID:   "mp-sub-1",
			Status:       domain.SubscriptionStatusActive,
			CurrentPrice: 9.90,
			Currency:     "BRL",
			StartedAt:    start,
		}

		saved, err := repo.Upsert(ctx, sub)
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, saved.ID)
		assert.Equal(t, "user-1", saved.UserID)
		assert.Equal(t, domain.SubscriptionStatusActive, saved.Status)
		assert.Equal(t, 9.90, saved.CurrentPrice)
	})

	t.Run("is idempotent on (source, external_id) and updates mutable fields", func(t *testing.T) {
		db := setupSubscriptionTestDB(t)
		repo := NewSubscriptionRepository(db)

		first, err := repo.Upsert(ctx, domain.Subscription{
			UserID:       "user-1",
			Source:       domain.SubscriptionSourceMercadoPago,
			ExternalID:   "mp-sub-1",
			Status:       domain.SubscriptionStatusActive,
			CurrentPrice: 9.90,
			StartedAt:    start,
		})
		assert.NoError(t, err)

		second, err := repo.Upsert(ctx, domain.Subscription{
			UserID:       "user-1",
			Source:       domain.SubscriptionSourceMercadoPago,
			ExternalID:   "mp-sub-1",
			Status:       domain.SubscriptionStatusCancelled,
			CurrentPrice: 9.90,
			StartedAt:    start,
		})
		assert.NoError(t, err)

		assert.Equal(t, first.ID, second.ID, "id must remain stable")
		assert.Equal(t, domain.SubscriptionStatusCancelled, second.Status)

		var count int64
		db.Model(&SubscriptionDB{}).Count(&count)
		assert.Equal(t, int64(1), count)
	})
}

func TestSubscriptionRepository_FindByExternalID(t *testing.T) {
	ctx := context.Background()
	repo := NewSubscriptionRepository(setupSubscriptionTestDB(t))

	_, err := repo.Upsert(ctx, domain.Subscription{
		UserID:     "user-1",
		Source:     domain.SubscriptionSourceMercadoPago,
		ExternalID: "mp-sub-1",
		Status:     domain.SubscriptionStatusActive,
		StartedAt:  time.Now(),
	})
	assert.NoError(t, err)

	found, err := repo.FindByExternalID(ctx, domain.SubscriptionSourceMercadoPago, "mp-sub-1")
	assert.NoError(t, err)
	assert.Equal(t, "user-1", found.UserID)

	_, err = repo.FindByExternalID(ctx, domain.SubscriptionSourceMercadoPago, "missing")
	assert.ErrorIs(t, err, ErrSubscriptionNotFound)
}

func TestSubscriptionRepository_List(t *testing.T) {
	ctx := context.Background()
	repo := NewSubscriptionRepository(setupSubscriptionTestDB(t))
	now := time.Now()

	_, _ = repo.Upsert(ctx, domain.Subscription{UserID: "u1", Source: domain.SubscriptionSourceMercadoPago, ExternalID: "a", Status: domain.SubscriptionStatusActive, StartedAt: now})
	_, _ = repo.Upsert(ctx, domain.Subscription{UserID: "u2", Source: domain.SubscriptionSourceMercadoPago, ExternalID: "b", Status: domain.SubscriptionStatusCancelled, StartedAt: now})
	_, _ = repo.Upsert(ctx, domain.Subscription{UserID: "u3", Source: domain.SubscriptionSourceMercadoPago, ExternalID: "c", Status: domain.SubscriptionStatusActive, StartedAt: now})

	all, err := repo.List(ctx, SubscriptionListFilter{})
	assert.NoError(t, err)
	assert.Len(t, all, 3)

	active, err := repo.List(ctx, SubscriptionListFilter{Status: domain.SubscriptionStatusActive})
	assert.NoError(t, err)
	assert.Len(t, active, 2)

	page, err := repo.List(ctx, SubscriptionListFilter{Page: 1, PageSize: 1})
	assert.NoError(t, err)
	assert.Len(t, page, 1)
}
