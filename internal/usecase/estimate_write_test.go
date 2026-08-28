package usecase_test

import (
	"context"
	"testing"

	"personal-finance/internal/domain"
	"personal-finance/internal/usecase"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestEstimate_AddEstimateCategory(t *testing.T) {
	type (
		input struct {
			category domain.EstimateCategories
		}
		expected struct {
			amount float64
			err    error
		}
	)

	categoryID := uuid.MustParse("33333333-0000-0000-0000-000000000001")

	tests := map[string]struct {
		// input
		input input
		// mocks
		mockSetup func(mockRepo *usecase.MockEstimateRepository, mockCategoryRepo *usecase.MockCategoryRepository)
		// expected
		expected expected
	}{
		"should force amount negative when category is an expense": {
			input: input{category: domain.EstimateCategories{CategoryID: &categoryID, Amount: 1000}},
			mockSetup: func(mockRepo *usecase.MockEstimateRepository, mockCategoryRepo *usecase.MockCategoryRepository) {
				mockCategoryRepo.On("FindByID", categoryID).Return(domain.Category{ID: &categoryID, IsIncome: false}, nil)
				mockRepo.On("AddEstimateCategory", domain.EstimateCategories{CategoryID: &categoryID, Amount: -1000}).
					Return(domain.EstimateCategories{CategoryID: &categoryID, Amount: -1000}, nil)
			},
			expected: expected{amount: -1000, err: nil},
		},
		"should force amount positive when category is income": {
			input: input{category: domain.EstimateCategories{CategoryID: &categoryID, Amount: -5000}},
			mockSetup: func(mockRepo *usecase.MockEstimateRepository, mockCategoryRepo *usecase.MockCategoryRepository) {
				mockCategoryRepo.On("FindByID", categoryID).Return(domain.Category{ID: &categoryID, IsIncome: true}, nil)
				mockRepo.On("AddEstimateCategory", domain.EstimateCategories{CategoryID: &categoryID, Amount: 5000}).
					Return(domain.EstimateCategories{CategoryID: &categoryID, Amount: 5000}, nil)
			},
			expected: expected{amount: 5000, err: nil},
		},
		"should return error when category_id is missing": {
			input:     input{category: domain.EstimateCategories{Amount: 1000}},
			mockSetup: func(_ *usecase.MockEstimateRepository, _ *usecase.MockCategoryRepository) {},
			expected:  expected{amount: 0, err: domain.ErrInvalidInput},
		},
		"should return error when category is not found": {
			input: input{category: domain.EstimateCategories{CategoryID: &categoryID, Amount: 1000}},
			mockSetup: func(_ *usecase.MockEstimateRepository, mockCategoryRepo *usecase.MockCategoryRepository) {
				mockCategoryRepo.On("FindByID", categoryID).Return(domain.Category{}, assert.AnError)
			},
			expected: expected{amount: 0, err: assert.AnError},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Arrange
			var (
				mockRepo         = &usecase.MockEstimateRepository{}
				mockCategoryRepo = &usecase.MockCategoryRepository{}
				mockSubCatRepo   = &usecase.MockSubCategory{}
				mockMovRepo      = &usecase.MockMovementRepository{}
				mockInvoiceUC    = &usecase.MockInvoice{}
				svc              = usecase.NewEstimate(mockRepo, mockCategoryRepo, mockSubCatRepo, mockMovRepo, mockInvoiceUC)
			)
			defer mockRepo.AssertExpectations(t)
			defer mockCategoryRepo.AssertExpectations(t)
			tc.mockSetup(mockRepo, mockCategoryRepo)

			// Act
			result, err := svc.AddEstimateCategory(context.Background(), tc.input.category)

			// Assert
			assert.ErrorIs(t, err, tc.expected.err)
			assert.Equal(t, tc.expected.amount, result.Amount)
		})
	}
}

