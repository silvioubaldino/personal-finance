package model

import (
	"errors"
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

	Balance struct {
		Period  Period  `json:"period"`
		Expense float64 `json:"expense"`
		Income  float64 `json:"income"`
	}

	Period struct {
		From time.Time `json:"from"`
		To   time.Time `json:"to"`
	}
)

func (p *Period) Validate() error {
	now := time.Now()
	if p.From == p.To {
		return errors.New("date must be informed")
	}

	if p.From.IsZero() {
		p.From = now
	}
	if p.To.IsZero() {
		p.To = now
	}

	if p.From.After(p.To) {
		return errors.New("'from' must be before 'to'")
	}

	return nil
}
