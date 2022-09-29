package model

import (
	"time"
)

type (
	Wallet struct {
		ID          int       `json:"id,omitempty" gorm:"primaryKey"`
		Description string    `json:"description,omitempty"`
		Balance     float64   `json:"balance"`
		DateCreate  time.Time `json:"date_create"`
		DateUpdate  time.Time `json:"date_update"`
	}

	TypePayment struct {
		ID          int       `json:"id,omitempty" gorm:"primaryKey"`
		Description string    `json:"description,omitempty"`
		DateCreate  time.Time `json:"date_create"`
		DateUpdate  time.Time `json:"date_update"`
	}

	Category struct {
		ID          int       `json:"id,omitempty" gorm:"primaryKey"`
		Description string    `json:"description,omitempty"`
		DateCreate  time.Time `json:"date_create"`
		DateUpdate  time.Time `json:"date_update"`
	}

	Transaction struct {
		ID            int       `json:"id,omitempty" gorm:"primaryKey"`
		Description   string    `json:"description,omitempty"`
		Amount        float64   `json:"amount"`
		Date          time.Time `json:"date"`
		WalletID      int       `json:"wallet_id"`
		TypePaymentID int       `json:"type_payment_id"`
		CategoryID    int       `json:"category_id"`
		DateCreate    time.Time `json:"date_create"`
		DateUpdate    time.Time `json:"date_update"`
	}
)
