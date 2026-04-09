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
	return NewStatementUseCase(visionGw, classGw, movRepo, catRepo, nil)
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
		mockSetup        func(*MockStatementClassificationGateway, *MockStatementMovementRepository, *MockStatementCategoryRepository)
		expectedSources  []string
		expectedCatIDs   []*uuid.UUID
		expectError      bool
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
					"user-123", walletID,
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
