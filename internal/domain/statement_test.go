package domain_test

import (
	"testing"
	"time"

	"personal-finance/internal/domain"

	"github.com/google/uuid"
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

func TestStripInstallmentSuffix(t *testing.T) {
	type (
		input struct {
			description string
		}
		expected struct {
			output string
		}
	)

	tests := map[string]struct {
		// input
		input input
		// expected
		expected expected
	}{
		"should strip the bare n/m suffix with leading zero": {
			input:    input{description: "MERCADO LIVRE 03/12"},
			expected: expected{output: "MERCADO LIVRE"},
		},
		"should strip the bare n/m suffix without leading zero": {
			input:    input{description: "MERCADO LIVRE 3/12"},
			expected: expected{output: "MERCADO LIVRE"},
		},
		"should strip the PARC prefixed suffix": {
			input:    input{description: "MAGALU*MAGAZINELUIZA PARC 3/12"},
			expected: expected{output: "MAGALU*MAGAZINELUIZA"},
		},
		"should strip the abbreviated PARC. prefixed suffix": {
			input:    input{description: "MAGALU*MAGAZINELUIZA PARC. 03/12"},
			expected: expected{output: "MAGALU*MAGAZINELUIZA"},
		},
		"should strip the PARCELA prefixed suffix": {
			input:    input{description: "MERCADO LIVRE PARCELA 03/12"},
			expected: expected{output: "MERCADO LIVRE"},
		},
		"should strip the PARCELA n DE m form with leading zero": {
			input:    input{description: "DROGARIA SP PARCELA 03 DE 12"},
			expected: expected{output: "DROGARIA SP"},
		},
		"should strip the PARCELA n DE m form without leading zero": {
			input:    input{description: "DROGARIA SP PARCELA 3 DE 12"},
			expected: expected{output: "DROGARIA SP"},
		},
		"should strip the suffix in lowercase": {
			input:    input{description: "Mercado Livre parcela 03/12"},
			expected: expected{output: "Mercado Livre"},
		},
		"should strip the suffix when surrounded by spaces": {
			input:    input{description: "MERCADO 03/12 LIVRE"},
			expected: expected{output: "MERCADO LIVRE"},
		},
		"should strip the suffix when it opens the description": {
			input:    input{description: "PARCELA 03/12 MERCADO LIVRE"},
			expected: expected{output: "MERCADO LIVRE"},
		},
		"should keep a description with no installment suffix untouched": {
			input:    input{description: "SUPERMERCADO BOM PRECO"},
			expected: expected{output: "SUPERMERCADO BOM PRECO"},
		},
		"should keep an empty description untouched": {
			input:    input{description: ""},
			expected: expected{output: ""},
		},
		"should not strip a number that is not an installment": {
			input:    input{description: "POSTO 24/7"},
			expected: expected{output: "POSTO 24/7"},
		},
		"should not strip a plain number in the merchant name": {
			input:    input{description: "PADARIA 2 IRMAOS"},
			expected: expected{output: "PADARIA 2 IRMAOS"},
		},
		"should not strip a merchant name carrying a big number": {
			input:    input{description: "PAGUE MENOS 1234"},
			expected: expected{output: "PAGUE MENOS 1234"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Arrange
			description := tc.input.description

			// Act
			output := domain.StripInstallmentSuffix(description)

			// Assert
			assert.Equal(t, tc.expected.output, output)
		})
	}
}

func TestInstallmentMatchConfidence(t *testing.T) {
	type (
		input struct {
			extracted domain.ExtractedMovement
			candidate domain.Movement
		}
		expected struct {
			output float64
		}
	)

	tests := map[string]struct {
		// input
		input input
		// expected
		expected expected
	}{
		"should return high confidence when totals, amount and description root match": {
			input: input{
				extracted: extractedInstallment("MERCADO LIVRE PARCELA 03/12", -120.00, 3, 12),
				candidate: registeredInstallment("Mercado Livre 03/12", -120.00, 3, 12),
			},
			expected: expected{output: domain.InstallmentMatchConfidenceHigh},
		},
		"should return high confidence when the amount differs within tolerance": {
			input: input{
				extracted: extractedInstallment("MERCADO LIVRE PARCELA 03/12", -120.00, 3, 12),
				candidate: registeredInstallment("MERCADO LIVRE PARCELA 04/12", -119.97, 3, 12),
			},
			expected: expected{output: domain.InstallmentMatchConfidenceHigh},
		},
		"should return medium confidence when only the description root diverges": {
			input: input{
				extracted: extractedInstallment("MAGALU*MAGAZINELUIZA PARC 04/12", -119.90, 4, 12),
				candidate: registeredInstallment("TV da sala", -119.90, 4, 12),
			},
			expected: expected{output: domain.InstallmentMatchConfidenceMedium},
		},
		"should return no confidence when the amount is beyond tolerance": {
			input: input{
				extracted: extractedInstallment("MERCADO LIVRE PARCELA 03/12", -120.00, 3, 12),
				candidate: registeredInstallment("Mercado Livre", -130.00, 3, 12),
			},
			expected: expected{output: domain.InstallmentMatchConfidenceNone},
		},
		"should return no confidence when the total of installments differs": {
			input: input{
				extracted: extractedInstallment("MERCADO LIVRE PARCELA 03/12", -120.00, 3, 12),
				candidate: registeredInstallment("Mercado Livre", -120.00, 3, 10),
			},
			expected: expected{output: domain.InstallmentMatchConfidenceNone},
		},
		"should return no confidence when the installment number is from another competency": {
			input: input{
				extracted: extractedInstallment("MERCADO LIVRE PARCELA 03/12", -120.00, 3, 12),
				candidate: registeredInstallment("Mercado Livre", -120.00, 5, 12),
			},
			expected: expected{output: domain.InstallmentMatchConfidenceNone},
		},
		"should return no confidence when the extracted item is not an installment": {
			input: input{
				extracted: domain.ExtractedMovement{Description: "NETFLIX", Amount: -55.90},
				candidate: registeredInstallment("Netflix", -55.90, 1, 2),
			},
			expected: expected{output: domain.InstallmentMatchConfidenceNone},
		},
		"should return no confidence when the candidate is not an installment": {
			input: input{
				extracted: extractedInstallment("MERCADO LIVRE PARCELA 03/12", -120.00, 3, 12),
				candidate: domain.Movement{Description: "Mercado Livre", Amount: -120.00},
			},
			expected: expected{output: domain.InstallmentMatchConfidenceNone},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Arrange
			var (
				extracted = tc.input.extracted
				candidate = tc.input.candidate
			)

			// Act
			output := domain.InstallmentMatchConfidence(extracted, candidate)

			// Assert
			assert.Equal(t, tc.expected.output, output)
		})
	}
}

