package usecase

import (
	"context"
	"testing"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/plataform/authentication"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

func mustParseDate(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func newStatementUseCase(
	visionGw *MockStatementVisionGateway,
	classGw *MockStatementClassificationGateway,
	movRepo *MockStatementMovementRepository,
	catRepo *MockStatementCategoryRepository,
) *StatementUseCase {
	return NewStatementUseCase(visionGw, classGw, movRepo, catRepo, nil, nil, nil, nil, nil, nil)
}

func newStatementUseCaseWithInvoice(
	visionGw *MockStatementVisionGateway,
	classGw *MockStatementClassificationGateway,
	movRepo *MockStatementMovementRepository,
	catRepo *MockStatementCategoryRepository,
	invoiceUC *MockStatementInvoiceUseCase,
	invoiceRepo *MockInvoiceRepository,
	ccRepo *MockStatementCreditCardRepository,
	txManager *MockTransactionManager,
) *StatementUseCase {
	return NewStatementUseCase(visionGw, classGw, movRepo, catRepo, nil, nil, invoiceUC, invoiceRepo, ccRepo, txManager)
}

func authedCtx() context.Context {
	return context.WithValue(context.Background(), authentication.UserID, "user-123")
}

// --- Classify ---

func TestStatementUseCase_Classify(t *testing.T) {
	catID := uuid.New()
	subCatID := uuid.New()

	categories := []domain.Category{
		{ID: &catID, Description: "Supermercado"},
	}

	movements := []domain.ExtractedMovement{
		{Description: "SUPERMERCADO BOM PRECO", Amount: -150.0, Date: "2024-01-15"},
		{Description: "SPOTIFY", Amount: -29.90, Date: "2024-01-15"},
	}

	tests := map[string]struct {
		mockSetup       func(*MockStatementClassificationGateway, *MockStatementMovementRepository, *MockStatementCategoryRepository)
		expectedSources []string
		expectedCatIDs  []*uuid.UUID
		expectError     bool
	}{
		"all movements matched by history": {
			mockSetup: func(classGw *MockStatementClassificationGateway, movRepo *MockStatementMovementRepository, catRepo *MockStatementCategoryRepository) {
				catRepo.On("FindAll").Return(categories, nil)
				// both movements match history
				movRepo.On("FindRecentCategorizedByNormalizedDescription", "supermercado bom preco").
					Return(catID, subCatID, nil)
				movRepo.On("FindRecentCategorizedByNormalizedDescription", "spotify").
					Return(catID, uuid.UUID{}, nil)
				// classGw should NOT be called
			},
			expectedSources: []string{"history", "history"},
			expectedCatIDs:  []*uuid.UUID{&catID, &catID},
		},
		"all movements need AI classification": {
			mockSetup: func(classGw *MockStatementClassificationGateway, movRepo *MockStatementMovementRepository, catRepo *MockStatementCategoryRepository) {
				catRepo.On("FindAll").Return(categories, nil)
				movRepo.On("FindRecentCategorizedByNormalizedDescription", mock.Anything).
					Return(nil, nil, nil)
				classGw.On("ClassifyMovements", movements, categories).
					Return([]domain.CategorySuggestion{
						{Description: movements[0].Description, CategoryID: &catID, Confidence: 0.9, Source: "ai"},
						{Description: movements[1].Description, CategoryID: &catID, Confidence: 0.75, Source: "ai"},
					}, nil)
			},
			expectedSources: []string{"ai", "ai"},
			expectedCatIDs:  []*uuid.UUID{&catID, &catID},
		},
		"mixed: one history hit, one AI": {
			mockSetup: func(classGw *MockStatementClassificationGateway, movRepo *MockStatementMovementRepository, catRepo *MockStatementCategoryRepository) {
				catRepo.On("FindAll").Return(categories, nil)
				movRepo.On("FindRecentCategorizedByNormalizedDescription", "supermercado bom preco").
					Return(catID, subCatID, nil)
				movRepo.On("FindRecentCategorizedByNormalizedDescription", "spotify").
					Return(nil, nil, nil)
				classGw.On("ClassifyMovements", []domain.ExtractedMovement{movements[1]}, categories).
					Return([]domain.CategorySuggestion{
						{Description: movements[1].Description, CategoryID: &catID, Confidence: 0.8, Source: "ai"},
					}, nil)
			},
			expectedSources: []string{"history", "ai"},
			expectedCatIDs:  []*uuid.UUID{&catID, &catID},
		},
		"empty movements returns error": {
			mockSetup: func(classGw *MockStatementClassificationGateway, movRepo *MockStatementMovementRepository, catRepo *MockStatementCategoryRepository) {
			},
			expectError: true,
		},
		"unauthenticated context returns error": {
			mockSetup: func(classGw *MockStatementClassificationGateway, movRepo *MockStatementMovementRepository, catRepo *MockStatementCategoryRepository) {
			},
			expectError: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			visionGw := &MockStatementVisionGateway{}
			classGw := &MockStatementClassificationGateway{}
			movRepo := &MockStatementMovementRepository{}
			catRepo := &MockStatementCategoryRepository{}

			tc.mockSetup(classGw, movRepo, catRepo)

			uc := newStatementUseCase(visionGw, classGw, movRepo, catRepo)

			var input domain.StatementClassifyInput
			var ctx context.Context

			switch name {
			case "empty movements returns error":
				ctx = authedCtx()
				input = domain.StatementClassifyInput{Movements: []domain.ExtractedMovement{}}
			case "unauthenticated context returns error":
				ctx = context.Background()
				input = domain.StatementClassifyInput{Movements: movements}
			default:
				ctx = authedCtx()
				input = domain.StatementClassifyInput{Movements: movements}
			}

			result, err := uc.Classify(ctx, input)

			if tc.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Len(t, result.Suggestions, len(movements))

			for i, s := range result.Suggestions {
				assert.Equal(t, tc.expectedSources[i], s.Source)
				assert.Equal(t, tc.expectedCatIDs[i], s.CategoryID)
			}

			classGw.AssertExpectations(t)
			movRepo.AssertExpectations(t)
			catRepo.AssertExpectations(t)
		})
	}
}

// --- Extract ---

func TestStatementUseCase_Extract(t *testing.T) {
	rawBytes := []byte("raw-file-bytes")
	decryptedBytes := []byte("decrypted-bytes")
	extracted := domain.StatementExtractResult{
		Movements: []domain.ExtractedMovement{{Description: "PIX", Amount: -10, Date: "2024-01-15"}},
	}

	t.Run("pdf is decrypted before extraction", func(t *testing.T) {
		visionGw := &MockStatementVisionGateway{}
		decryptor := &MockStatementPDFDecryptor{}

		decryptor.On("Prepare", rawBytes, "s3cret").Return(decryptedBytes, nil)
		visionGw.On("ExtractMovements", decryptedBytes, "application/pdf", "").Return(extracted, nil)

		uc := NewStatementUseCase(visionGw, &MockStatementClassificationGateway{},
			&MockStatementMovementRepository{}, &MockStatementCategoryRepository{}, nil, decryptor, nil, nil, nil, nil)

		result, err := uc.Extract(authedCtx(), rawBytes, "application/pdf", "s3cret", "", nil)

		assert.NoError(t, err)
		assert.Equal(t, extracted, result)
		decryptor.AssertExpectations(t)
		visionGw.AssertExpectations(t)
	})

	t.Run("image skips decryption", func(t *testing.T) {
		visionGw := &MockStatementVisionGateway{}
		decryptor := &MockStatementPDFDecryptor{}

		visionGw.On("ExtractMovements", rawBytes, "image/png", "").Return(extracted, nil)

		uc := NewStatementUseCase(visionGw, &MockStatementClassificationGateway{},
			&MockStatementMovementRepository{}, &MockStatementCategoryRepository{}, nil, decryptor, nil, nil, nil, nil)

		result, err := uc.Extract(authedCtx(), rawBytes, "image/png", "", "", nil)

		assert.NoError(t, err)
		assert.Equal(t, extracted, result)
		decryptor.AssertNotCalled(t, "Prepare", mock.Anything, mock.Anything)
		visionGw.AssertExpectations(t)
	})

	for name, prepErr := range map[string]error{
		"password required propagates without calling vision": domain.ErrStatementPasswordRequired,
		"wrong password propagates without calling vision":    domain.ErrStatementWrongPassword,
	} {
		t.Run(name, func(t *testing.T) {
			visionGw := &MockStatementVisionGateway{}
			decryptor := &MockStatementPDFDecryptor{}

			decryptor.On("Prepare", rawBytes, "").Return([]byte(nil), prepErr)

			uc := NewStatementUseCase(visionGw, &MockStatementClassificationGateway{},
				&MockStatementMovementRepository{}, &MockStatementCategoryRepository{}, nil, decryptor, nil, nil, nil, nil)

			_, err := uc.Extract(authedCtx(), rawBytes, "application/pdf", "", "", nil)

			assert.ErrorIs(t, err, prepErr)
			decryptor.AssertExpectations(t)
			visionGw.AssertNotCalled(t, "ExtractMovements", mock.Anything, mock.Anything)
		})
	}
}

// --- Confirm ---

func TestStatementUseCase_Confirm(t *testing.T) {
	walletID := uuid.New()
	catID := uuid.New()
	uncategorizedID := uuid.MustParse(domain.UncategorizedCategoryID)
	uncategorizedIncomeID := uuid.MustParse(domain.UncategorizedIncomeCategoryID)
	recurrenceID := uuid.New()

	tests := map[string]struct {
		input           domain.StatementConfirmInput
		mockSetup       func(*MockStatementMovementRepository)
		expectedCreated int
		expectedSkipped int
		expectError     bool
	}{
		"saves movement with provided category_id": {
			input: domain.StatementConfirmInput{
				WalletID: walletID,
				Movements: []domain.ExtractedMovement{
					{Description: "SUPERMERCADO", Amount: -100.0, Date: "2024-01-15", CategoryID: &catID},
				},
			},
			mockSetup: func(movRepo *MockStatementMovementRepository) {
				movRepo.On("FindExistingHashes", "user-123", mock.Anything).Return(map[string]bool{}, nil)
				movRepo.On("Add", (*gorm.DB)(nil), mock.MatchedBy(func(m domain.Movement) bool {
					return m.CategoryID != nil && *m.CategoryID == catID
				})).Return(domain.Movement{}, nil)
			},
			expectedCreated: 1,
		},
		"saves movement as uncategorized when category_id is nil": {
			input: domain.StatementConfirmInput{
				WalletID: walletID,
				Movements: []domain.ExtractedMovement{
					{Description: "SPOTIFY", Amount: -30.0, Date: "2024-01-15"},
				},
			},
			mockSetup: func(movRepo *MockStatementMovementRepository) {
				movRepo.On("FindExistingHashes", "user-123", mock.Anything).Return(map[string]bool{}, nil)
				movRepo.On("Add", (*gorm.DB)(nil), mock.MatchedBy(func(m domain.Movement) bool {
					return m.CategoryID != nil && *m.CategoryID == uncategorizedID
				})).Return(domain.Movement{}, nil)
			},
			expectedCreated: 1,
		},
		"saves income as uncategorized income when category_id is nil": {
			input: domain.StatementConfirmInput{
				WalletID: walletID,
				Movements: []domain.ExtractedMovement{
					{Description: "DINHEIRO RETIRADO EMERGENCIA", Amount: 395.0, Date: "2024-01-15"},
				},
			},
			mockSetup: func(movRepo *MockStatementMovementRepository) {
				movRepo.On("FindExistingHashes", "user-123", mock.Anything).Return(map[string]bool{}, nil)
				movRepo.On("Add", (*gorm.DB)(nil), mock.MatchedBy(func(m domain.Movement) bool {
					return m.CategoryID != nil && *m.CategoryID == uncategorizedIncomeID
				})).Return(domain.Movement{}, nil)
			},
			expectedCreated: 1,
		},
		"saves zero amount in the expense fallback": {
			input: domain.StatementConfirmInput{
				WalletID: walletID,
				Movements: []domain.ExtractedMovement{
					{Description: "ESTORNO", Amount: 0, Date: "2024-01-15"},
				},
			},
			mockSetup: func(movRepo *MockStatementMovementRepository) {
				movRepo.On("FindExistingHashes", "user-123", mock.Anything).Return(map[string]bool{}, nil)
				movRepo.On("Add", (*gorm.DB)(nil), mock.MatchedBy(func(m domain.Movement) bool {
					return m.CategoryID != nil && *m.CategoryID == uncategorizedID
				})).Return(domain.Movement{}, nil)
			},
			expectedCreated: 1,
		},
		"provided category_id prevails over the income fallback": {
			input: domain.StatementConfirmInput{
				WalletID: walletID,
				Movements: []domain.ExtractedMovement{
					{Description: "RENDIMENTO", Amount: 500.0, Date: "2024-01-15", CategoryID: &catID},
				},
			},
			mockSetup: func(movRepo *MockStatementMovementRepository) {
				movRepo.On("FindExistingHashes", "user-123", mock.Anything).Return(map[string]bool{}, nil)
				movRepo.On("Add", (*gorm.DB)(nil), mock.MatchedBy(func(m domain.Movement) bool {
					return m.CategoryID != nil && *m.CategoryID == catID
				})).Return(domain.Movement{}, nil)
			},
			expectedCreated: 1,
		},
		"recurrent income without existing movement uses the income fallback": {
			input: domain.StatementConfirmInput{
				WalletID: walletID,
				Movements: []domain.ExtractedMovement{
					{Description: "ALUGUEL RECEBIDO", Amount: 1200.0, Date: "2024-01-15", RecurrenceID: &recurrenceID},
				},
			},
			mockSetup: func(movRepo *MockStatementMovementRepository) {
				movRepo.On("FindExistingHashes", "user-123", mock.Anything).Return(map[string]bool{}, nil)
				movRepo.On("FindByRecurrentIDAndMonth", recurrenceID, mustParseDate("2024-01-15")).
					Return(nil, nil)
				movRepo.On("Add", (*gorm.DB)(nil), mock.MatchedBy(func(m domain.Movement) bool {
					return m.CategoryID != nil && *m.CategoryID == uncategorizedIncomeID
				})).Return(domain.Movement{}, nil)
			},
			expectedCreated: 1,
		},
		"skips duplicate movement": {
			input: domain.StatementConfirmInput{
				WalletID: walletID,
				Movements: []domain.ExtractedMovement{
					{Description: "DUPLICATE", Amount: -50.0, Date: "2024-01-15"},
				},
			},
			mockSetup: func(movRepo *MockStatementMovementRepository) {
				existingHash := domain.ComputeIdempotencyHash(
					"user-123", walletID.String(),
					mustParseDate("2024-01-15"),
					-50.0, "DUPLICATE",
				)
				movRepo.On("FindExistingHashes", "user-123", mock.Anything).
					Return(map[string]bool{existingHash: true}, nil)
			},
			expectedSkipped: 1,
		},
		"empty movements returns error": {
			input: domain.StatementConfirmInput{
				WalletID:  walletID,
				Movements: []domain.ExtractedMovement{},
			},
			mockSetup:   func(movRepo *MockStatementMovementRepository) {},
			expectError: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			visionGw := &MockStatementVisionGateway{}
			classGw := &MockStatementClassificationGateway{}
			movRepo := &MockStatementMovementRepository{}
			catRepo := &MockStatementCategoryRepository{}

			tc.mockSetup(movRepo)

			uc := newStatementUseCase(visionGw, classGw, movRepo, catRepo)

			if tc.expectError {
				_, err := uc.Confirm(authedCtx(), tc.input)
				assert.Error(t, err)
				return
			}

			result, err := uc.Confirm(authedCtx(), tc.input)
			assert.NoError(t, err)
			if tc.expectedCreated > 0 {
				assert.Equal(t, tc.expectedCreated, result.Created)
			}
			if tc.expectedSkipped > 0 {
				assert.Equal(t, tc.expectedSkipped, result.Skipped)
			}

			movRepo.AssertExpectations(t)
		})
	}
}

