package output

import (
	"time"

	"personal-finance/internal/domain"

	"github.com/google/uuid"
)

type InvoiceOutput struct {
	ID           *uuid.UUID   `json:"id,omitempty"`
	Name         string       `json:"name"`
	CreditCardID *uuid.UUID   `json:"credit_card_id"`
	PeriodStart  time.Time    `json:"period_start"`
	PeriodEnd    time.Time    `json:"period_end"`
	DueDate      time.Time    `json:"due_date"`
	PaymentDate  *time.Time   `json:"payment_date,omitempty"`
	Amount       float64      `json:"amount"`
	IsPaid       bool         `json:"is_paid"`
	Wallet       WalletOutput `json:"wallet,omitempty"`
	DateUpdate   time.Time    `json:"date_update"`
}

func ToInvoiceOutput(input domain.Invoice) InvoiceOutput {
	return InvoiceOutput{
		ID:           input.ID,
		Name:         input.DueDate.Month().String(),
		CreditCardID: input.CreditCardID,
		PeriodStart:  input.PeriodStart,
		PeriodEnd:    input.PeriodEnd,
		DueDate:      input.DueDate,
		PaymentDate:  input.PaymentDate,
		Amount:       input.Amount,
		IsPaid:       input.IsPaid,
		Wallet:       ToWalletOutput(input.Wallet),
		DateUpdate:   input.DateUpdate,
	}
}