func TestResolveTargetInvoicePeriodEnd(t *testing.T) {
	type (
		input struct {
			creditCard domain.CreditCard
			dates      []time.Time
		}
		expected struct {
			periodEnd time.Time
			found     bool
		}
	)

	closingDay3 := domain.CreditCard{ClosingDay: 3, DueDay: 10}

	tests := map[string]struct {
		// input
		input input
		// expected
		expected expected
	}{
		"should pick the period holding most of the items": {
			input: input{
				creditCard: closingDay3,
				dates: []time.Time{
					date("2026-05-12"), date("2026-05-20"), date("2026-05-28"),
					date("2026-09-03"),
				},
			},
			expected: expected{periodEnd: date("2026-06-03"), found: true},
		},
		"should ignore a single outlier dated far in the past": {
			input: input{
				creditCard: closingDay3,
				dates: []time.Time{
					date("2025-01-10"),
					date("2026-05-12"), date("2026-05-20"),
				},
			},
			expected: expected{periodEnd: date("2026-06-03"), found: true},
		},
		"should return not found when the card has no closing day": {
			input: input{
				creditCard: domain.CreditCard{},
				dates:      []time.Time{date("2026-05-12")},
			},
			expected: expected{periodEnd: time.Time{}, found: false},
		},
		"should return not found when there are no dates": {
			input: input{
				creditCard: closingDay3,
				dates:      []time.Time{},
			},
			expected: expected{periodEnd: time.Time{}, found: false},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Arrange
			var (
				creditCard = tc.input.creditCard
				dates      = tc.input.dates
			)

			// Act
			periodEnd, found := domain.ResolveTargetInvoicePeriodEnd(creditCard, dates)

			// Assert
			assert.Equal(t, tc.expected.found, found)
			assert.Equal(t, tc.expected.periodEnd, periodEnd)
		})
	}
}

// --- fixtures ---

func date(value string) time.Time {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		panic(err)
	}
	return parsed
}

func extractedInstallment(description string, amount float64, number, total int) domain.ExtractedMovement {
	return domain.ExtractedMovement{
		Description:       description,
		Amount:            amount,
		InstallmentNumber: &number,
		TotalInstallments: &total,
	}
}

func registeredInstallment(description string, amount float64, number, total int) domain.Movement {
	var (
		movementID = uuid.New()
		groupID    = uuid.New()
	)
	return domain.Movement{
		ID:          &movementID,
		Description: description,
		Amount:      amount,
		CreditCardInfo: &domain.CreditCardMovement{
			InstallmentGroupID: &groupID,
			InstallmentNumber:  &number,
			TotalInstallments:  &total,
		},
	}
}
