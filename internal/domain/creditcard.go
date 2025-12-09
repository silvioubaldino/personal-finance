package domain

import (
	"time"

	"github.com/google/uuid"
)

type CreditCard struct {
	ID              *uuid.UUID `json:"id,omitempty" gorm:"primaryKey"`
	Name            string     `json:"name"`
	CreditLimit     float64    `json:"credit_limit"`
	ClosingDay      int        `json:"closing_day"`
	DueDay          int        `json:"due_day"`
	DefaultWalletID *uuid.UUID `json:"default_wallet_id"`
	DefaultWallet   Wallet     `json:"wallets,omitempty"`
	UserID          string     `json:"user_id"`
	DateCreate      time.Time  `json:"date_create"`
	DateUpdate      time.Time  `json:"date_update"`
}

type CreditCardWithOpenInvoices struct {
	CreditCard
	OpenInvoices []Invoice `json:"open_invoices"`
}

func (c CreditCard) HasSufficientLimit(amount float64) bool {
	return c.CreditLimit+amount >= 0
}