func TestResolveCategoryID(t *testing.T) {
	expenseFallbackID := uuid.MustParse(domain.UncategorizedCategoryID)
	incomeFallbackID := uuid.MustParse(domain.UncategorizedIncomeCategoryID)
	chosenID := uuid.New()

	tests := map[string]struct {
		provided *uuid.UUID
		amount   float64
		expected uuid.UUID
	}{
		"provided category wins over positive amount": {provided: &chosenID, amount: 100, expected: chosenID},
		"provided category wins over negative amount": {provided: &chosenID, amount: -100, expected: chosenID},
		"positive amount falls back to income":        {amount: 100, expected: incomeFallbackID},
		"negative amount falls back to expense":       {amount: -100, expected: expenseFallbackID},
		"zero amount falls back to expense":           {amount: 0, expected: expenseFallbackID},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := resolveCategoryID(tc.provided, tc.amount, expenseFallbackID, incomeFallbackID)

			assert.Equal(t, tc.expected, got)
		})
	}
}

// --- Extract (new source_type / warning behavior) ---

func TestStatementUseCase_Extract_SourceType(t *testing.T) {
	type (
		input struct {
			sourceType string
		}
		expected struct {
			docType  domain.DocumentType
			warnType string
			err      error
		}
	)

	invoiceResult := domain.StatementExtractResult{
		DocumentType: domain.DocInvoice,
		Confidence:   0.95,
		Movements: []domain.ExtractedMovement{
			{Date: "2026-05-12", Description: "MERCADO LIVRE", Amount: -120.0, TypePayment: "credit_card"},
		},
	}

	statementResult := domain.StatementExtractResult{
		DocumentType: domain.DocStatement,
		Confidence:   0.92,
		Movements: []domain.ExtractedMovement{
			{Date: "2026-05-12", Description: "PIX FULANO", Amount: -150.0, TypePayment: "pix"},
		},
	}

	rawBytes := []byte("file-bytes")

	tests := map[string]struct {
		// input
		input input
		// mocks
		mockSetup func(*MockStatementVisionGateway)
		// expected
		expected expected
	}{
		"should return invoice document_type when source_type=invoice and IA agrees": {
			input: input{sourceType: "invoice"},
			mockSetup: func(gw *MockStatementVisionGateway) {
				gw.On("ExtractMovements", rawBytes, "image/png", "invoice").Return(invoiceResult, nil)
			},
			expected: expected{
				docType:  domain.DocInvoice,
				warnType: "",
				err:      nil,
			},
		},
		"should add document_type_mismatch warning when source_type=invoice but IA detects statement": {
			input: input{sourceType: "invoice"},
			mockSetup: func(gw *MockStatementVisionGateway) {
				gw.On("ExtractMovements", rawBytes, "image/png", "invoice").Return(statementResult, nil)
			},
			expected: expected{
				docType:  domain.DocStatement,
				warnType: "document_type_mismatch",
				err:      nil,
			},
		},
		"should return detected type when source_type is absent (auto-detect)": {
			input: input{sourceType: ""},
			mockSetup: func(gw *MockStatementVisionGateway) {
				gw.On("ExtractMovements", rawBytes, "image/png", "").Return(invoiceResult, nil)
			},
			expected: expected{
				docType:  domain.DocInvoice,
				warnType: "",
				err:      nil,
			},
		},
		"should return document_type=unknown with low_confidence warning when gateway returns ErrStatementNotAStatement": {
			input: input{sourceType: ""},
			mockSetup: func(gw *MockStatementVisionGateway) {
				gw.On("ExtractMovements", rawBytes, "image/png", "").Return(domain.StatementExtractResult{}, domain.ErrStatementNotAStatement)
			},
			expected: expected{
				docType:  domain.DocUnknown,
				warnType: "low_confidence",
				err:      nil,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Arrange
			var (
				visionGw = &MockStatementVisionGateway{}
				classGw  = &MockStatementClassificationGateway{}
				movRepo  = &MockStatementMovementRepository{}
				catRepo  = &MockStatementCategoryRepository{}
				uc       = NewStatementUseCase(visionGw, classGw, movRepo, catRepo, nil, nil, nil, nil, nil, nil)
			)
			defer visionGw.AssertExpectations(t)
			tc.mockSetup(visionGw)

			// Act
			result, err := uc.Extract(authedCtx(), rawBytes, "image/png", "", tc.input.sourceType, nil)

			// Assert
			assert.ErrorIs(t, err, tc.expected.err)
			assert.Equal(t, tc.expected.docType, result.DocumentType)
			if tc.expected.warnType != "" {
				found := false
				for _, w := range result.Warnings {
					if w.Type == tc.expected.warnType {
						found = true
						break
					}
				}
				assert.True(t, found, "expected warning type %q not found in %v", tc.expected.warnType, result.Warnings)
			}
		})
	}
}

