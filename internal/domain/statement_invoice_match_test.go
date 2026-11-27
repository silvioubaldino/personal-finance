package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestIsCardPaymentDescription(t *testing.T) {
	tests := map[string]struct {
		description string
		expected    bool
	}{
		"pagamento fatura":          {description: "PAGAMENTO FATURA NUBANK", expected: true},
		"pag fat abbreviation":      {description: "PAG FAT CARTAO 1234", expected: true},
		"pgto cartao with accent":   {description: "Pgto Cartão de Crédito", expected: true},
		"boleto cartao":             {description: "BOLETO CARTAO ITAU", expected: true},
		"pagamento da fatura":       {description: "Pagamento da fatura", expected: true},
		"regular pix is not":        {description: "PIX FULANO DE TAL", expected: false},
		"supermarket is not":        {description: "MERCADO PIRATININGA", expected: false},
		"empty description is not":  {description: "", expected: false},
		"fatura of utility is not":  {description: "ENERGIA ELETRICA FATURA 03/2026", expected: false},
		"loose cartao word is not":  {description: "COMPRA CARTAO DEBITO LOJA", expected: false},
		"punctuation is normalized": {description: "PAG.FAT.CARTAO", expected: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, IsCardPaymentDescription(tc.description))
		})
	}
}

func TestSuggestInvoicePaymentMatch(t *testing.T) {
	mercadoPago := OpenInvoiceForPayment{
		InvoiceID: uuid.New(), CreditCardID: uuid.New(), CreditCardName: "Mercado Pago",
		DueDate: d(2026, 2, 4), RemainingAmount: -1503.90,
	}
	nubank := OpenInvoiceForPayment{
		InvoiceID: uuid.New(), CreditCardID: uuid.New(), CreditCardName: "Nubank",
		DueDate: d(2026, 2, 10), RemainingAmount: -800.00,
	}
	nubankNext := OpenInvoiceForPayment{
		InvoiceID: uuid.New(), CreditCardID: nubank.CreditCardID, CreditCardName: "Nubank",
		DueDate: d(2026, 3, 10), RemainingAmount: -800.00,
	}

	tests := map[string]struct {
		line         ExtractedMovement
		invoices     []OpenInvoiceForPayment
		expectedNil  bool
		expectedID   uuid.UUID
		expectedKind string
		confidence   float64
	}{
		"exact remaining amount until due date is a pre-decided settlement": {
			line:         ExtractedMovement{Date: "2026-02-03", Description: "PAGAMENTO FATURA", Amount: -1503.90},
			invoices:     []OpenInvoiceForPayment{mercadoPago},
			expectedID:   mercadoPago.InvoiceID,
			expectedKind: InvoicePaymentKindSettlement,
			confidence:   InvoicePaymentMatchConfidenceHigh,
		},
		"exact remaining amount up to 5 days after due date is still a settlement": {
			line:         ExtractedMovement{Date: "2026-02-09", Description: "PAGAMENTO FATURA", Amount: -1503.90},
			invoices:     []OpenInvoiceForPayment{mercadoPago},
			expectedID:   mercadoPago.InvoiceID,
			expectedKind: InvoicePaymentKindSettlement,
			confidence:   InvoicePaymentMatchConfidenceHigh,
		},
		"exact remaining amount 6 days after due date is not matched": {
			line:        ExtractedMovement{Date: "2026-02-10", Description: "PAGAMENTO FATURA", Amount: -1503.90},
			invoices:    []OpenInvoiceForPayment{mercadoPago},
			expectedNil: true,
		},
		"one cent of tolerance on the settlement amount": {
			line:         ExtractedMovement{Date: "2026-02-03", Description: "PAGAMENTO FATURA", Amount: -1503.91},
			invoices:     []OpenInvoiceForPayment{mercadoPago},
			expectedID:   mercadoPago.InvoiceID,
			expectedKind: InvoicePaymentKindSettlement,
			confidence:   InvoicePaymentMatchConfidenceHigh,
		},
		"two cents above the remaining is beyond the tolerance": {
			line:        ExtractedMovement{Date: "2026-02-03", Description: "PAGAMENTO FATURA", Amount: -1503.92},
			invoices:    []OpenInvoiceForPayment{mercadoPago},
			expectedNil: true,
		},
		"lower amount flagged by the model is a suggested advance": {
			line:         ExtractedMovement{Date: "2026-01-11", Description: "PIX MERCADO PAGO FATURA", Amount: -150.00, IsCardPayment: true},
			invoices:     []OpenInvoiceForPayment{mercadoPago},
			expectedID:   mercadoPago.InvoiceID,
			expectedKind: InvoicePaymentKindAdvance,
			confidence:   InvoicePaymentMatchConfidenceMedium,
		},
		"lower amount with a card payment phrase is a suggested advance": {
			line:         ExtractedMovement{Date: "2026-01-11", Description: "PAG FAT CARTAO", Amount: -150.00},
			invoices:     []OpenInvoiceForPayment{mercadoPago},
			expectedID:   mercadoPago.InvoiceID,
			expectedKind: InvoicePaymentKindAdvance,
			confidence:   InvoicePaymentMatchConfidenceMedium,
		},
		"lower amount with the card name in the description is a suggested advance": {
			line:         ExtractedMovement{Date: "2026-01-11", Description: "PIX MERCADO PAGO", Amount: -150.00},
			invoices:     []OpenInvoiceForPayment{mercadoPago},
			expectedID:   mercadoPago.InvoiceID,
			expectedKind: InvoicePaymentKindAdvance,
			confidence:   InvoicePaymentMatchConfidenceMedium,
		},
		"advance after the due date is not matched": {
			line:        ExtractedMovement{Date: "2026-02-05", Description: "PAGAMENTO FATURA", Amount: -150.00},
			invoices:    []OpenInvoiceForPayment{mercadoPago},
			expectedNil: true,
		},
		"amount above the remaining is not matched": {
			line:        ExtractedMovement{Date: "2026-02-03", Description: "PAGAMENTO FATURA", Amount: -2000.00},
			invoices:    []OpenInvoiceForPayment{mercadoPago},
			expectedNil: true,
		},
		"no textual signal means no match even when the amount coincides": {
			line:        ExtractedMovement{Date: "2026-02-03", Description: "ALUGUEL", Amount: -1503.90},
			invoices:    []OpenInvoiceForPayment{mercadoPago},
			expectedNil: true,
		},
		"income lines are never matched": {
			line:        ExtractedMovement{Date: "2026-02-03", Description: "PAGAMENTO FATURA ESTORNO", Amount: 1503.90},
			invoices:    []OpenInvoiceForPayment{mercadoPago},
			expectedNil: true,
		},
		"invalid date is not matched": {
			line:        ExtractedMovement{Date: "03/02/2026", Description: "PAGAMENTO FATURA", Amount: -1503.90},
			invoices:    []OpenInvoiceForPayment{mercadoPago},
			expectedNil: true,
		},
		"no open invoices means no match": {
			line:        ExtractedMovement{Date: "2026-02-03", Description: "PAGAMENTO FATURA", Amount: -1503.90},
			invoices:    nil,
			expectedNil: true,
		},
		"with two candidates the nearest due date wins": {
			line:         ExtractedMovement{Date: "2026-02-09", Description: "PAGAMENTO FATURA", Amount: -100.00},
			invoices:     []OpenInvoiceForPayment{mercadoPago, nubankNext, nubank},
			expectedID:   nubank.InvoiceID,
			expectedKind: InvoicePaymentKindAdvance,
			confidence:   InvoicePaymentMatchConfidenceMedium,
		},
		"the card name in the description wins over a nearer due date": {
			line:         ExtractedMovement{Date: "2026-02-03", Description: "PAGAMENTO FATURA NUBANK", Amount: -100.00},
			invoices:     []OpenInvoiceForPayment{mercadoPago, nubank},
			expectedID:   nubank.InvoiceID,
			expectedKind: InvoicePaymentKindAdvance,
			confidence:   InvoicePaymentMatchConfidenceMedium,
		},
		"an already covered invoice (zero remaining) is skipped": {
			line: ExtractedMovement{Date: "2026-02-03", Description: "PAGAMENTO FATURA", Amount: -100.00},
			invoices: []OpenInvoiceForPayment{{
				InvoiceID: uuid.New(), CreditCardID: uuid.New(), CreditCardName: "Zero",
				DueDate: d(2026, 2, 4), RemainingAmount: 0,
			}},
			expectedNil: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			match := SuggestInvoicePaymentMatch(tc.line, tc.invoices)

			if tc.expectedNil {
				assert.Nil(t, match)
				return
			}
			if assert.NotNil(t, match) {
				assert.Equal(t, tc.expectedID, match.InvoiceID)
				assert.Equal(t, tc.expectedKind, match.Kind)
				assert.Equal(t, tc.confidence, match.Confidence)
				assert.Equal(t, tc.confidence >= InvoicePaymentMatchConfidenceHigh, match.IsDecisionHighConfidence())
			}
		})
	}
}

func TestSuggestInvoicePaymentMatch_FillsDisplayFields(t *testing.T) {
	invoice := OpenInvoiceForPayment{
		InvoiceID: uuid.New(), CreditCardID: uuid.New(), CreditCardName: "Mercado Pago",
		DueDate: time.Date(2026, 2, 4, 0, 0, 0, 0, time.UTC), RemainingAmount: -1503.90,
	}

	match := SuggestInvoicePaymentMatch(
		ExtractedMovement{Date: "2026-01-11", Description: "PIX", Amount: -150, IsCardPayment: true},
		[]OpenInvoiceForPayment{invoice},
	)

	assert.Equal(t, &InvoicePaymentMatch{
		InvoiceID: invoice.InvoiceID, CreditCardID: invoice.CreditCardID, CreditCardName: "Mercado Pago",
		DueDate: "2026-02-04", RemainingAmount: -1503.90, Kind: InvoicePaymentKindAdvance, Confidence: 0.7,
	}, match)
}
