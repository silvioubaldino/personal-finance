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
	return NewStatementUseCase(visionGw, classGw, movRepo, catRepo, nil, nil, nil, nil)
}

func newStatementUseCaseWithInvoice(
	visionGw *MockStatementVisionGateway,
	classGw *MockStatementClassificationGateway,
	movRepo *MockStatementMovementRepository,
	catRepo *MockStatementCategoryRepository,
	invoiceUC *MockStatementInvoiceUseCase,
	ccRepo *MockStatementCreditCardRepository,
) *StatementUseCase {
	return NewStatementUseCase(visionGw, classGw, movRepo, catRepo, nil, nil, invoiceUC, ccRepo)
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
			&MockStatementMovementRepository{}, &MockStatementCategoryRepository{}, nil, decryptor, nil, nil)

		result, err := uc.Extract(authedCtx(), rawBytes, "application/pdf", "s3cret", "")

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
			&MockStatementMovementRepository{}, &MockStatementCategoryRepository{}, nil, decryptor, nil, nil)

		result, err := uc.Extract(authedCtx(), rawBytes, "image/png", "", "")

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
				&MockStatementMovementRepository{}, &MockStatementCategoryRepository{}, nil, decryptor, nil, nil)

			_, err := uc.Extract(authedCtx(), rawBytes, "application/pdf", "", "")

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
				uc       = NewStatementUseCase(visionGw, classGw, movRepo, catRepo, nil, nil, nil, nil)
			)
			defer visionGw.AssertExpectations(t)
			tc.mockSetup(visionGw)

			// Act
			result, err := uc.Extract(authedCtx(), rawBytes, "image/png", "", tc.input.sourceType)

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

// --- ConfirmInvoice ---

