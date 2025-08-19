package output

import (
	"time"

	"personal-finance/internal/domain"

	"github.com/google/uuid"
)

type InvoiceOutput struct {
	ID          *uuid.UUID          `json:"id,omitempty"`
	Name        string              `json:"name"`
	CreditCard  CreditCardOutputDTO `json:"credit_card"`
	PeriodStart time.Time           `json:"period_start"`
	PeriodEnd   time.Time           `json:"period_end"`
	DueDate     time.Time           `json:"due_date"`
	PaymentDate *time.Time          `json:"payment_date,omitempty"`
	Amount      float64             `json:"amount"`
	IsPaid      bool                `json:"is_paid"`
	Wallet      WalletOutputDTO     `json:"wallet,omitempty"`
	DateUpdate  time.Time           `json:"date_update"`
}

func ToInvoiceOutput(input domain.Invoice) InvoiceOutput {
	return InvoiceOutput{
		ID:          input.ID,
		Name:        input.DueDate.Month().String(),
		CreditCard:  ToCreditCardOutputDTO(input.CreditCard),
		PeriodStart: input.PeriodStart,
		PeriodEnd:   input.PeriodEnd,
		DueDate:     input.DueDate,
		PaymentDate: input.PaymentDate,
		Amount:      input.Amount,
		IsPaid:      input.IsPaid,
		Wallet:      ToWalletOutputDTO(input.Wallet),
		DateUpdate:  input.DateUpdate,
	}
}

type DetailedInvoiceOutput struct {
	Invoice   InvoiceOutput      `json:"invoice"`
	Movements MovementListOutput `json:"movements"`
}

func ToDetailedInvoiceOutput(input domain.DetailedInvoice) DetailedInvoiceOutput {
	return DetailedInvoiceOutput{
		Invoice:   ToInvoiceOutput(input.Invoice),
		Movements: ToMovementListOutput(input.Movements),
	}
}
