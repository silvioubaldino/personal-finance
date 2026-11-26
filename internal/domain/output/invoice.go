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

	// Derivados do adiantamento (AYD-007@context): Amount segue bruto.
	AdvancePaidAmount float64 `json:"advance_paid_amount"`
	RemainingAmount   float64 `json:"remaining_amount"`
}

type AdvancePaymentOutput struct {
	ID          uuid.UUID  `json:"id"`
	PaymentDate string     `json:"payment_date"`
	Amount      float64    `json:"amount"`
	WalletID    *uuid.UUID `json:"wallet_id,omitempty"`
	Description string     `json:"description"`
}

func ToAdvancePaymentsOutput(payments []domain.InvoiceAdvancePayment) []AdvancePaymentOutput {
	result := make([]AdvancePaymentOutput, len(payments))
	for i, payment := range payments {
		result[i] = AdvancePaymentOutput{
			ID:          payment.ID,
			PaymentDate: payment.PaymentDate.Format("2006-01-02"),
			Amount:      payment.Amount,
			WalletID:    payment.WalletID,
			Description: payment.Description,
		}
	}
	return result
}

// InvoiceWithAdvancePaymentsOutput é a Invoice com a lista advance_payments: devolvida por
// GET /v2/invoices/{id} e pelos endpoints de adiantamento.
type InvoiceWithAdvancePaymentsOutput struct {
	InvoiceOutput
	AdvancePayments []AdvancePaymentOutput `json:"advance_payments"`
}

func ToInvoiceWithAdvancePaymentsOutput(input domain.Invoice) InvoiceWithAdvancePaymentsOutput {
	return InvoiceWithAdvancePaymentsOutput{
		InvoiceOutput:   ToInvoiceOutput(input),
		AdvancePayments: ToAdvancePaymentsOutput(input.AdvancePayments),
	}
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

		AdvancePaidAmount: input.AdvancePaidAmount,
		RemainingAmount:   input.RemainingAmount,
	}
}

type DetailedInvoiceOutput struct {
	InvoiceOutput
	AdvancePayments []AdvancePaymentOutput `json:"advance_payments"`
	Movements       MovementListOutput     `json:"movements"`
}

func ToDetailedInvoiceOutput(input domain.DetailedInvoice) DetailedInvoiceOutput {
	return DetailedInvoiceOutput{
		InvoiceOutput:   ToInvoiceOutput(input.Invoice),
		AdvancePayments: ToAdvancePaymentsOutput(input.AdvancePayments),
		Movements:       ToMovementListOutput(input.Movements),
	}
}
