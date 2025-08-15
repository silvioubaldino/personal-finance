package output

import (
	"time"

	"personal-finance/internal/domain"

	"github.com/google/uuid"
)

type InvoiceOutput struct {
	ID           *uuid.UUID   `json:"id,omitempty"`
	CreditCardID *uuid.UUID   `json:"credit_card_id"`
	PeriodStart  time.Time    `json:"period_start"`
	PeriodEnd    time.Time    `json:"period_end"`
	DueDate      time.Time    `json:"due_date"`
	PaymentDate  *time.Time   `json:"payment_date,omitempty"`
	Amount       float64      `json:"amount"`
	IsPaid       bool         `json:"is_paid"`
	WalletID     *uuid.UUID   `json:"wallet_id,omitempty"`
	Wallet       WalletOutput `json:"wallet,omitempty"`
	DateCreate   time.Time    `json:"date_create"`
	DateUpdate   time.Time    `json:"date_update"`
}

func ToInvoiceOutput(input domain.Invoice) InvoiceOutput {
	return InvoiceOutput{
		ID:           input.ID,
		CreditCardID: input.CreditCardID,
		PeriodStart:  input.PeriodStart,
		PeriodEnd:    input.PeriodEnd,
		DueDate:      input.DueDate,
		PaymentDate:  input.PaymentDate,
		Amount:       input.Amount,
		IsPaid:       input.IsPaid,
		WalletID:     input.WalletID,
		Wallet:       ToWalletOutput(input.Wallet),
		DateCreate:   input.DateCreate,
		DateUpdate:   input.DateUpdate,
	}
}