func TestEstimate_UpdateEstimateCategoryAmount(t *testing.T) {
	type (
		input struct {
			id     uuid.UUID
			amount float64
		}
		expected struct {
			amount float64
			err    error
		}
	)

	estimateID := uuid.MustParse("33333333-0000-0000-0000-000000000002")
	categoryID := uuid.MustParse("33333333-0000-0000-0000-000000000003")

	tests := map[string]struct {
		// input
		input input
		// mocks
		mockSetup func(mockRepo *usecase.MockEstimateRepository, mockCategoryRepo *usecase.MockCategoryRepository)
		// expected
		expected expected
	}{
		"should force amount negative when category is an expense": {
			input: input{id: estimateID, amount: 800},
			mockSetup: func(mockRepo *usecase.MockEstimateRepository, mockCategoryRepo *usecase.MockCategoryRepository) {
				mockRepo.On("FindCategoryByID", estimateID).Return(domain.EstimateCategories{ID: &estimateID, CategoryID: &categoryID}, nil)
				mockCategoryRepo.On("FindByID", categoryID).Return(domain.Category{ID: &categoryID, IsIncome: false}, nil)
				mockRepo.On("UpdateEstimateCategoryAmount", &estimateID, -800.0).
					Return(domain.EstimateCategories{ID: &estimateID, Amount: -800}, nil)
			},
			expected: expected{amount: -800, err: nil},
		},
		"should force amount positive when category is income": {
			input: input{id: estimateID, amount: -4000},
			mockSetup: func(mockRepo *usecase.MockEstimateRepository, mockCategoryRepo *usecase.MockCategoryRepository) {
				mockRepo.On("FindCategoryByID", estimateID).Return(domain.EstimateCategories{ID: &estimateID, CategoryID: &categoryID}, nil)
				mockCategoryRepo.On("FindByID", categoryID).Return(domain.Category{ID: &categoryID, IsIncome: true}, nil)
				mockRepo.On("UpdateEstimateCategoryAmount", &estimateID, 4000.0).
					Return(domain.EstimateCategories{ID: &estimateID, Amount: 4000}, nil)
			},
			expected: expected{amount: 4000, err: nil},
		},
		"should return error when estimate category is not found": {
			input: input{id: estimateID, amount: 800},
			mockSetup: func(mockRepo *usecase.MockEstimateRepository, _ *usecase.MockCategoryRepository) {
				mockRepo.On("FindCategoryByID", estimateID).Return(domain.EstimateCategories{}, assert.AnError)
			},
			expected: expected{amount: 0, err: assert.AnError},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Arrange
			var (
				mockRepo         = &usecase.MockEstimateRepository{}
				mockCategoryRepo = &usecase.MockCategoryRepository{}
				mockSubCatRepo   = &usecase.MockSubCategory{}
				mockMovRepo      = &usecase.MockMovementRepository{}
				mockInvoiceUC    = &usecase.MockInvoice{}
				svc              = usecase.NewEstimate(mockRepo, mockCategoryRepo, mockSubCatRepo, mockMovRepo, mockInvoiceUC)
			)
			defer mockRepo.AssertExpectations(t)
			defer mockCategoryRepo.AssertExpectations(t)
			tc.mockSetup(mockRepo, mockCategoryRepo)

			// Act
			result, err := svc.UpdateEstimateCategoryAmount(context.Background(), &tc.input.id, tc.input.amount)

			// Assert
			assert.ErrorIs(t, err, tc.expected.err)
			assert.Equal(t, tc.expected.amount, result.Amount)
		})
	}
}