func TestStatementUseCase_ConfirmInvoice(t *testing.T) {
	creditCardID := uuid.New()
	invoiceID := uuid.New()
	catID := uuid.New()
	uncategorizedID := uuid.MustParse(domain.UncategorizedCategoryID)

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
		mockSetup func(*MockStatementMovementRepository, *MockStatementInvoiceUseCase, *MockStatementCreditCardRepository)
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
			mockSetup: func(movRepo *MockStatementMovementRepository, invoiceUC *MockStatementInvoiceUseCase, ccRepo *MockStatementCreditCardRepository) {
				ccRepo.On("FindByID", creditCardID).Return(domain.CreditCard{ID: &creditCardID}, nil)
				movRepo.On("FindExistingHashes", "user-123", mock.Anything).Return(map[string]bool{}, nil)
				invoiceUC.On("FindOrCreateInvoiceForMovement", (*uuid.UUID)(nil), &creditCardID, mustParseDate("2026-05-12")).
					Return(fixtureInvoice, nil)
				movRepo.On("Add", (*gorm.DB)(nil), mock.MatchedBy(func(m domain.Movement) bool {
					return m.TypePayment == domain.TypePaymentCreditCard && !m.IsPaid && m.CategoryID != nil && *m.CategoryID == catID
				})).Return(domain.Movement{}, nil)
				invoiceUC.On("UpdateAmount", invoiceID, -55.90).Return(fixtureInvoice, nil)
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
			mockSetup: func(movRepo *MockStatementMovementRepository, invoiceUC *MockStatementInvoiceUseCase, ccRepo *MockStatementCreditCardRepository) {
				ccRepo.On("FindByID", creditCardID).Return(domain.CreditCard{ID: &creditCardID}, nil)
				movRepo.On("FindExistingHashes", "user-123", mock.Anything).Return(map[string]bool{}, nil)
				invoiceUC.On("FindOrCreateInvoiceForMovement", (*uuid.UUID)(nil), &creditCardID, mustParseDate("2026-05-12")).
					Return(fixtureInvoice, nil)
				movRepo.On("Add", (*gorm.DB)(nil), mock.MatchedBy(func(m domain.Movement) bool {
					return m.CategoryID != nil && *m.CategoryID == uncategorizedID
				})).Return(domain.Movement{}, nil)
				invoiceUC.On("UpdateAmount", invoiceID, -29.90).Return(fixtureInvoice, nil)
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
			mockSetup: func(movRepo *MockStatementMovementRepository, invoiceUC *MockStatementInvoiceUseCase, ccRepo *MockStatementCreditCardRepository) {
				ccRepo.On("FindByID", creditCardID).Return(domain.CreditCard{ID: &creditCardID}, nil)
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
			mockSetup: func(movRepo *MockStatementMovementRepository, invoiceUC *MockStatementInvoiceUseCase, ccRepo *MockStatementCreditCardRepository) {
				ccRepo.On("FindByID", creditCardID).Return(domain.CreditCard{ID: &creditCardID}, nil)
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
			mockSetup: func(movRepo *MockStatementMovementRepository, invoiceUC *MockStatementInvoiceUseCase, ccRepo *MockStatementCreditCardRepository) {
				ccRepo.On("FindByID", creditCardID).Return(domain.CreditCard{}, assert.AnError)
			},
			expected: expected{err: assert.AnError},
		},
		"should return error when movements is empty": {
			input: input{
				payload: domain.InvoiceConfirmInput{
					CreditCardID: creditCardID,
					Movements:    []domain.ExtractedMovement{},
				},
			},
			mockSetup: func(movRepo *MockStatementMovementRepository, invoiceUC *MockStatementInvoiceUseCase, ccRepo *MockStatementCreditCardRepository) {
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
			mockSetup: func(movRepo *MockStatementMovementRepository, invoiceUC *MockStatementInvoiceUseCase, ccRepo *MockStatementCreditCardRepository) {
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
			mockSetup: func(movRepo *MockStatementMovementRepository, invoiceUC *MockStatementInvoiceUseCase, ccRepo *MockStatementCreditCardRepository) {
				ccRepo.On("FindByID", creditCardID).Return(domain.CreditCard{ID: &creditCardID}, nil)
				movRepo.On("FindExistingHashes", "user-123", mock.Anything).Return(map[string]bool{}, nil)
				// 10 installments generated (3..12 = 10 remaining including current)
				invoiceUC.On("FindOrCreateInvoiceForMovement", (*uuid.UUID)(nil), &creditCardID, mock.Anything).
					Return(fixtureInvoice, nil)
				movRepo.On("Add", (*gorm.DB)(nil), mock.MatchedBy(func(m domain.Movement) bool {
					return m.TypePayment == domain.TypePaymentCreditCard
				})).Return(domain.Movement{}, nil)
				invoiceUC.On("UpdateAmount", invoiceID, mock.Anything).Return(fixtureInvoice, nil)
				ccRepo.On("UpdateLimitDelta", (*gorm.DB)(nil), creditCardID, mock.Anything).Return(domain.CreditCard{}, nil)
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
			mockSetup: func(movRepo *MockStatementMovementRepository, invoiceUC *MockStatementInvoiceUseCase, ccRepo *MockStatementCreditCardRepository) {
				ccRepo.On("FindByID", creditCardID).Return(domain.CreditCard{ID: &creditCardID}, nil)

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
			mockSetup: func(movRepo *MockStatementMovementRepository, invoiceUC *MockStatementInvoiceUseCase, ccRepo *MockStatementCreditCardRepository) {
				ccRepo.On("FindByID", creditCardID).Return(domain.CreditCard{ID: &creditCardID}, nil)

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
				invoiceUC.On("UpdateAmount", invoiceID, mock.Anything).Return(fixtureInvoice, nil)
				ccRepo.On("UpdateLimitDelta", (*gorm.DB)(nil), creditCardID, mock.Anything).Return(domain.CreditCard{}, nil)
			},
			// 10 parcelas no total (3..12); 2 já existem (puladas), 8 são criadas.
			expected: expected{created: 8, skipped: 2, err: nil},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Arrange
			var (
				visionGw  = &MockStatementVisionGateway{}
				classGw   = &MockStatementClassificationGateway{}
				movRepo   = &MockStatementMovementRepository{}
				catRepo   = &MockStatementCategoryRepository{}
				invoiceUC = &MockStatementInvoiceUseCase{}
				ccRepo    = &MockStatementCreditCardRepository{}
				uc        = newStatementUseCaseWithInvoice(visionGw, classGw, movRepo, catRepo, invoiceUC, ccRepo)
			)
			defer movRepo.AssertExpectations(t)
			defer invoiceUC.AssertExpectations(t)
			defer ccRepo.AssertExpectations(t)
			tc.mockSetup(movRepo, invoiceUC, ccRepo)

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
