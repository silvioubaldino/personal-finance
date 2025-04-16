package domain

import (
	"time"
)

type TypePayment struct {
	ID          int       `json:"id,omitempty" gorm:"primaryKey"`
	Description string    `json:"description,omitempty"`
	UserID      string    `json:"user_id"`
	DateCreate  time.Time `json:"date_create"`
	DateUpdate  time.Time `json:"date_update"`
}
