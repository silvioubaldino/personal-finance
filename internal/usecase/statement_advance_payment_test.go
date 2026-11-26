package usecase

import (
	"testing"

	"personal-finance/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestStatementUseCase_Extract_InvoicePaymentLines(t *testing.T) {
	type (
		input struct {
			gatewayResult domain.StatementExtractResult
			creditCardID  *uuid.UUID
			card          *domain.CreditCard
		}
		expected struct {
			reasons        []string // exclusion_reason por item, na ordem
			amounts        []float64
			advanceWarning string // "" = sem warning
			mismatch       bool
			paymentWarning bool
		}
	)

	creditCardID := uuid.New()
	// Cartão do caso Mercado Pago (AYD-007@context): fecha dia 29, vence dia 4.
	mercadoPago := domain.CreditCard{ID: &creditCardID, ClosingDay: 29, DueDay: 4}

	str := func(s string) *string { return &s }
	f64 := func(v float64) *float64 { return &v }

	mercadoPagoItems := func() []domain.ExtractedMovement {
		return []domain.ExtractedMovement{
			{Date: "2025-12-31", Description: "COMPRA A", Amount: -1000.00},
			{Date: "2026-01-20", Description: "COMPRA B", Amount: -503.90},
			{Date: "2025-12-30", Description: "Pagamento da fatura de dezembro/2025", Amount: 178.44},
			{Date: "2026-01-11", Description: "Pagamento da fatura de janeiro/2026", Amount: 150.00},
			{Date: "2026-01-16", Description: "Pagamento da fatura de janeiro/2026", Amount: 49.00},
		}
	}

	tests := map[string]struct {
		input    input
		expected expected
	}{
		"should split previous invoice payment from advances by date using the credit card (Mercado Pago)": {
			input: input{
				creditCardID: &creditCardID,
				card:         &mercadoPago,
				gatewayResult: domain.StatementExtractResult{
					DocumentType: domain.DocInvoice, Confidence: 0.95,
					InvoiceMeta: &domain.InvoiceMeta{TotalAmount: f64(-1304.90)},
					Movements:   mercadoPagoItems(),
				},
			},
			expected: expected{
				reasons:        []string{"", "", "invoice_payment", "advance_payment", "advance_payment"},
				amounts:        []float64{-1000.00, -503.90, 178.44, 150.00, 49.00},
				advanceWarning: "199.00",
				paymentWarning: true,
			},
		},
		"should normalize payment lines to positive amounts": {
			input: input{
				creditCardID: &creditCardID,
				card:         &mercadoPago,
				gatewayResult: domain.StatementExtractResult{
					DocumentType: domain.DocInvoice, Confidence: 0.95,
					InvoiceMeta: &domain.InvoiceMeta{TotalAmount: f64(-1304.90)},
					Movements: []domain.ExtractedMovement{
						{Date: "2025-12-31", Description: "COMPRA A", Amount: -1503.90},
						{Date: "2026-01-11", Description: "PAGAMENTO ON LINE", Amount: -199.00},
					},
				},
			},
			expected: expected{
				reasons:        []string{"", "advance_payment"},
				amounts:        []float64{-1503.90, 199.00},
				advanceWarning: "199.00",
			},
		},
		"should fall back to invoice_meta dates when there is no credit card": {
			input: input{
				gatewayResult: domain.StatementExtractResult{
					DocumentType: domain.DocInvoice, Confidence: 0.95,
					InvoiceMeta: &domain.InvoiceMeta{
						TotalAmount: f64(-1304.90), ClosingDate: str("2026-01-29"), DueDate: str("2026-02-04"),
					},
					Movements: mercadoPagoItems(),
				},
			},
			expected: expected{
				reasons:        []string{"", "", "invoice_payment", "advance_payment", "advance_payment"},
				amounts:        []float64{-1000.00, -503.90, 178.44, 150.00, 49.00},
				advanceWarning: "199.00",
				paymentWarning: true,
			},
		},
		"should treat every payment as invoice_payment when there is no card and no invoice_meta dates": {
			input: input{
				gatewayResult: domain.StatementExtractResult{
					DocumentType: domain.DocInvoice, Confidence: 0.95,
					Movements: mercadoPagoItems(),
				},
			},
			expected: expected{
				reasons:        []string{"", "", "invoice_payment", "invoice_payment", "invoice_payment"},
				amounts:        []float64{-1000.00, -503.90, 178.44, 150.00, 49.00},
				paymentWarning: true,
			},
		},
		"should tie-break by total when the previous invoice was paid late": {
			input: input{
				creditCardID: &creditCardID,
				card:         &mercadoPago,
				gatewayResult: domain.StatementExtractResult{
					DocumentType: domain.DocInvoice, Confidence: 0.95,
					// total = só as compras: o pagamento de 10/01 (depois do vencimento de
					// 04/01) é da fatura anterior, paga com atraso.
					InvoiceMeta: &domain.InvoiceMeta{TotalAmount: f64(-1503.90)},
					Movements: []domain.ExtractedMovement{
						{Date: "2025-12-31", Description: "COMPRA A", Amount: -1503.90},
						{Date: "2026-01-10", Description: "Pagamento da fatura de dezembro/2025", Amount: 178.44},
					},
				},
			},
			expected: expected{
				reasons:        []string{"", "invoice_payment"},
				amounts:        []float64{-1503.90, 178.44},
				paymentWarning: true,
			},
		},
		"should keep date classification when it reconciles with the total": {
			input: input{
				creditCardID: &creditCardID,
				card:         &mercadoPago,
				gatewayResult: domain.StatementExtractResult{
					DocumentType: domain.DocInvoice, Confidence: 0.95,
					InvoiceMeta: &domain.InvoiceMeta{TotalAmount: f64(-1304.90)},
					Movements: []domain.ExtractedMovement{
						{Date: "2025-12-31", Description: "COMPRA A", Amount: -1503.90},
						{Date: "2026-01-11", Description: "Pagamento da fatura", Amount: 199.00},
					},
				},
			},
			expected: expected{
				reasons:        []string{"", "advance_payment"},
				amounts:        []float64{-1503.90, 199.00},
				advanceWarning: "199.00",
			},
		},
		"should still warn about total mismatch when neither classification reconciles": {
			input: input{
				creditCardID: &creditCardID,
				card:         &mercadoPago,
				gatewayResult: domain.StatementExtractResult{
					DocumentType: domain.DocInvoice, Confidence: 0.95,
					InvoiceMeta: &domain.InvoiceMeta{TotalAmount: f64(-9999.99)},
					Movements: []domain.ExtractedMovement{
						{Date: "2025-12-31", Description: "COMPRA A", Amount: -1503.90},
						{Date: "2026-01-11", Description: "Pagamento da fatura", Amount: 199.00},
					},
				},
			},
			expected: expected{
				reasons:        []string{"", "advance_payment"},
				amounts:        []float64{-1503.90, 199.00},
				advanceWarning: "199.00",
				mismatch:       true,
			},
		},
		"should not flag an invoice without payment lines": {
			input: input{
				creditCardID: &creditCardID,
				card:         &mercadoPago,
				gatewayResult: domain.StatementExtractResult{
					DocumentType: domain.DocInvoice, Confidence: 0.95,
					InvoiceMeta: &domain.InvoiceMeta{TotalAmount: f64(-1503.90)},
					Movements: []domain.ExtractedMovement{
						{Date: "2025-12-31", Description: "COMPRA A", Amount: -1503.90},
					},
				},
			},
			expected: expected{reasons: []string{""}, amounts: []float64{-1503.90}},
		},
		"should leave a bank statement untouched": {
			input: input{
				gatewayResult: domain.StatementExtractResult{
					DocumentType: domain.DocStatement, Confidence: 0.95,
					Movements: []domain.ExtractedMovement{
						{Date: "2026-01-11", Description: "PAGAMENTO FATURA", Amount: -150.00},
					},
				},
			},
			expected: expected{reasons: []string{""}, amounts: []float64{-150.00}},
		},
	}

	rawBytes := []byte("file-bytes")

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Arrange
			var (
				visionGw    = &MockStatementVisionGateway{}
				classGw     = &MockStatementClassificationGateway{}
				movRepo     = &MockStatementMovementRepository{}
				catRepo     = &MockStatementCategoryRepository{}
				invoiceUC   = &MockStatementInvoiceUseCase{}
				invoiceRepo = &MockInvoiceRepository{}
				ccRepo      = &MockStatementCreditCardRepository{}
				txManager   = &MockTransactionManager{}
				uc          = newStatementUseCaseWithInvoice(visionGw, classGw, movRepo, catRepo, invoiceUC, invoiceRepo, ccRepo, txManager)
			)
			sourceType := "invoice"
			if tc.input.gatewayResult.DocumentType == domain.DocStatement {
				sourceType = "statement"
			}
			visionGw.On("ExtractMovements", rawBytes, "application/pdf", sourceType).Return(tc.input.gatewayResult, nil)
			if tc.input.card != nil {
				ccRepo.On("FindByID", creditCardID).Return(*tc.input.card, nil)
				movRepo.On("FindInstallmentCandidatesByCreditCard", creditCardID, 12).Return(domain.MovementList{}, nil).Maybe()
			}

			// Act
			result, err := uc.Extract(authedCtx(), rawBytes, "application/pdf", "", sourceType, tc.input.creditCardID)

			// Assert
			assert.NoError(t, err)
			assert.Equal(t, tc.expected.reasons, exclusionReasonsOf(result))
			amounts := make([]float64, len(result.Movements))
			for i, m := range result.Movements {
				amounts[i] = m.Amount
			}
			assert.Equal(t, tc.expected.amounts, amounts)
			assert.Equal(t, tc.expected.advanceWarning, warningField(result, domain.WarningAdvancePaymentDetected, false))
			assert.Equal(t, tc.expected.advanceWarning != "", result.HasWarning(domain.WarningAdvancePaymentDetected))
			assert.Equal(t, tc.expected.mismatch, result.HasWarning(domain.WarningTotalAmountMismatch), "warnings: %v", result.Warnings)
			assert.Equal(t, tc.expected.paymentWarning, result.HasWarning(domain.WarningInvoicePaymentExcluded))
		})
	}
}
