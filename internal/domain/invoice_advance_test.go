package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestInvoice_WithAdvancePayments(t *testing.T) {
	id1, id2 := uuid.New(), uuid.New()
	date := time.Date(2026, 1, 11, 0, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		amount            float64
		payments          []InvoiceAdvancePayment
		expectedAdvance   float64
		expectedRemaining float64
	}{
		"no advance keeps remaining equal to amount": {
			amount:            -1503.90,
			expectedAdvance:   0,
			expectedRemaining: -1503.90,
		},
		"sums advances and subtracts from amount (Mercado Pago case)": {
			amount: -1503.90,
			payments: []InvoiceAdvancePayment{
				{ID: id1, PaymentDate: date, Amount: -150},
				{ID: id2, PaymentDate: date.AddDate(0, 0, 5), Amount: -49},
			},
			expectedAdvance:   -199,
			expectedRemaining: -1304.90,
		},
		"rounds floating point noise to cents": {
			amount: -0.3,
			payments: []InvoiceAdvancePayment{
				{ID: id1, Amount: -0.1},
				{ID: id2, Amount: -0.1},
			},
			expectedAdvance:   -0.2,
			expectedRemaining: -0.1,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			invoice := Invoice{Amount: tc.amount}

			result := invoice.WithAdvancePayments(tc.payments)

			assert.Equal(t, tc.amount, result.Amount)
			assert.Equal(t, tc.expectedAdvance, result.AdvancePaidAmount)
			assert.Equal(t, tc.expectedRemaining, result.RemainingAmount)
			assert.Equal(t, tc.payments, result.AdvancePayments)
		})
	}
}

func TestNewInvoiceAdvancePayments(t *testing.T) {
	id := uuid.New()
	walletID := uuid.New()
	date := time.Date(2026, 1, 11, 0, 0, 0, 0, time.UTC)

	movements := MovementList{
		{ID: &id, Amount: -150, Date: &date, WalletID: &walletID, Description: "Adiantamento", TypePayment: TypePaymentInvoiceAdvancePayment},
		{ID: &id, Amount: -10, TypePayment: TypePaymentInvoicePayment},
		{Amount: -5, TypePayment: TypePaymentInvoiceAdvancePayment},
	}

	result := NewInvoiceAdvancePayments(movements)

	assert.Equal(t, []InvoiceAdvancePayment{
		{ID: id, PaymentDate: date, Amount: -150, WalletID: &walletID, Description: "Adiantamento"},
	}, result)
}
