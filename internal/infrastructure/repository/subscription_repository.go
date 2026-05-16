package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"personal-finance/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrSubscriptionNotFound = errors.New("subscription not found")

type SubscriptionRepository struct {
	db *gorm.DB
}

func NewSubscriptionRepository(db *gorm.DB) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

// Upsert inserts or updates a subscription identified by (source, external_id).
// CreatedAt is preserved on update; UpdatedAt is refreshed.
func (r *SubscriptionRepository) Upsert(ctx context.Context, sub domain.Subscription) (domain.Subscription, error) {
	now := time.Now()
	if sub.ID == uuid.Nil {
		sub.ID = uuid.New()
	}
	if sub.CreatedAt.IsZero() {
		sub.CreatedAt = now
	}
	sub.UpdatedAt = now

	row := FromSubscriptionDomain(sub)

	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "source"}, {Name: "external_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"plan_id",
				"status",
				"current_price",
				"currency",
				"current_period_end",
				"cancelled_at",
				"external_product_id",
				"updated_at",
			}),
		}).
		Create(&row).Error
	if err != nil {
		return domain.Subscription{}, fmt.Errorf("error upserting subscription: %w: %s", ErrDatabaseError, err.Error())
	}

	var result SubscriptionDB
	err = r.db.WithContext(ctx).
		Where("source = ? AND external_id = ?", row.Source, row.ExternalID).
		First(&result).Error
	if err != nil {
		return domain.Subscription{}, fmt.Errorf("error reading subscription after upsert: %w: %s", ErrDatabaseError, err.Error())
	}

	return result.ToDomain(), nil
}

func (r *SubscriptionRepository) FindByExternalID(ctx context.Context, source domain.SubscriptionSource, externalID string) (domain.Subscription, error) {
	var row SubscriptionDB
	err := r.db.WithContext(ctx).
		Where("source = ? AND external_id = ?", string(source), externalID).
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Subscription{}, ErrSubscriptionNotFound
		}
		return domain.Subscription{}, fmt.Errorf("error finding subscription: %w: %s", ErrDatabaseError, err.Error())
	}
	return row.ToDomain(), nil
}

func (r *SubscriptionRepository) FindActiveByUser(ctx context.Context, userID string) ([]domain.Subscription, error) {
	var rows []SubscriptionDB
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND status = ?", userID, string(domain.SubscriptionStatusActive)).
		Order("started_at desc").
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("error listing active subscriptions: %w: %s", ErrDatabaseError, err.Error())
	}
	out := make([]domain.Subscription, len(rows))
	for i, row := range rows {
		out[i] = row.ToDomain()
	}
	return out, nil
}

type SubscriptionListFilter struct {
	Status   domain.SubscriptionStatus
	Source   domain.SubscriptionSource
	Page     int
	PageSize int
}

func (r *SubscriptionRepository) List(ctx context.Context, filter SubscriptionListFilter) ([]domain.Subscription, error) {
	if filter.PageSize <= 0 || filter.PageSize > 200 {
		filter.PageSize = 50
	}
	if filter.Page < 1 {
		filter.Page = 1
	}

	query := r.db.WithContext(ctx).Model(&SubscriptionDB{})
	if filter.Status != "" {
		query = query.Where("status = ?", string(filter.Status))
	}
	if filter.Source != "" {
		query = query.Where("source = ?", string(filter.Source))
	}

	var rows []SubscriptionDB
	err := query.
		Order("started_at desc").
		Limit(filter.PageSize).
		Offset((filter.Page - 1) * filter.PageSize).
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("error listing subscriptions: %w: %s", ErrDatabaseError, err.Error())
	}

	out := make([]domain.Subscription, len(rows))
	for i, row := range rows {
		out[i] = row.ToDomain()
	}
	return out, nil
}
