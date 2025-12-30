package domain

import (
	"time"

	"github.com/google/uuid"
)

type Wallet struct {
	ID             *uuid.UUID `json:"id,omitempty" gorm:"primaryKey"`
	Description    string     `json:"description,omitempty"`
	Balance        float64    `json:"balance"`
	UserID         string     `json:"user_id"`
	InitialBalance float64    `json:"initial_balance"`
	InitialDate    time.Time  `json:"initial_date"`
	DateCreate     time.Time  `json:"date_create"`
	DateUpdate     time.Time  `json:"date_update"`
}

func (w *Wallet) HasSufficientBalance(amount float64) bool {
	if amount >= 0 {
		return true
	}
	return w.Balance+amount >= 0
}

func (w *Wallet) Pay(amount float64) error {
	if !w.HasSufficientBalance(amount) {
		return ErrWalletInsufficient
	}
	w.Balance += amount
	return nil
}

func (w *Wallet) RevertPayment(amount float64) error {
	w.Balance -= amount
	return nil
}
