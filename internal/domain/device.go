package domain

import (
	"time"

	"github.com/google/uuid"
)

type Platform string

const (
	PlatformIOS     Platform = "ios"
	PlatformAndroid Platform = "android"
)

func (p Platform) IsValid() bool {
	return p == PlatformIOS || p == PlatformAndroid
}

type Device struct {
	ID            uuid.UUID  `json:"id"`
	UserID        string     `json:"user_id"`
	ExpoPushToken string     `json:"expo_push_token"`
	Platform      Platform   `json:"platform"`
	DateCreate    time.Time  `json:"date_create"`
	DateUpdate    time.Time  `json:"date_update"`
	LastSeenAt    *time.Time `json:"last_seen_at,omitempty"`
}

func (d *Device) UpdateLastSeen() {
	now := time.Now()
	d.LastSeenAt = &now
	d.DateUpdate = now
}