// --- Itens que não pertencem à fatura (AYD-004) ---

func TestStatementUseCase_Extract_ItemsNotBelongingToInvoice(t *testing.T) {
	type (
		input struct {
			gatewayResult domain.StatementExtractResult
		}
		expected struct {
			excludedDescriptions []string
			warnTypes            []string
			mismatchExpected     string
			mismatchDetected     string
		}
	)

	total := -6035.06
	positiveTotal := 6035.06
	wrongTotal := -9999.99

	tests := map[string]struct {
		// input
		input input
		// expected
		expected expected
	}{
		"should mark the payment line and warn, leaving it in the list": {
			input: input{gatewayResult: domain.StatementExtractResult{
				DocumentType: domain.DocInvoice,
				Confidence:   0.95,
				Movements: []domain.ExtractedMovement{
					{Date: "2026-07-03", Description: "MERCADINHO PIRATININGA", Amount: -92.65},
					{Date: "2026-07-07", Description: "PAGAMENTO ON LINE", Amount: -5212.59},
				},
			}},
			expected: expected{
				excludedDescriptions: []string{"PAGAMENTO ON LINE"},
				warnTypes:            []string{domain.WarningInvoicePaymentExcluded},
			},
		},
		"should not warn about the total when the remaining items match it": {
			input: input{gatewayResult: domain.StatementExtractResult{
				DocumentType: domain.DocInvoice,
				Confidence:   0.95,
				InvoiceMeta:  &domain.InvoiceMeta{TotalAmount: &total},
				Movements: []domain.ExtractedMovement{
					{Date: "2026-07-03", Description: "COMPRA A", Amount: -6035.06},
					{Date: "2026-07-07", Description: "PAGAMENTO ON LINE", Amount: -5212.59},
				},
			}},
			expected: expected{
				excludedDescriptions: []string{"PAGAMENTO ON LINE"},
				warnTypes:            []string{domain.WarningInvoicePaymentExcluded},
			},
		},
		"should compare magnitudes when the model reports the total with the opposite sign": {
			input: input{gatewayResult: domain.StatementExtractResult{
				DocumentType: domain.DocInvoice,
				Confidence:   0.95,
				InvoiceMeta:  &domain.InvoiceMeta{TotalAmount: &positiveTotal},
				Movements: []domain.ExtractedMovement{
					{Date: "2026-07-03", Description: "COMPRA A", Amount: -6035.06},
				},
			}},
			expected: expected{},
		},
		"should warn when the sum diverges from the declared total": {
			input: input{gatewayResult: domain.StatementExtractResult{
				DocumentType: domain.DocInvoice,
				Confidence:   0.95,
				InvoiceMeta:  &domain.InvoiceMeta{TotalAmount: &wrongTotal},
				Movements: []domain.ExtractedMovement{
					{Date: "2026-07-03", Description: "COMPRA A", Amount: -100.00},
				},
			}},
			expected: expected{
				warnTypes:        []string{domain.WarningTotalAmountMismatch},
				mismatchExpected: "-9999.99",
				mismatchDetected: "-100.00",
			},
		},
		"should leave a bank statement untouched": {
			input: input{gatewayResult: domain.StatementExtractResult{
				DocumentType: domain.DocStatement,
				Confidence:   0.95,
				Movements: []domain.ExtractedMovement{
					{Date: "2026-07-07", Description: "PAGAMENTO ON LINE", Amount: -5212.59},
				},
			}},
			expected: expected{},
		},
	}

	rawBytes := []byte("file-bytes")

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Arrange
			var (
				visionGw = &MockStatementVisionGateway{}
				classGw  = &MockStatementClassificationGateway{}
				movRepo  = &MockStatementMovementRepository{}
				catRepo  = &MockStatementCategoryRepository{}
				uc       = NewStatementUseCase(visionGw, classGw, movRepo, catRepo, nil, nil, nil, nil, nil, nil)
			)
			defer visionGw.AssertExpectations(t)
			visionGw.On("ExtractMovements", rawBytes, "application/pdf", "invoice").
				Return(tc.input.gatewayResult, nil)

			// Act
			result, err := uc.Extract(authedCtx(), rawBytes, "application/pdf", "", "invoice", nil)

			// Assert
			assert.ErrorIs(t, err, nil)
			assert.Equal(t, len(tc.input.gatewayResult.Movements), len(result.Movements),
				"no movement may be removed from the list")
			assert.Equal(t, tc.expected.excludedDescriptions, excludedDescriptionsOf(result))
			for _, want := range tc.expected.warnTypes {
				assert.True(t, result.HasWarning(want), "expected warning %q in %v", want, result.Warnings)
			}
			assert.Equal(t, tc.expected.mismatchExpected, warningField(result, domain.WarningTotalAmountMismatch, true))
			assert.Equal(t, tc.expected.mismatchDetected, warningField(result, domain.WarningTotalAmountMismatch, false))
		})
	}
}

// excludedDescriptionsOf devolve as descrições dos itens marcados como fora da fatura.
func excludedDescriptionsOf(r domain.StatementExtractResult) []string {
	var out []string
	for _, m := range r.Movements {
		if m.Excluded && m.ExclusionReason == domain.ExclusionReasonInvoicePayment {
			out = append(out, m.Description)
		}
	}
	return out
}

