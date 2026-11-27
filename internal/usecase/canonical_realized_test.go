package usecase

import (
	"testing"

	"personal-finance/internal/domain"

	"github.com/stretchr/testify/assert"
)

func TestIsCanonicalRealized_InvoicePaymentSet(t *testing.T) {
	tests := map[string]struct {
		typePayment domain.TypePayment
		expected    bool
	}{
		"debit movement is realized":                      {typePayment: domain.TypePaymentDebit, expected: true},
		"invoice settlement is never realized":            {typePayment: domain.TypePaymentInvoicePayment, expected: false},
		"invoice advance payment is never realized":       {typePayment: domain.TypePaymentInvoiceAdvancePayment, expected: false},
		"internal transfer is never realized":             {typePayment: domain.TypePaymentInternalTransfer, expected: false},
		"pix movement is realized":                        {typePayment: domain.TypePaymentPix, expected: true},
		"empty type payment is realized (standalone row)": {typePayment: "", expected: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			movement := domain.Movement{TypePayment: tc.typePayment}

			assert.Equal(t, tc.expected, isCanonicalRealized(movement))
		})
	}
}
