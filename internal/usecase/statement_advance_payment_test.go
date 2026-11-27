package usecase

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"personal-finance/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestStatementUseCase_Extract_InvoicePaymentLines(t *testing.T) {
	type (
		input struct {
			gatewayResult domain.StatementExtractResult
			creditCardID  *uuid.UUID
			card          *domain.CreditCard
		}
		expected struct {
			reasons        []string
			amounts        []float64
			advanceWarning string
			mismatch       bool
			paymentWarning bool
		}
	)

	creditCardID := uuid.New()
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
			invoiceUC.On("FindUnpaid").Return([]domain.Invoice{}, nil).Maybe()
			if tc.input.card != nil {
				ccRepo.On("FindByID", creditCardID).Return(*tc.input.card, nil)
				movRepo.On("FindInstallmentCandidatesByCreditCard", creditCardID, 12).Return(domain.MovementList{}, nil).Maybe()
			}

			result, err := uc.Extract(authedCtx(), rawBytes, "application/pdf", "", sourceType, tc.input.creditCardID)

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

func TestStatementUseCase_Extract_InvoicePaymentMatch(t *testing.T) {
	invoiceID := uuid.New()
	cardID := uuid.New()
	otherInvoiceID := uuid.New()
	otherCardID := uuid.New()

	openInvoice := func(id, card uuid.UUID, name string, due time.Time, remaining float64) domain.Invoice {
		return domain.Invoice{
			ID: &id, CreditCardID: &card, CreditCard: domain.CreditCard{Name: name},
			DueDate: due, Amount: remaining, RemainingAmount: remaining,
		}
	}
	mercadoPago := openInvoice(invoiceID, cardID, "Mercado Pago", time.Date(2026, 2, 4, 0, 0, 0, 0, time.UTC), -1503.90)
	nubank := openInvoice(otherInvoiceID, otherCardID, "Nubank", time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC), -800)

	tests := map[string]struct {
		lines        []domain.ExtractedMovement
		invoices     []domain.Invoice
		invoicesErr  error
		expectCalled bool
		sourceDoc    domain.DocumentType
		expected     func(t *testing.T, result domain.StatementExtractResult)
	}{
		"should pre-decide a settlement when the amount equals the remaining": {
			lines: []domain.ExtractedMovement{
				{Date: "2026-02-03", Description: "PAGAMENTO FATURA MERCADO PAGO", Amount: -1503.90},
				{Date: "2026-02-03", Description: "SUPERMERCADO", Amount: -50},
			},
			invoices:     []domain.Invoice{mercadoPago, nubank},
			expectCalled: true,
			expected: func(t *testing.T, result domain.StatementExtractResult) {
				first := result.Movements[0]
				assert.NotNil(t, first.InvoicePaymentMatch)
				assert.Equal(t, domain.InvoicePaymentKindSettlement, first.InvoicePaymentMatch.Kind)
				assert.Equal(t, "Mercado Pago", first.InvoicePaymentMatch.CreditCardName)
				assert.Equal(t, "2026-02-04", first.InvoicePaymentMatch.DueDate)
				assert.Equal(t, &domain.InvoicePaymentLink{InvoiceID: invoiceID, Kind: domain.InvoicePaymentKindSettlement}, first.InvoicePayment)
				assert.Nil(t, result.Movements[1].InvoicePaymentMatch)
				assert.Nil(t, result.Movements[1].InvoicePayment)
				assert.True(t, result.HasWarning(domain.WarningInvoicePaymentMatchFound))
			},
		},
		"should only suggest (not decide) an advance": {
			lines: []domain.ExtractedMovement{
				{Date: "2026-01-11", Description: "PIX MERCADO PAGO FATURA", Amount: -150, IsCardPayment: true},
			},
			invoices:     []domain.Invoice{mercadoPago},
			expectCalled: true,
			expected: func(t *testing.T, result domain.StatementExtractResult) {
				line := result.Movements[0]
				assert.NotNil(t, line.InvoicePaymentMatch)
				assert.Equal(t, domain.InvoicePaymentKindAdvance, line.InvoicePaymentMatch.Kind)
				assert.Equal(t, -1503.90, line.InvoicePaymentMatch.RemainingAmount)
				assert.Nil(t, line.InvoicePayment, "advance is a suggestion the user accepts")
				assert.True(t, result.HasWarning(domain.WarningInvoicePaymentMatchFound))
			},
		},
		"should not warn when no line matches": {
			lines:        []domain.ExtractedMovement{{Date: "2026-01-11", Description: "ALUGUEL", Amount: -1503.90}},
			invoices:     []domain.Invoice{mercadoPago},
			expectCalled: true,
			expected: func(t *testing.T, result domain.StatementExtractResult) {
				assert.Nil(t, result.Movements[0].InvoicePaymentMatch)
				assert.False(t, result.HasWarning(domain.WarningInvoicePaymentMatchFound))
			},
		},
		"should be a soft failure when invoices cannot be loaded": {
			lines:        []domain.ExtractedMovement{{Date: "2026-01-11", Description: "PAGAMENTO FATURA", Amount: -150}},
			invoicesErr:  errors.New("db down"),
			expectCalled: true,
			expected: func(t *testing.T, result domain.StatementExtractResult) {
				assert.Nil(t, result.Movements[0].InvoicePaymentMatch)
				assert.Len(t, result.Movements, 1)
			},
		},
		"should not look for invoices when the document is a card invoice": {
			lines:        []domain.ExtractedMovement{{Date: "2026-01-11", Description: "COMPRA", Amount: -150}},
			sourceDoc:    domain.DocInvoice,
			expectCalled: false,
			expected: func(t *testing.T, result domain.StatementExtractResult) {
				assert.Nil(t, result.Movements[0].InvoicePaymentMatch)
			},
		},
	}

	rawBytes := []byte("file-bytes")

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var (
				visionGw  = &MockStatementVisionGateway{}
				invoiceUC = &MockStatementInvoiceUseCase{}
				uc        = newStatementUseCaseWithInvoice(visionGw, &MockStatementClassificationGateway{}, &MockStatementMovementRepository{},
					&MockStatementCategoryRepository{}, invoiceUC, &MockInvoiceRepository{}, &MockStatementCreditCardRepository{}, &MockTransactionManager{})
			)
			docType := tc.sourceDoc
			if docType == "" {
				docType = domain.DocStatement
			}
			visionGw.On("ExtractMovements", rawBytes, "application/pdf", "statement").
				Return(domain.StatementExtractResult{DocumentType: docType, Confidence: 0.95, Movements: tc.lines}, nil)
			if tc.expectCalled {
				invoiceUC.On("FindUnpaid").Return(tc.invoices, tc.invoicesErr)
			}

			result, err := uc.Extract(authedCtx(), rawBytes, "application/pdf", "", "statement", nil)

			assert.NoError(t, err)
			tc.expected(t, result)
			invoiceUC.AssertExpectations(t)
		})
	}
}