// warningField devolve Expected (expected=true) ou Detected do warning informado.
func warningField(r domain.StatementExtractResult, warningType string, expected bool) string {
	for _, w := range r.Warnings {
		if w.Type != warningType {
			continue
		}
		if expected {
			return w.Expected
		}
		return w.Detected
	}
	return ""
}

// --- ConfirmInvoice ---

func TestStatementUseCase_ConfirmInvoice(t *testing.T) {
	creditCardID := uuid.New()
	invoiceID := uuid.New()
	catID := uuid.New()
	walletID := uuid.New()
	uncategorizedID := uuid.MustParse(domain.UncategorizedCategoryID)

	// Cartão com carteira default (exigida no caminho de fatura), fechamento no
	// dia 3 e limite folgado — o suficiente para nenhum item do teste esbarrar
	// nas validações de limite/competência, que têm cenários próprios.
	fixtureCreditCard := domain.CreditCard{
		ID:              &creditCardID,
		ClosingDay:      3,
		DueDay:          10,
		CreditLimit:     100000,
		DefaultWalletID: &walletID,
	}

	fixtureInvoice := domain.Invoice{
		ID:     &invoiceID,
		IsPaid: false,
	}

	paidInvoice := domain.Invoice{
		ID:     &invoiceID,
		IsPaid: true,
	}

	type (
		input struct {
			payload domain.InvoiceConfirmInput
		}
		expected struct {
			created int
			skipped int
			err     error
		}
	)

	tests := map[string]struct {
		// input
		input input
		// mocks
		mockSetup func(*MockStatementMovementRepository, *MockStatementInvoiceUseCase, *MockInvoiceRepository, *MockStatementCreditCardRepository, *MockTransactionManager)
		// expected
		expected expected
	}{
		"should create movement with TypePayment=credit_card and IsPaid=false": {
			input: input{
				payload: domain.InvoiceConfirmInput{
					CreditCardID: creditCardID,
					Movements: []domain.ExtractedMovement{
						{Date: "2026-05-12", Description: "NETFLIX", Amount: -55.90, CategoryID: &catID},
					},
				},
			},
			mockSetup: func(movRepo *MockStatementMovementRepository, invoiceUC *MockStatementInvoiceUseCase, invoiceRepo *MockInvoiceRepository, ccRepo *MockStatementCreditCardRepository, txManager *MockTransactionManager) {
				ccRepo.On("FindByID", creditCardID).Return(fixtureCreditCard, nil)
				movRepo.On("FindExistingHashes", "user-123", mock.Anything).Return(map[string]bool{}, nil)
				invoiceUC.On("FindOrCreateInvoiceForMovement", (*uuid.UUID)(nil), &creditCardID, mustParseDate("2026-05-12")).
					Return(fixtureInvoice, nil)
				movRepo.On("Add", (*gorm.DB)(nil), mock.MatchedBy(func(m domain.Movement) bool {
					return m.TypePayment == domain.TypePaymentCreditCard && !m.IsPaid && m.CategoryID != nil && *m.CategoryID == catID
				})).Return(domain.Movement{}, nil)
				txManager.On("WithTransaction", mock.Anything).Return(nil)
				invoiceRepo.On("UpdateAmount", (*gorm.DB)(nil), invoiceID, -55.90).Return(fixtureInvoice, nil)
				ccRepo.On("UpdateLimitDelta", (*gorm.DB)(nil), creditCardID, -55.90).Return(domain.CreditCard{}, nil)
			},
			expected: expected{created: 1, skipped: 0, err: nil},
		},
		"should use uncategorized category when category_id is nil": {
			input: input{
				payload: domain.InvoiceConfirmInput{
					CreditCardID: creditCardID,
					Movements: []domain.ExtractedMovement{
						{Date: "2026-05-12", Description: "SPOTIFY", Amount: -29.90},
					},
				},
			},
			mockSetup: func(movRepo *MockStatementMovementRepository, invoiceUC *MockStatementInvoiceUseCase, invoiceRepo *MockInvoiceRepository, ccRepo *MockStatementCreditCardRepository, txManager *MockTransactionManager) {
				ccRepo.On("FindByID", creditCardID).Return(fixtureCreditCard, nil)
				movRepo.On("FindExistingHashes", "user-123", mock.Anything).Return(map[string]bool{}, nil)
				invoiceUC.On("FindOrCreateInvoiceForMovement", (*uuid.UUID)(nil), &creditCardID, mustParseDate("2026-05-12")).
					Return(fixtureInvoice, nil)
				movRepo.On("Add", (*gorm.DB)(nil), mock.MatchedBy(func(m domain.Movement) bool {
					return m.CategoryID != nil && *m.CategoryID == uncategorizedID
				})).Return(domain.Movement{}, nil)
				txManager.On("WithTransaction", mock.Anything).Return(nil)
				invoiceRepo.On("UpdateAmount", (*gorm.DB)(nil), invoiceID, -29.90).Return(fixtureInvoice, nil)
				ccRepo.On("UpdateLimitDelta", (*gorm.DB)(nil), creditCardID, -29.90).Return(domain.CreditCard{}, nil)
			},
			expected: expected{created: 1, skipped: 0, err: nil},
		},
		"should skip duplicate movement scoped by credit_card_id": {
			input: input{
				payload: domain.InvoiceConfirmInput{
					CreditCardID: creditCardID,
					Movements: []domain.ExtractedMovement{
						{Date: "2026-05-12", Description: "DUPLICATE", Amount: -50.0},
					},
				},
			},
			mockSetup: func(movRepo *MockStatementMovementRepository, invoiceUC *MockStatementInvoiceUseCase, invoiceRepo *MockInvoiceRepository, ccRepo *MockStatementCreditCardRepository, txManager *MockTransactionManager) {
				ccRepo.On("FindByID", creditCardID).Return(fixtureCreditCard, nil)
				existingHash := domain.ComputeIdempotencyHash(
					"user-123", creditCardID.String(),
					mustParseDate("2026-05-12"),
					-50.0, "DUPLICATE",
				)
				movRepo.On("FindExistingHashes", "user-123", mock.Anything).
					Return(map[string]bool{existingHash: true}, nil)
			},
			expected: expected{created: 0, skipped: 1, err: nil},
		},
		"should return ErrInvoiceAlreadyPaid when target invoice is paid": {
			input: input{
				payload: domain.InvoiceConfirmInput{
					CreditCardID: creditCardID,
					Movements: []domain.ExtractedMovement{
						{Date: "2026-05-12", Description: "PAID INVOICE ITEM", Amount: -100.0},
					},
				},
			},
			mockSetup: func(movRepo *MockStatementMovementRepository, invoiceUC *MockStatementInvoiceUseCase, invoiceRepo *MockInvoiceRepository, ccRepo *MockStatementCreditCardRepository, txManager *MockTransactionManager) {
				ccRepo.On("FindByID", creditCardID).Return(fixtureCreditCard, nil)
				movRepo.On("FindExistingHashes", "user-123", mock.Anything).Return(map[string]bool{}, nil)
				invoiceUC.On("FindOrCreateInvoiceForMovement", (*uuid.UUID)(nil), &creditCardID, mustParseDate("2026-05-12")).
					Return(paidInvoice, nil)
			},
			expected: expected{created: 0, skipped: 0, err: ErrInvoiceAlreadyPaid},
		},
		"should return error when credit_card not found": {
			input: input{
				payload: domain.InvoiceConfirmInput{
					CreditCardID: creditCardID,
					Movements: []domain.ExtractedMovement{
						{Date: "2026-05-12", Description: "ITEM", Amount: -10.0},
					},
				},
			},
			mockSetup: func(movRepo *MockStatementMovementRepository, invoiceUC *MockStatementInvoiceUseCase, invoiceRepo *MockInvoiceRepository, ccRepo *MockStatementCreditCardRepository, txManager *MockTransactionManager) {
				ccRepo.On("FindByID", creditCardID).Return(domain.CreditCard{}, assert.AnError)
			},
			expected: expected{err: assert.AnError},
		},
		"should skip the previous invoice payment line even when the client sends it unmarked": {
			input: input{
				payload: domain.InvoiceConfirmInput{
					CreditCardID: creditCardID,
					Movements: []domain.ExtractedMovement{
						{Date: "2026-07-07", Description: "PAGAMENTO ON LINE", Amount: -5212.59},
					},
				},
			},
			mockSetup: func(movRepo *MockStatementMovementRepository, invoiceUC *MockStatementInvoiceUseCase, invoiceRepo *MockInvoiceRepository, ccRepo *MockStatementCreditCardRepository, txManager *MockTransactionManager) {
				ccRepo.On("FindByID", creditCardID).Return(fixtureCreditCard, nil)
				movRepo.On("FindExistingHashes", "user-123", mock.Anything).Return(map[string]bool{}, nil)
			},
			expected: expected{created: 0, skipped: 1, err: nil},
		},
		"should skip a movement flagged as excluded by the extract": {
			input: input{
				payload: domain.InvoiceConfirmInput{
					CreditCardID: creditCardID,
					Movements: []domain.ExtractedMovement{
						{
							Date: "2026-07-07", Description: "QUALQUER COISA", Amount: -100.0,
							Excluded: true, ExclusionReason: domain.ExclusionReasonInvoicePayment,
						},
					},
				},
			},
			mockSetup: func(movRepo *MockStatementMovementRepository, invoiceUC *MockStatementInvoiceUseCase, invoiceRepo *MockInvoiceRepository, ccRepo *MockStatementCreditCardRepository, txManager *MockTransactionManager) {
				ccRepo.On("FindByID", creditCardID).Return(fixtureCreditCard, nil)
				movRepo.On("FindExistingHashes", "user-123", mock.Anything).Return(map[string]bool{}, nil)
			},
			expected: expected{created: 0, skipped: 1, err: nil},
		},
		"should return error when movements is empty": {
			input: input{
				payload: domain.InvoiceConfirmInput{
					CreditCardID: creditCardID,
					Movements:    []domain.ExtractedMovement{},
				},
			},
			mockSetup: func(movRepo *MockStatementMovementRepository, invoiceUC *MockStatementInvoiceUseCase, invoiceRepo *MockInvoiceRepository, ccRepo *MockStatementCreditCardRepository, txManager *MockTransactionManager) {
			},
			expected: expected{err: domain.ErrInvalidInput},
		},
		"should return error for unauthenticated context": {
			input: input{
				payload: domain.InvoiceConfirmInput{
					CreditCardID: creditCardID,
					Movements: []domain.ExtractedMovement{
						{Date: "2026-05-12", Description: "ITEM", Amount: -10.0},
					},
				},
			},
			mockSetup: func(movRepo *MockStatementMovementRepository, invoiceUC *MockStatementInvoiceUseCase, invoiceRepo *MockInvoiceRepository, ccRepo *MockStatementCreditCardRepository, txManager *MockTransactionManager) {
			},
			expected: expected{err: domain.ErrUnauthorized},
		},
		"should generate installment series for movement with installment data": {
			input: input{
				payload: domain.InvoiceConfirmInput{
					CreditCardID: creditCardID,
					Movements: []domain.ExtractedMovement{
						{
							Date:              "2026-05-12",
							Description:       "MERCADO LIVRE PARCELA 03/12",
							Amount:            -120.0,
							InstallmentNumber: func() *int { n := 3; return &n }(),
							TotalInstallments: func() *int { n := 12; return &n }(),
						},
					},
				},
			},
			mockSetup: func(movRepo *MockStatementMovementRepository, invoiceUC *MockStatementInvoiceUseCase, invoiceRepo *MockInvoiceRepository, ccRepo *MockStatementCreditCardRepository, txManager *MockTransactionManager) {
				ccRepo.On("FindByID", creditCardID).Return(fixtureCreditCard, nil)
				movRepo.On("FindExistingHashes", "user-123", mock.Anything).Return(map[string]bool{}, nil)
				// 10 installments generated (3..12 = 10 remaining including current)
				invoiceUC.On("FindOrCreateInvoiceForMovement", (*uuid.UUID)(nil), &creditCardID, mock.Anything).
					Return(fixtureInvoice, nil)
				movRepo.On("Add", (*gorm.DB)(nil), mock.MatchedBy(func(m domain.Movement) bool {
					return m.TypePayment == domain.TypePaymentCreditCard
				})).Return(domain.Movement{}, nil)
				// Série inteira numa transação só: as 10 parcelas caem na mesma fatura
				// (o mock devolve sempre fixtureInvoice), então os deltas se somam num
				// único update de total e num único ajuste de limite.
				txManager.On("WithTransaction", mock.Anything).Return(nil)
				invoiceRepo.On("UpdateAmount", (*gorm.DB)(nil), invoiceID, -1200.0).Return(fixtureInvoice, nil)
				ccRepo.On("UpdateLimitDelta", (*gorm.DB)(nil), creditCardID, -1200.0).Return(domain.CreditCard{}, nil)
			},
			expected: expected{created: 10, skipped: 0, err: nil},
		},
		"should skip all installments on re-import (dedup must cover full series, not just installment #1)": {
			input: input{
				payload: domain.InvoiceConfirmInput{
					CreditCardID: creditCardID,
					Movements: []domain.ExtractedMovement{
						{
							Date:              "2026-05-12",
							Description:       "MERCADO LIVRE PARCELA 03/12",
							Amount:            -120.0,
							InstallmentNumber: func() *int { n := 3; return &n }(),
							TotalInstallments: func() *int { n := 12; return &n }(),
						},
					},
				},
			},
			mockSetup: func(movRepo *MockStatementMovementRepository, invoiceUC *MockStatementInvoiceUseCase, invoiceRepo *MockInvoiceRepository, ccRepo *MockStatementCreditCardRepository, txManager *MockTransactionManager) {
				ccRepo.On("FindByID", creditCardID).Return(fixtureCreditCard, nil)

				// Simula que todas as 10 parcelas da série (3..12) já foram importadas
				// anteriormente: cada uma tem seu próprio hash de idempotência, calculado
				// com a data daquela parcela específica.
				existingHashes := map[string]bool{}
				baseDate := mustParseDate("2026-05-12")
				for i := 0; i < 10; i++ {
					instDate := baseDate.AddDate(0, i, 0)
					h := domain.ComputeIdempotencyHash(
						"user-123", creditCardID.String(), instDate, -120.0, "MERCADO LIVRE PARCELA 03/12",
					)
					existingHashes[h] = true
				}
				movRepo.On("FindExistingHashes", "user-123", mock.Anything).Return(existingHashes, nil)
				// Nenhuma chamada a FindOrCreateInvoiceForMovement/Add/UpdateAmount/UpdateLimitDelta
				// deve ocorrer: toda a série já existe.
			},
			expected: expected{created: 0, skipped: 10, err: nil},
		},
		"should skip only the already-imported installments, creating the remaining ones": {
			input: input{
				payload: domain.InvoiceConfirmInput{
					CreditCardID: creditCardID,
					Movements: []domain.ExtractedMovement{
						{
							Date:              "2026-05-12",
							Description:       "MERCADO LIVRE PARCELA 03/12",
							Amount:            -120.0,
							InstallmentNumber: func() *int { n := 3; return &n }(),
							TotalInstallments: func() *int { n := 12; return &n }(),
						},
					},
				},
			},
			mockSetup: func(movRepo *MockStatementMovementRepository, invoiceUC *MockStatementInvoiceUseCase, invoiceRepo *MockInvoiceRepository, ccRepo *MockStatementCreditCardRepository, txManager *MockTransactionManager) {
				ccRepo.On("FindByID", creditCardID).Return(fixtureCreditCard, nil)

				// Apenas a primeira parcela (3/12) NÃO está nos hashes existentes, então o
				// pré-check no topo do loop deixa passar; dentro da série, as parcelas 4 e 5
				// (índices 1 e 2 da série gerada) já existem e devem ser puladas
				// individualmente pelo dedup por-parcela, enquanto as demais são criadas.
				existingHashes := map[string]bool{}
				baseDate := mustParseDate("2026-05-12")
				for _, i := range []int{1, 2} {
					instDate := baseDate.AddDate(0, i, 0)
					h := domain.ComputeIdempotencyHash(
						"user-123", creditCardID.String(), instDate, -120.0, "MERCADO LIVRE PARCELA 03/12",
					)
					existingHashes[h] = true
				}
				movRepo.On("FindExistingHashes", "user-123", mock.Anything).Return(existingHashes, nil)
				invoiceUC.On("FindOrCreateInvoiceForMovement", (*uuid.UUID)(nil), &creditCardID, mock.Anything).
					Return(fixtureInvoice, nil)
				movRepo.On("Add", (*gorm.DB)(nil), mock.MatchedBy(func(m domain.Movement) bool {
					return m.TypePayment == domain.TypePaymentCreditCard
				})).Return(domain.Movement{}, nil)
				txManager.On("WithTransaction", mock.Anything).Return(nil)
				// 8 parcelas gravadas × -120,00 na mesma fatura.
				invoiceRepo.On("UpdateAmount", (*gorm.DB)(nil), invoiceID, -960.0).Return(fixtureInvoice, nil)
				ccRepo.On("UpdateLimitDelta", (*gorm.DB)(nil), creditCardID, -960.0).Return(domain.CreditCard{}, nil)
			},
			// 10 parcelas no total (3..12); 2 já existem (puladas), 8 são criadas.
			expected: expected{created: 8, skipped: 2, err: nil},
		},
		"should not touch invoice total nor card limit when the movement fails to persist": {
			input: input{
				payload: domain.InvoiceConfirmInput{
					CreditCardID: creditCardID,
					Movements: []domain.ExtractedMovement{
						{Date: "2026-05-12", Description: "NETFLIX", Amount: -55.90, CategoryID: &catID},
					},
				},
			},
			mockSetup: func(movRepo *MockStatementMovementRepository, invoiceUC *MockStatementInvoiceUseCase, invoiceRepo *MockInvoiceRepository, ccRepo *MockStatementCreditCardRepository, txManager *MockTransactionManager) {
				ccRepo.On("FindByID", creditCardID).Return(fixtureCreditCard, nil)
				movRepo.On("FindExistingHashes", "user-123", mock.Anything).Return(map[string]bool{}, nil)
				invoiceUC.On("FindOrCreateInvoiceForMovement", (*uuid.UUID)(nil), &creditCardID, mustParseDate("2026-05-12")).
					Return(fixtureInvoice, nil)
				txManager.On("WithTransaction", mock.Anything).Return(nil)
				movRepo.On("Add", (*gorm.DB)(nil), mock.Anything).Return(domain.Movement{}, assert.AnError)
				// Nenhuma expectativa de UpdateAmount/UpdateLimitDelta: a transação aborta no
				// Add, e chamá-las faria o mock entrar em pânico por retorno não configurado.
			},
			expected: expected{created: 0, skipped: 1, err: nil},
		},
		"should discard the whole installment series when one installment fails to persist": {
			input: input{
				payload: domain.InvoiceConfirmInput{
					CreditCardID: creditCardID,
					Movements: []domain.ExtractedMovement{
						{
							Date:              "2026-05-12",
							Description:       "MERCADO LIVRE PARCELA 03/12",
							Amount:            -120.0,
							InstallmentNumber: func() *int { n := 3; return &n }(),
							TotalInstallments: func() *int { n := 12; return &n }(),
						},
					},
				},
			},
			mockSetup: func(movRepo *MockStatementMovementRepository, invoiceUC *MockStatementInvoiceUseCase, invoiceRepo *MockInvoiceRepository, ccRepo *MockStatementCreditCardRepository, txManager *MockTransactionManager) {
				ccRepo.On("FindByID", creditCardID).Return(fixtureCreditCard, nil)
				movRepo.On("FindExistingHashes", "user-123", mock.Anything).Return(map[string]bool{}, nil)
				invoiceUC.On("FindOrCreateInvoiceForMovement", (*uuid.UUID)(nil), &creditCardID, mock.Anything).
					Return(fixtureInvoice, nil)
				txManager.On("WithTransaction", mock.Anything).Return(nil)
				// As 3 primeiras parcelas gravam; a 4ª falha. Sem transação, a série ficaria
				// pela metade — 3 lançamentos órfãos somados na fatura e no limite.
				movRepo.On("Add", (*gorm.DB)(nil), mock.Anything).Return(domain.Movement{}, nil).Times(3)
				movRepo.On("Add", (*gorm.DB)(nil), mock.Anything).Return(domain.Movement{}, assert.AnError).Once()
			},
			expected: expected{created: 0, skipped: 10, err: nil},
		},
	}

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
			defer movRepo.AssertExpectations(t)
			defer invoiceUC.AssertExpectations(t)
			defer invoiceRepo.AssertExpectations(t)
			defer ccRepo.AssertExpectations(t)
			defer txManager.AssertExpectations(t)
			tc.mockSetup(movRepo, invoiceUC, invoiceRepo, ccRepo, txManager)

			ctx := authedCtx()
			if name == "should return error for unauthenticated context" {
				ctx = context.Background()
			}

			// Act
			result, err := uc.ConfirmInvoice(ctx, tc.input.payload)

			// Assert
			assert.ErrorIs(t, err, tc.expected.err)
			assert.Equal(t, tc.expected.created, result.Created)
			assert.Equal(t, tc.expected.skipped, result.Skipped)
		})
	}
}

