package domain

import (
	"time"

	"github.com/google/uuid"
)

type UserConsent struct {
	ID          uuid.UUID `json:"id"`
	UserID      string    `json:"user_id"`
	TermVersion string    `json:"term_version"`
	AgreedAt    time.Time `json:"agreed_at"`
	IPAddress   string    `json:"ip_address,omitempty"`
	UserAgent   string    `json:"user_agent,omitempty"`
}

func NewUserConsent(userID, termVersion, ipAddress, userAgent string) UserConsent {
	return UserConsent{
		ID:          uuid.New(),
		UserID:      userID,
		TermVersion: termVersion,
		AgreedAt:    time.Now(),
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
	}
}
