package domain

import (
	"time"

	"github.com/google/uuid"
)

type DetailedInvoice struct {
	Invoice
	Movements MovementList `json:"movements"`
}

type Invoice struct {
	ID           *uuid.UUID `json:"id,omitempty" gorm:"primaryKey"`
	CreditCardID *uuid.UUID `json:"credit_card_id"`
	CreditCard   CreditCard `json:"credit_card"`
	PeriodStart  time.Time  `json:"period_start"`
	PeriodEnd    time.Time  `json:"period_end"`
	DueDate      time.Time  `json:"due_date"`
	PaymentDate  *time.Time `json:"payment_date,omitempty"`
	Amount       float64    `json:"amount"`
	IsPaid       bool       `json:"is_paid"`
	WalletID     *uuid.UUID `json:"wallet_id,omitempty"`
	Wallet       Wallet     `json:"wallets,omitempty"`
	UserID       string     `json:"user_id"`
	DateCreate   time.Time  `json:"date_create"`
	DateUpdate   time.Time  `json:"date_update"`
}

func calculateInvoicePeriod(creditCardClosingDay int, date time.Time) (time.Time, time.Time) {
	year := date.Year()
	month := date.Month()

	periodStart := time.Date(year, month, creditCardClosingDay+1, 0, 0, 0, 0, date.Location())
	if date.Day() <= creditCardClosingDay {
		periodStart = periodStart.AddDate(0, -1, 0)
	}

	periodEnd := periodStart.AddDate(0, 1, 0).AddDate(0, 0, -1)

	return periodStart, periodEnd
}

func calculateDueDate(creditCardDueDay int, periodEnd time.Time) time.Time {
	dueMonth := periodEnd.Month()
	dueYear := periodEnd.Year()

	if periodEnd.Day() >= creditCardDueDay {
		dueMonth = dueMonth + 1
		if dueMonth > 12 {
			dueMonth = 1
			dueYear = dueYear + 1
		}
	}

	dueDate := time.Date(dueYear, dueMonth, creditCardDueDay, 0, 0, 0, 0, periodEnd.Location())

	return dueDate
}

func BuildInvoice(creditCard CreditCard, movementDate time.Time) Invoice {
	periodStart, periodEnd := calculateInvoicePeriod(creditCard.ClosingDay, movementDate)
	dueDate := calculateDueDate(creditCard.DueDay, periodEnd)

	return Invoice{
		CreditCardID: creditCard.ID,
		PeriodStart:  periodStart,
		PeriodEnd:    periodEnd,
		DueDate:      dueDate,
		Amount:       0,
		IsPaid:       false,
		WalletID:     creditCard.DefaultWalletID,
	}
}