// --- Extract: enriquecimentos de fatura (Fase 6) ---

func TestStatementUseCase_Extract_InvoiceEnrichment(t *testing.T) {
	var (
		creditCardID = uuid.New()
		walletID     = uuid.New()
		groupID      = uuid.New()
		registeredID = uuid.New()
	)

	creditCard := domain.CreditCard{
		ID:              &creditCardID,
		ClosingDay:      3,
		DueDay:          10,
		CreditLimit:     100000,
		DefaultWalletID: &walletID,
	}

	registeredInstallment := domain.Movement{
		ID:          &registeredID,
		Description: "TV da sala",
		Amount:      -119.90,
		CreditCardInfo: &domain.CreditCardMovement{
			CreditCardID:       &creditCardID,
			InstallmentGroupID: &groupID,
			InstallmentNumber:  intPtr(3),
			TotalInstallments:  intPtr(12),
		},
	}

	sameRootInstallment := registeredInstallment
	sameRootInstallment.Description = "Mercado Livre 02/12"

	type (
		input struct {
			gatewayResult domain.StatementExtractResult
			creditCardID  *uuid.UUID
		}
		expected struct {
			exclusionReasons []string
			linkedGroups     []string
			matchedGroups    []string
			warnTypes        []string
		}
	)

	tests := map[string]struct {
		// input
		input input
		// mocks
		mockSetup func(*MockStatementMovementRepository, *MockStatementCreditCardRepository)
		// expected
		expected expected
	}{
		"should mark as future installment the item dated after the invoice period end": {
			input: input{
				creditCardID: &creditCardID,
				gatewayResult: invoiceExtraction(
					domain.ExtractedMovement{Date: "2026-05-12", Description: "MERCADO LIVRE", Amount: -100.0},
					domain.ExtractedMovement{Date: "2026-05-20", Description: "IFOOD", Amount: -60.0},
					installmentItem("2026-09-03", "DROGARIA SP PARCELA 03 DE 03", -198.67, 3, 3),
				),
			},
			mockSetup: func(movRepo *MockStatementMovementRepository, ccRepo *MockStatementCreditCardRepository) {
				ccRepo.On("FindByID", creditCardID).Return(creditCard, nil)
			},
			expected: expected{
				exclusionReasons: []string{"", "", domain.ExclusionReasonFutureInstallment},
				linkedGroups:     []string{"", "", ""},
				matchedGroups:    []string{"", "", ""},
				warnTypes:        []string{domain.WarningFutureInstallmentExcluded},
			},
		},
		"should not mark a non-installment item dated after the invoice period end": {
			input: input{
				creditCardID: &creditCardID,
				gatewayResult: invoiceExtraction(
					domain.ExtractedMovement{Date: "2026-05-12", Description: "MERCADO LIVRE", Amount: -100.0},
					domain.ExtractedMovement{Date: "2026-09-03", Description: "POSTO SHELL", Amount: -220.0},
				),
			},
			mockSetup: func(movRepo *MockStatementMovementRepository, ccRepo *MockStatementCreditCardRepository) {
				ccRepo.On("FindByID", creditCardID).Return(creditCard, nil)
			},
			expected: expected{
				// Compra comum depois do fechamento pertence à fatura seguinte e é
				// parenteada pela resolução por data (AYD-004 decisão 2). Marcá-la faria
				// o usuário perdê-la — a decisão 7 vale só para PARCELAS.
				exclusionReasons: []string{"", ""},
				linkedGroups:     []string{"", ""},
				matchedGroups:    []string{"", ""},
				warnTypes:        nil,
			},
		},
		"should not mark an item dated inside the invoice period": {
			input: input{
				creditCardID: &creditCardID,
				gatewayResult: invoiceExtraction(
					domain.ExtractedMovement{Date: "2026-05-12", Description: "MERCADO LIVRE", Amount: -100.0},
					domain.ExtractedMovement{Date: "2026-06-03", Description: "IFOOD", Amount: -60.0},
				),
			},
			mockSetup: func(movRepo *MockStatementMovementRepository, ccRepo *MockStatementCreditCardRepository) {
				ccRepo.On("FindByID", creditCardID).Return(creditCard, nil)
			},
			expected: expected{
				exclusionReasons: []string{"", ""},
				linkedGroups:     []string{"", ""},
				matchedGroups:    []string{"", ""},
				warnTypes:        nil,
			},
		},
		"should pre-link the item when the installment match has high confidence": {
			input: input{
				creditCardID: &creditCardID,
				gatewayResult: invoiceExtraction(
					installmentItem("2026-05-12", "MERCADO LIVRE PARCELA 03/12", -119.90, 3, 12),
				),
			},
			mockSetup: func(movRepo *MockStatementMovementRepository, ccRepo *MockStatementCreditCardRepository) {
				ccRepo.On("FindByID", creditCardID).Return(creditCard, nil)
				movRepo.On("FindInstallmentCandidatesByCreditCard", creditCardID, 12).
					Return(domain.MovementList{sameRootInstallment}, nil)
			},
			expected: expected{
				exclusionReasons: []string{""},
				linkedGroups:     []string{groupID.String()},
				matchedGroups:    []string{groupID.String()},
				warnTypes:        []string{domain.WarningInstallmentMatchFound},
			},
		},
		"should only suggest the match when the description root diverges": {
			input: input{
				creditCardID: &creditCardID,
				gatewayResult: invoiceExtraction(
					installmentItem("2026-05-12", "MAGALU*MAGAZINELUIZA PARC 03/12", -119.90, 3, 12),
				),
			},
			mockSetup: func(movRepo *MockStatementMovementRepository, ccRepo *MockStatementCreditCardRepository) {
				ccRepo.On("FindByID", creditCardID).Return(creditCard, nil)
				movRepo.On("FindInstallmentCandidatesByCreditCard", creditCardID, 12).
					Return(domain.MovementList{registeredInstallment}, nil)
			},
			expected: expected{
				exclusionReasons: []string{""},
				linkedGroups:     []string{""},
				matchedGroups:    []string{groupID.String()},
				warnTypes:        []string{domain.WarningInstallmentMatchFound},
			},
		},
		"should not enrich anything when credit_card_id is absent": {
			input: input{
				creditCardID: nil,
				gatewayResult: invoiceExtraction(
					domain.ExtractedMovement{Date: "2026-05-12", Description: "MERCADO LIVRE", Amount: -100.0},
					installmentItem("2026-09-03", "DROGARIA SP PARCELA 03 DE 03", -198.67, 3, 3),
				),
			},
			mockSetup: func(movRepo *MockStatementMovementRepository, ccRepo *MockStatementCreditCardRepository) {
			},
			expected: expected{
				exclusionReasons: []string{"", ""},
				linkedGroups:     []string{"", ""},
				matchedGroups:    []string{"", ""},
				warnTypes:        nil,
			},
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
			defer visionGw.AssertExpectations(t)
			defer movRepo.AssertExpectations(t)
			defer ccRepo.AssertExpectations(t)
			visionGw.On("ExtractMovements", rawBytes, "application/pdf", "invoice").
				Return(tc.input.gatewayResult, nil)
			tc.mockSetup(movRepo, ccRepo)

			// Act
			result, err := uc.Extract(authedCtx(), rawBytes, "application/pdf", "", "invoice", tc.input.creditCardID)

			// Assert
			assert.ErrorIs(t, err, nil)
			assert.Equal(t, len(tc.input.gatewayResult.Movements), len(result.Movements),
				"no movement may be removed from the list")
			assert.Equal(t, tc.expected.exclusionReasons, exclusionReasonsOf(result))
			assert.Equal(t, tc.expected.linkedGroups, linkedGroupsOf(result))
			assert.Equal(t, tc.expected.matchedGroups, matchedGroupsOf(result))
			assert.Equal(t, tc.expected.warnTypes, enrichmentWarningsOf(result))
		})
	}
}

