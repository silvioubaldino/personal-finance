package domain_test

import (
	"testing"

	"personal-finance/internal/domain"

	"github.com/stretchr/testify/assert"
)

func TestIsInvoicePaymentDescription(t *testing.T) {
	type (
		input struct {
			description string
		}
		expected struct {
			output bool
		}
	)

	tests := map[string]struct {
		// input
		input input
		// expected
		expected expected
	}{
		"should detect the Inter payment line": {
			input:    input{description: "PAGAMENTO ON LINE"},
			expected: expected{output: true},
		},
		"should detect the Itau payment line": {
			input:    input{description: "PAGAMENTO EFETUADO"},
			expected: expected{output: true},
		},
		"should detect the Nubank payment line regardless of case": {
			input:    input{description: "Pagamento recebido"},
			expected: expected{output: true},
		},
		"should detect the Bradesco payment line with punctuation": {
			input:    input{description: "PAGTO. POR DEB. CONTA"},
			expected: expected{output: true},
		},
		"should detect the PGTO abbreviation": {
			input:    input{description: "PGTO FATURA ANTERIOR"},
			expected: expected{output: true},
		},
		"should not flag a drugstore named PAGUE MENOS": {
			input:    input{description: "PAGUE MENOS 1234"},
			expected: expected{output: false},
		},
		"should not flag a PAGSEGURO merchant": {
			input:    input{description: "PAGSEGURO *LOJA X"},
			expected: expected{output: false},
		},
		"should not flag a PAGBANK merchant": {
			input:    input{description: "PAGBANK LOJA"},
			expected: expected{output: false},
		},
		"should not flag a regular purchase": {
			input:    input{description: "MERCADINHO PIRATININGA"},
			expected: expected{output: false},
		},
		"should not flag a merchant with a leading number": {
			input:    input{description: "212 SHIBATA"},
			expected: expected{output: false},
		},
		"should not flag when the payment word is not the first one": {
			input:    input{description: "LOJA DE PAGAMENTO RAPIDO"},
			expected: expected{output: false},
		},
		"should not flag an empty description": {
			input:    input{description: ""},
			expected: expected{output: false},
		},
		"should not flag a description with no letters": {
			input:    input{description: "123 456"},
			expected: expected{output: false},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Arrange
			description := tc.input.description

			// Act
			output := domain.IsInvoicePaymentDescription(description)

			// Assert
			assert.Equal(t, tc.expected.output, output)
		})
	}
}