func TestStatementUseCase_Confirm_InvoicePaymentLines(t *testing.T) {
	walletID := uuid.New()
	invoiceID := uuid.New()
	date := time.Date(2026, 1, 11, 0, 0, 0, 0, time.UTC)

	advanceLink := &domain.InvoicePaymentLink{InvoiceID: invoiceID, Kind: domain.InvoicePaymentKindAdvance}
	settlementLink := &domain.InvoicePaymentLink{InvoiceID: invoiceID, Kind: domain.InvoicePaymentKindSettlement}
	openInvoice := domain.Invoice{ID: &invoiceID, Amount: -1503.90}

	line := func(link *domain.InvoicePaymentLink, amount float64) domain.ExtractedMovement {
		return domain.ExtractedMovement{Date: "2026-01-11", Description: "PAGAMENTO FATURA", Amount: amount, InvoicePayment: link}
	}

	tests := map[string]struct {
		movement        domain.ExtractedMovement
		existingHashes  map[string]bool
		mockSetup       func(invoiceUC *MockStatementInvoiceUseCase)
		expectedCreated int
		expectedSkipped int
		expectedPayment int
		expectedErrors  int
		errorContains   string
	}{
		"advance creates an invoice payment and no expense": {
			movement: line(advanceLink, -150),
			mockSetup: func(invoiceUC *MockStatementInvoiceUseCase) {
				invoiceUC.On("FindByID", invoiceID).Return(openInvoice, nil)
				invoiceUC.On("AdvancePayment", invoiceID, mock.MatchedBy(func(in AdvancePaymentInput) bool {
					return in.Amount == -150 && *in.WalletID == walletID && in.PaymentDate.Equal(date) &&
						in.IdempotencyHash != nil && *in.IdempotencyHash != ""
				})).Return(openInvoice, nil)
			},
			expectedPayment: 1,
		},
		"settlement runs the pay rule with the line amount and date": {
			movement: line(settlementLink, -1503.90),
			mockSetup: func(invoiceUC *MockStatementInvoiceUseCase) {
				invoiceUC.On("FindByID", invoiceID).Return(openInvoice, nil)
				invoiceUC.On("PayInvoice", invoiceID, mock.MatchedBy(func(in PayInvoiceInput) bool {
					return in.WalletID == walletID && *in.Amount == -1503.90 && in.PaymentDate.Equal(date) &&
						in.IdempotencyHash != nil
				})).Return(openInvoice, nil)
			},
			expectedPayment: 1,
		},
		"positive amount on the line is normalized to a debit": {
			movement: line(advanceLink, 150),
			mockSetup: func(invoiceUC *MockStatementInvoiceUseCase) {
				invoiceUC.On("FindByID", invoiceID).Return(openInvoice, nil)
				invoiceUC.On("AdvancePayment", invoiceID, mock.MatchedBy(func(in AdvancePaymentInput) bool {
					return in.Amount == -150
				})).Return(openInvoice, nil)
			},
			expectedPayment: 1,
		},
		"paid invoice goes to errors and skipped, never to expense": {
			movement: line(advanceLink, -150),
			mockSetup: func(invoiceUC *MockStatementInvoiceUseCase) {
				paid := openInvoice
				paid.IsPaid = true
				invoiceUC.On("FindByID", invoiceID).Return(paid, nil)
			},
			expectedSkipped: 1,
			expectedErrors:  1,
			errorContains:   "invoice is already paid",
		},
		"advance not lower than the remaining is rejected": {
			movement: line(advanceLink, -2000),
			mockSetup: func(invoiceUC *MockStatementInvoiceUseCase) {
				invoiceUC.On("FindByID", invoiceID).Return(openInvoice, nil)
				invoiceUC.On("AdvancePayment", invoiceID, mock.Anything).
					Return(domain.Invoice{}, fmt.Errorf("advance invoice payment: %w", ErrAdvancePaymentSettlesInvoice))
			},
			expectedSkipped: 1,
			expectedErrors:  1,
			errorContains:   "not lower than the invoice remaining amount",
		},
		"settlement above the remaining is rejected as invalid amount": {
			movement: line(settlementLink, -2000),
			mockSetup: func(invoiceUC *MockStatementInvoiceUseCase) {
				invoiceUC.On("FindByID", invoiceID).Return(openInvoice, nil)
				invoiceUC.On("PayInvoice", invoiceID, mock.Anything).
					Return(domain.Invoice{}, domain.WrapInvalidInput(ErrInvalidPaymentAmount, "pay invoice"))
			},
			expectedSkipped: 1,
			expectedErrors:  1,
			errorContains:   "not valid for this invoice",
		},
		"invoice not found is rejected": {
			movement: line(advanceLink, -150),
			mockSetup: func(invoiceUC *MockStatementInvoiceUseCase) {
				invoiceUC.On("FindByID", invoiceID).Return(domain.Invoice{}, errors.New("not found"))
			},
			expectedSkipped: 1,
			expectedErrors:  1,
			errorContains:   "invoice not found",
		},
		"unknown kind is rejected without touching the invoice": {
			movement:        line(&domain.InvoicePaymentLink{InvoiceID: invoiceID, Kind: "other"}, -150),
			mockSetup:       func(invoiceUC *MockStatementInvoiceUseCase) {},
			expectedSkipped: 1,
			expectedErrors:  1,
			errorContains:   "invalid invoice payment kind",
		},
		"existing advance with same amount within 3 days is skipped silently": {
			movement: line(advanceLink, -150),
			mockSetup: func(invoiceUC *MockStatementInvoiceUseCase) {
				withAdvance := openInvoice
				withAdvance.AdvancePayments = []domain.InvoiceAdvancePayment{
					{ID: uuid.New(), PaymentDate: date.AddDate(0, 0, -3), Amount: -150},
				}
				invoiceUC.On("FindByID", invoiceID).Return(withAdvance, nil)
			},
			expectedSkipped: 1,
		},
		"existing advance more than 3 days away is not a duplicate": {
			movement: line(advanceLink, -150),
			mockSetup: func(invoiceUC *MockStatementInvoiceUseCase) {
				withAdvance := openInvoice
				withAdvance.AdvancePayments = []domain.InvoiceAdvancePayment{
					{ID: uuid.New(), PaymentDate: date.AddDate(0, 0, -4), Amount: -150},
				}
				invoiceUC.On("FindByID", invoiceID).Return(withAdvance, nil)
				invoiceUC.On("AdvancePayment", invoiceID, mock.Anything).Return(openInvoice, nil)
			},
			expectedPayment: 1,
		},
		"existing advance with a different amount is not a duplicate": {
			movement: line(advanceLink, -150),
			mockSetup: func(invoiceUC *MockStatementInvoiceUseCase) {
				withAdvance := openInvoice
				withAdvance.AdvancePayments = []domain.InvoiceAdvancePayment{
					{ID: uuid.New(), PaymentDate: date, Amount: -49},
				}
				invoiceUC.On("FindByID", invoiceID).Return(withAdvance, nil)
				invoiceUC.On("AdvancePayment", invoiceID, mock.Anything).Return(openInvoice, nil)
			},
			expectedPayment: 1,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var (
				movRepo   = &MockStatementMovementRepository{}
				invoiceUC = &MockStatementInvoiceUseCase{}
				uc        = newStatementUseCaseWithInvoice(&MockStatementVisionGateway{}, &MockStatementClassificationGateway{}, movRepo,
					&MockStatementCategoryRepository{}, invoiceUC, &MockInvoiceRepository{}, &MockStatementCreditCardRepository{}, &MockTransactionManager{})
			)
			existing := tc.existingHashes
			if existing == nil {
				existing = map[string]bool{}
			}
			movRepo.On("FindExistingHashes", "user-123", mock.Anything).Return(existing, nil)
			tc.mockSetup(invoiceUC)

			result, err := uc.Confirm(authedCtx(), domain.StatementConfirmInput{
				WalletID:  walletID,
				Movements: []domain.ExtractedMovement{tc.movement},
			})

			assert.NoError(t, err)
			assert.Equal(t, tc.expectedCreated, result.Created, "no expense may be created from an invoice payment line")
			assert.Equal(t, tc.expectedSkipped, result.Skipped)
			assert.Equal(t, tc.expectedPayment, result.InvoicePaymentsCreated)
			assert.Len(t, result.Errors, tc.expectedErrors)
			if tc.errorContains != "" {
				assert.Contains(t, result.Errors[0], tc.errorContains)
			}
			movRepo.AssertNotCalled(t, "Add", mock.Anything, mock.Anything)
			invoiceUC.AssertExpectations(t)
		})
	}
}