// --- ConfirmInvoice: vínculo de parcela já registrada (Fase 6) ---

func TestStatementUseCase_ConfirmInvoice_InstallmentLink(t *testing.T) {
	var (
		creditCardID  = uuid.New()
		otherCardID   = uuid.New()
		walletID      = uuid.New()
		invoiceID     = uuid.New()
		groupID       = uuid.New()
		registeredID  = uuid.New()
		registeredAt  = mustParseDate("2026-05-12")
		targetInvoice = domain.Invoice{ID: &invoiceID, Amount: -500.0, IsPaid: false}
	)

	creditCard := domain.CreditCard{
		ID:              &creditCardID,
		ClosingDay:      3,
		DueDay:          10,
		CreditLimit:     100000,
		DefaultWalletID: &walletID,
	}

	registered := domain.Movement{
		ID:          &registeredID,
		Description: "TV da sala",
		Amount:      -120.00,
		Date:        &registeredAt,
		IsPaid:      false,
		CreditCardInfo: &domain.CreditCardMovement{
			InvoiceID:          &invoiceID,
			CreditCardID:       &creditCardID,
			InstallmentGroupID: &groupID,
			InstallmentNumber:  intPtr(3),
			TotalInstallments:  intPtr(12),
		},
	}

	registeredOnAnotherCard := registered
	registeredOnAnotherCard.CreditCardInfo = &domain.CreditCardMovement{
		InvoiceID:          &invoiceID,
		CreditCardID:       &otherCardID,
		InstallmentGroupID: &groupID,
		InstallmentNumber:  intPtr(3),
		TotalInstallments:  intPtr(12),
	}

	linkedItem := installmentItem("2026-05-12", "MERCADO LIVRE PARCELA 03/12", -120.50, 3, 12)
	linkedItem.InstallmentGroupID = &groupID

	type (
		input struct {
			payload domain.InvoiceConfirmInput
		}
		expected struct {
			created int
			skipped int
			err     error
		}
	)

	tests := map[string]struct {
		// input
		input input
		// mocks
		mockSetup func(*MockStatementMovementRepository, *MockStatementInvoiceUseCase, *MockInvoiceRepository, *MockStatementCreditCardRepository, *MockTransactionManager)
		// expected
		expected expected
	}{
		"should update only the installment amount and skip the whole series": {
			input: input{payload: domain.InvoiceConfirmInput{
				CreditCardID: creditCardID,
				Movements:    []domain.ExtractedMovement{linkedItem},
			}},
			mockSetup: func(movRepo *MockStatementMovementRepository, invoiceUC *MockStatementInvoiceUseCase, invoiceRepo *MockInvoiceRepository, ccRepo *MockStatementCreditCardRepository, txManager *MockTransactionManager) {
				ccRepo.On("FindByID", creditCardID).Return(creditCard, nil)
				movRepo.On("FindExistingHashes", "user-123", mock.Anything).Return(map[string]bool{}, nil)
				movRepo.On("FindByInstallmentGroupFromNumber", groupID, 3).
					Return(domain.MovementList{registered}, nil)
				invoiceUC.On("FindOrCreateInvoiceForMovement", &invoiceID, &creditCardID, registeredAt).
					Return(targetInvoice, nil)
				txManager.On("WithTransaction", mock.Anything).Return(nil)
				// Só o valor muda: o método é estreito e não carrega descrição, data nem is_paid.
				movRepo.On("UpdateAmount", (*gorm.DB)(nil), registeredID, -120.50).
					Return(domain.Movement{}, nil)
				// Fatura e limite ajustados pelo delta (-120,50 − (-120,00)), não pelo valor cheio.
				invoiceRepo.On("UpdateAmount", (*gorm.DB)(nil), invoiceID, -500.50).Return(targetInvoice, nil)
				ccRepo.On("UpdateLimitDelta", (*gorm.DB)(nil), creditCardID, -0.50).Return(domain.CreditCard{}, nil)
			},
			// 12 − 3 + 1 = 10 parcelas contabilizadas como puladas; nada criado.
			expected: expected{created: 0, skipped: 10, err: nil},
		},
		"should create the series when the linked group belongs to another card": {
			input: input{payload: domain.InvoiceConfirmInput{
				CreditCardID: creditCardID,
				Movements:    []domain.ExtractedMovement{linkedSeriesItem(groupID)},
			}},
			mockSetup: func(movRepo *MockStatementMovementRepository, invoiceUC *MockStatementInvoiceUseCase, invoiceRepo *MockInvoiceRepository, ccRepo *MockStatementCreditCardRepository, txManager *MockTransactionManager) {
				ccRepo.On("FindByID", creditCardID).Return(creditCard, nil)
				movRepo.On("FindExistingHashes", "user-123", mock.Anything).Return(map[string]bool{}, nil)
				movRepo.On("FindByInstallmentGroupFromNumber", groupID, 3).
					Return(domain.MovementList{registeredOnAnotherCard}, nil)
				expectInstallmentSeriesCreation(movRepo, invoiceUC, invoiceRepo, ccRepo, txManager, creditCardID, targetInvoice, invoiceID)
			},
			expected: expected{created: 10, skipped: 0, err: nil},
		},
		"should create the series when the linked group does not exist or belongs to another user": {
			input: input{payload: domain.InvoiceConfirmInput{
				CreditCardID: creditCardID,
				Movements:    []domain.ExtractedMovement{linkedSeriesItem(groupID)},
			}},
			mockSetup: func(movRepo *MockStatementMovementRepository, invoiceUC *MockStatementInvoiceUseCase, invoiceRepo *MockInvoiceRepository, ccRepo *MockStatementCreditCardRepository, txManager *MockTransactionManager) {
				ccRepo.On("FindByID", creditCardID).Return(creditCard, nil)
				movRepo.On("FindExistingHashes", "user-123", mock.Anything).Return(map[string]bool{}, nil)
				// A consulta é escopada por usuário (BuildBaseQuery): o grupo de outro
				// usuário simplesmente não volta, igual a um grupo inexistente.
				movRepo.On("FindByInstallmentGroupFromNumber", groupID, 3).
					Return(domain.MovementList{}, nil)
				expectInstallmentSeriesCreation(movRepo, invoiceUC, invoiceRepo, ccRepo, txManager, creditCardID, targetInvoice, invoiceID)
			},
			expected: expected{created: 10, skipped: 0, err: nil},
		},
		"should skip the item dated after the invoice period end": {
			input: input{payload: domain.InvoiceConfirmInput{
				CreditCardID: creditCardID,
				Movements: []domain.ExtractedMovement{
					{Date: "2026-05-12", Description: "MERCADO LIVRE", Amount: -100.0},
					{Date: "2026-05-20", Description: "IFOOD", Amount: -60.0},
					installmentItem("2026-09-03", "DROGARIA SP PARCELA 03 DE 03", -198.67, 3, 3),
				},
			}},
			mockSetup: func(movRepo *MockStatementMovementRepository, invoiceUC *MockStatementInvoiceUseCase, invoiceRepo *MockInvoiceRepository, ccRepo *MockStatementCreditCardRepository, txManager *MockTransactionManager) {
				ccRepo.On("FindByID", creditCardID).Return(creditCard, nil)
				movRepo.On("FindExistingHashes", "user-123", mock.Anything).Return(map[string]bool{}, nil)
				invoiceUC.On("FindOrCreateInvoiceForMovement", (*uuid.UUID)(nil), &creditCardID, mock.Anything).
					Return(targetInvoice, nil)
				txManager.On("WithTransaction", mock.Anything).Return(nil)
				movRepo.On("Add", (*gorm.DB)(nil), mock.Anything).Return(domain.Movement{}, nil)
				invoiceRepo.On("UpdateAmount", (*gorm.DB)(nil), invoiceID, -600.0).Return(targetInvoice, nil)
				invoiceRepo.On("UpdateAmount", (*gorm.DB)(nil), invoiceID, -560.0).Return(targetInvoice, nil)
				ccRepo.On("UpdateLimitDelta", (*gorm.DB)(nil), creditCardID, -100.0).Return(domain.CreditCard{}, nil)
				ccRepo.On("UpdateLimitDelta", (*gorm.DB)(nil), creditCardID, -60.0).Return(domain.CreditCard{}, nil)
			},
			expected: expected{created: 2, skipped: 1, err: nil},
		},
		"should return ErrInsufficientCreditLimit when the item exceeds the card limit": {
			input: input{payload: domain.InvoiceConfirmInput{
				CreditCardID: creditCardID,
				Movements: []domain.ExtractedMovement{
					{Date: "2026-05-12", Description: "NOTEBOOK", Amount: -5000.0},
				},
			}},
			mockSetup: func(movRepo *MockStatementMovementRepository, invoiceUC *MockStatementInvoiceUseCase, invoiceRepo *MockInvoiceRepository, ccRepo *MockStatementCreditCardRepository, txManager *MockTransactionManager) {
				tightCard := creditCard
				tightCard.CreditLimit = 100.0
				ccRepo.On("FindByID", creditCardID).Return(tightCard, nil)
				movRepo.On("FindExistingHashes", "user-123", mock.Anything).Return(map[string]bool{}, nil)
				invoiceUC.On("FindOrCreateInvoiceForMovement", (*uuid.UUID)(nil), &creditCardID, mustParseDate("2026-05-12")).
					Return(targetInvoice, nil)
			},
			expected: expected{created: 0, skipped: 0, err: ErrInsufficientCreditLimit},
		},
		"should return ErrCreditCardNoDefaultWallet when the card has no default wallet": {
			input: input{payload: domain.InvoiceConfirmInput{
				CreditCardID: creditCardID,
				Movements: []domain.ExtractedMovement{
					{Date: "2026-05-12", Description: "NETFLIX", Amount: -55.90},
				},
			}},
			mockSetup: func(movRepo *MockStatementMovementRepository, invoiceUC *MockStatementInvoiceUseCase, invoiceRepo *MockInvoiceRepository, ccRepo *MockStatementCreditCardRepository, txManager *MockTransactionManager) {
				walletlessCard := creditCard
				walletlessCard.DefaultWalletID = nil
				ccRepo.On("FindByID", creditCardID).Return(walletlessCard, nil)
			},
			expected: expected{created: 0, skipped: 0, err: ErrCreditCardNoDefaultWallet},
		},
	}

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
			defer movRepo.AssertExpectations(t)
			defer invoiceUC.AssertExpectations(t)
			defer ccRepo.AssertExpectations(t)
			tc.mockSetup(movRepo, invoiceUC, invoiceRepo, ccRepo, txManager)

			// Act
			result, err := uc.ConfirmInvoice(authedCtx(), tc.input.payload)

			// Assert
			assert.ErrorIs(t, err, tc.expected.err)
			assert.Equal(t, tc.expected.created, result.Created)
			assert.Equal(t, tc.expected.skipped, result.Skipped)
			movRepo.AssertNotCalled(t, "UpdateStatementLink")
		})
	}
}

