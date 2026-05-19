package domain

import (
	"time"

	"github.com/google/uuid"
)

type SubscriptionSource string

const (
	SubscriptionSourceMercadoPago SubscriptionSource = "mercadopago"
	SubscriptionSourceApple       SubscriptionSource = "apple"
	SubscriptionSourceGoogle      SubscriptionSource = "google"
)

type SubscriptionStatus string

const (
	SubscriptionStatusActive    SubscriptionStatus = "active"
	SubscriptionStatusCancelled SubscriptionStatus = "cancelled"
	SubscriptionStatusExpired   SubscriptionStatus = "expired"
	SubscriptionStatusPaused    SubscriptionStatus = "paused"
	SubscriptionStatusPastDue   SubscriptionStatus = "past_due"
)

type Subscription struct {
	ID                uuid.UUID
	UserID            string
	Source            SubscriptionSource
	ExternalID        string
	ExternalProductID string
	PlanID            string
	Status            SubscriptionStatus
	CurrentPrice      float64
	Currency          string
	StartedAt         time.Time
	CurrentPeriodEnd  *time.Time
	CancelledAt       *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