func TestEstimate_AddEstimateSubCategory(t *testing.T) {
	type (
		input struct {
			subEstimate domain.EstimateSubCategories
		}
		expected struct {
			amount float64
			err    error
		}
	)

	subCategoryID := uuid.MustParse("33333333-0000-0000-0000-000000000004")
	categoryID := uuid.MustParse("33333333-0000-0000-0000-000000000005")

	tests := map[string]struct {
		// input
		input input
		// mocks
		mockSetup func(mockRepo *usecase.MockEstimateRepository, mockCategoryRepo *usecase.MockCategoryRepository, mockSubCatRepo *usecase.MockSubCategory)
		// expected
		expected expected
	}{
		"should force amount negative when the parent category is an expense": {
			input: input{subEstimate: domain.EstimateSubCategories{SubCategoryID: &subCategoryID, Amount: 300}},
			mockSetup: func(mockRepo *usecase.MockEstimateRepository, mockCategoryRepo *usecase.MockCategoryRepository, mockSubCatRepo *usecase.MockSubCategory) {
				mockSubCatRepo.On("FindByID", subCategoryID).Return(domain.SubCategory{ID: &subCategoryID, CategoryID: &categoryID}, nil)
				mockCategoryRepo.On("FindByID", categoryID).Return(domain.Category{ID: &categoryID, IsIncome: false}, nil)
				mockRepo.On("AddEstimateSubCategory", domain.EstimateSubCategories{SubCategoryID: &subCategoryID, Amount: -300}).
					Return(domain.EstimateSubCategories{SubCategoryID: &subCategoryID, Amount: -300}, nil)
			},
			expected: expected{amount: -300, err: nil},
		},
		"should return error when sub_category_id is missing": {
			input:     input{subEstimate: domain.EstimateSubCategories{Amount: 300}},
			mockSetup: func(_ *usecase.MockEstimateRepository, _ *usecase.MockCategoryRepository, _ *usecase.MockSubCategory) {},
			expected:  expected{amount: 0, err: domain.ErrInvalidInput},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Arrange
			var (
				mockRepo         = &usecase.MockEstimateRepository{}
				mockCategoryRepo = &usecase.MockCategoryRepository{}
				mockSubCatRepo   = &usecase.MockSubCategory{}
				mockMovRepo      = &usecase.MockMovementRepository{}
				mockInvoiceUC    = &usecase.MockInvoice{}
				svc              = usecase.NewEstimate(mockRepo, mockCategoryRepo, mockSubCatRepo, mockMovRepo, mockInvoiceUC)
			)
			defer mockRepo.AssertExpectations(t)
			defer mockCategoryRepo.AssertExpectations(t)
			defer mockSubCatRepo.AssertExpectations(t)
			tc.mockSetup(mockRepo, mockCategoryRepo, mockSubCatRepo)

			// Act
			result, err := svc.AddEstimateSubCategory(context.Background(), tc.input.subEstimate)

			// Assert
			assert.ErrorIs(t, err, tc.expected.err)
			assert.Equal(t, tc.expected.amount, result.Amount)
		})
	}
}

func TestEstimate_UpdateEstimateSubCategoryAmount(t *testing.T) {
	type (
		input struct {
			id     uuid.UUID
			amount float64
		}
		expected struct {
			amount float64
			err    error
		}
	)

	subEstimateID := uuid.MustParse("33333333-0000-0000-0000-000000000006")
	subCategoryID := uuid.MustParse("33333333-0000-0000-0000-000000000007")
	categoryID := uuid.MustParse("33333333-0000-0000-0000-000000000008")

	tests := map[string]struct {
		// input
		input input
		// mocks
		mockSetup func(mockRepo *usecase.MockEstimateRepository, mockCategoryRepo *usecase.MockCategoryRepository, mockSubCatRepo *usecase.MockSubCategory)
		// expected
		expected expected
	}{
		"should force amount positive when the parent category is income": {
			input: input{id: subEstimateID, amount: -150},
			mockSetup: func(mockRepo *usecase.MockEstimateRepository, mockCategoryRepo *usecase.MockCategoryRepository, mockSubCatRepo *usecase.MockSubCategory) {
				mockRepo.On("FindSubCategoryByID", subEstimateID).Return(domain.EstimateSubCategories{ID: &subEstimateID, SubCategoryID: &subCategoryID}, nil)
				mockSubCatRepo.On("FindByID", subCategoryID).Return(domain.SubCategory{ID: &subCategoryID, CategoryID: &categoryID}, nil)
				mockCategoryRepo.On("FindByID", categoryID).Return(domain.Category{ID: &categoryID, IsIncome: true}, nil)
				mockRepo.On("UpdateEstimateSubCategoryAmount", &subEstimateID, 150.0).
					Return(domain.EstimateSubCategories{ID: &subEstimateID, Amount: 150}, nil)
			},
			expected: expected{amount: 150, err: nil},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Arrange
			var (
				mockRepo         = &usecase.MockEstimateRepository{}
				mockCategoryRepo = &usecase.MockCategoryRepository{}
				mockSubCatRepo   = &usecase.MockSubCategory{}
				mockMovRepo      = &usecase.MockMovementRepository{}
				mockInvoiceUC    = &usecase.MockInvoice{}
				svc              = usecase.NewEstimate(mockRepo, mockCategoryRepo, mockSubCatRepo, mockMovRepo, mockInvoiceUC)
			)
			defer mockRepo.AssertExpectations(t)
			defer mockCategoryRepo.AssertExpectations(t)
			defer mockSubCatRepo.AssertExpectations(t)
			tc.mockSetup(mockRepo, mockCategoryRepo, mockSubCatRepo)

			// Act
			result, err := svc.UpdateEstimateSubCategoryAmount(context.Background(), &tc.input.id, tc.input.amount)

			// Assert
			assert.ErrorIs(t, err, tc.expected.err)
			assert.Equal(t, tc.expected.amount, result.Amount)
		})
	}
}