func TestStatementUseCase_Confirm_InvoicePaymentLine_Reimport(t *testing.T) {
	walletID := uuid.New()
	invoiceID := uuid.New()
	movement := domain.ExtractedMovement{
		Date: "2026-01-11", Description: "PAGAMENTO FATURA", Amount: -150,
		InvoicePayment: &domain.InvoicePaymentLink{InvoiceID: invoiceID, Kind: domain.InvoicePaymentKindAdvance},
	}

	movRepo := &MockStatementMovementRepository{}
	invoiceUC := &MockStatementInvoiceUseCase{}
	uc := newStatementUseCaseWithInvoice(&MockStatementVisionGateway{}, &MockStatementClassificationGateway{}, movRepo,
		&MockStatementCategoryRepository{}, invoiceUC, &MockInvoiceRepository{}, &MockStatementCreditCardRepository{}, &MockTransactionManager{})

	hash := domain.ComputeIdempotencyHash("user-123", walletID.String(), time.Date(2026, 1, 11, 0, 0, 0, 0, time.UTC), -150, "PAGAMENTO FATURA")
	movRepo.On("FindExistingHashes", "user-123", mock.Anything).Return(map[string]bool{hash: true}, nil)

	result, err := uc.Confirm(authedCtx(), domain.StatementConfirmInput{WalletID: walletID, Movements: []domain.ExtractedMovement{movement}})

	assert.NoError(t, err)
	assert.Equal(t, 1, result.Skipped)
	assert.Equal(t, 0, result.InvoicePaymentsCreated)
	assert.Empty(t, result.Errors)
	invoiceUC.AssertNotCalled(t, "FindByID", mock.Anything)
}
