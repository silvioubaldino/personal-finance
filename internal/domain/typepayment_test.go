package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTypePayment_IsInvoicePayment(t *testing.T) {
	tests := map[string]struct {
		typePayment TypePayment
		expected    bool
	}{
		"settlement is an invoice payment":       {typePayment: TypePaymentInvoicePayment, expected: true},
		"advance payment is an invoice payment":  {typePayment: TypePaymentInvoiceAdvancePayment, expected: true},
		"remainder is not an invoice payment":    {typePayment: TypePaymentInvoiceRemainder, expected: false},
		"credit card purchase is not a payment":  {typePayment: TypePaymentCreditCard, expected: false},
		"internal transfer is not a payment":     {typePayment: TypePaymentInternalTransfer, expected: false},
		"empty type is not an invoice payment":   {typePayment: "", expected: false},
		"pix is not an invoice payment":          {typePayment: TypePaymentPix, expected: false},
		"unknown type is not an invoice payment": {typePayment: "other", expected: false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.typePayment.IsInvoicePayment())
		})
	}
}

func TestInvoicePaymentTypes_MatchesHelper(t *testing.T) {
	for _, typePayment := range InvoicePaymentTypes() {
		assert.True(t, typePayment.IsInvoicePayment(), string(typePayment))
	}
	assert.ElementsMatch(t,
		[]string{"invoice_payment", "invoice_advance_payment", "internal_transfer"},
		NonRealizedTypePayments(),
	)
}
