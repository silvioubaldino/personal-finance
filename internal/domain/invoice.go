package domain

import (
	"math"
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

	AdvancePaidAmount float64                 `json:"advance_paid_amount" gorm:"-"`
	RemainingAmount   float64                 `json:"remaining_amount" gorm:"-"`
	AdvancePayments   []InvoiceAdvancePayment `json:"advance_payments,omitempty" gorm:"-"`
}

type InvoiceAdvancePayment struct {
	ID          uuid.UUID  `json:"id"`
	PaymentDate time.Time  `json:"payment_date"`
	Amount      float64    `json:"amount"`
	WalletID    *uuid.UUID `json:"wallet_id,omitempty"`
	Description string     `json:"description"`
}

func RoundCents(value float64) float64 {
	return math.Round(value*100) / 100
}

func NewInvoiceAdvancePayments(movements MovementList) []InvoiceAdvancePayment {
	result := make([]InvoiceAdvancePayment, 0, len(movements))
	for _, movement := range movements {
		if movement.ID == nil || movement.TypePayment != TypePaymentInvoiceAdvancePayment {
			continue
		}
		payment := InvoiceAdvancePayment{
			ID:          *movement.ID,
			Amount:      movement.Amount,
			WalletID:    movement.WalletID,
			Description: movement.Description,
		}
		if movement.Date != nil {
			payment.PaymentDate = *movement.Date
		}
		result = append(result, payment)
	}
	return result
}

func (i Invoice) WithAdvancePayments(payments []InvoiceAdvancePayment) Invoice {
	var advance float64
	for _, payment := range payments {
		advance += payment.Amount
	}
	i.AdvancePayments = payments
	i.AdvancePaidAmount = RoundCents(advance)
	i.RemainingAmount = RoundCents(i.Amount - advance)
	return i
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