// --- fixtures & projections ---

func intPtr(v int) *int {
	return &v
}

func invoiceExtraction(movements ...domain.ExtractedMovement) domain.StatementExtractResult {
	return domain.StatementExtractResult{
		DocumentType: domain.DocInvoice,
		Confidence:   0.94,
		Movements:    movements,
	}
}

func installmentItem(date, description string, amount float64, number, total int) domain.ExtractedMovement {
	return domain.ExtractedMovement{
		Date:              date,
		Description:       description,
		Amount:            amount,
		InstallmentNumber: intPtr(number),
		TotalInstallments: intPtr(total),
	}
}

// linkedSeriesItem devolve um item parcelado que o cliente mandou vinculado a um
// grupo — usado nos cenários em que o vínculo é rejeitado e o item cai no
// caminho normal de criação da série.
func linkedSeriesItem(groupID uuid.UUID) domain.ExtractedMovement {
	item := installmentItem("2026-05-12", "MERCADO LIVRE PARCELA 03/12", -120.00, 3, 12)
	item.InstallmentGroupID = &groupID
	return item
}

// expectInstallmentSeriesCreation registra as expectativas da criação da série
// inteira (3..12 = 10 parcelas de -120,00 na mesma fatura).
func expectInstallmentSeriesCreation(
	movRepo *MockStatementMovementRepository,
	invoiceUC *MockStatementInvoiceUseCase,
	invoiceRepo *MockInvoiceRepository,
	ccRepo *MockStatementCreditCardRepository,
	txManager *MockTransactionManager,
	creditCardID uuid.UUID,
	invoice domain.Invoice,
	invoiceID uuid.UUID,
) {
	invoiceUC.On("FindOrCreateInvoiceForMovement", (*uuid.UUID)(nil), &creditCardID, mock.Anything).
		Return(invoice, nil)
	txManager.On("WithTransaction", mock.Anything).Return(nil)
	movRepo.On("Add", (*gorm.DB)(nil), mock.Anything).Return(domain.Movement{}, nil)
	invoiceRepo.On("UpdateAmount", (*gorm.DB)(nil), invoiceID, invoice.Amount-1200.0).Return(invoice, nil)
	ccRepo.On("UpdateLimitDelta", (*gorm.DB)(nil), creditCardID, -1200.0).Return(domain.CreditCard{}, nil)
}

func exclusionReasonsOf(r domain.StatementExtractResult) []string {
	out := make([]string, len(r.Movements))
	for i, m := range r.Movements {
		out[i] = m.ExclusionReason
	}
	return out
}

func linkedGroupsOf(r domain.StatementExtractResult) []string {
	out := make([]string, len(r.Movements))
	for i, m := range r.Movements {
		out[i] = uuidOrEmpty(m.InstallmentGroupID)
	}
	return out
}

func matchedGroupsOf(r domain.StatementExtractResult) []string {
	out := make([]string, len(r.Movements))
	for i, m := range r.Movements {
		if m.InstallmentMatch != nil {
			out[i] = m.InstallmentMatch.InstallmentGroupID.String()
		}
	}
	return out
}

// enrichmentWarningsOf devolve só os warnings da Fase 6, na ordem em que são emitidos.
func enrichmentWarningsOf(r domain.StatementExtractResult) []string {
	var out []string
	for _, w := range r.Warnings {
		switch w.Type {
		case domain.WarningFutureInstallmentExcluded, domain.WarningInstallmentMatchFound:
			out = append(out, w.Type)
		}
	}
	return out
}

func uuidOrEmpty(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}
