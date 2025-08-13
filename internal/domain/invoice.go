package domain

import (
	"time"

	"github.com/google/uuid"
)

type Invoice struct {
	ID           *uuid.UUID `json:"id,omitempty" gorm:"primaryKey"`
	CreditCardID *uuid.UUID `json:"credit_card_id"`
	PeriodStart  time.Time  `json:"period_start"`
	PeriodEnd    time.Time  `json:"period_end"`
	DueDay       time.Time  `json:"due_day"`
	PaymentDate  *time.Time `json:"payment_date,omitempty"`
	Amount       float64    `json:"amount"`
	IsPaid       bool       `json:"is_paid"`
	WalletID     *uuid.UUID `json:"wallet_id,omitempty"`
	Wallet       Wallet     `json:"wallets,omitempty"`
	UserID       string     `json:"user_id"`
	DateCreate   time.Time  `json:"date_create"`
	DateUpdate   time.Time  `json:"date_update"`
}
