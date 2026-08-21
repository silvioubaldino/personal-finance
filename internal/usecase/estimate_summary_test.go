package usecase_test

import (
	"context"
	"testing"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/usecase"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func floatPtr(v float64) *float64 {
	return &v
}

func testMonthPeriod(month, year int) domain.Period {
	from := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, -1).Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	return domain.Period{From: from, To: to}
}

func movementWithCategory(amount float64, isPaid bool, typePayment domain.TypePayment, category domain.Category) domain.Movement {
	id := uuid.New()
	return domain.Movement{
		ID:          &id,
		Amount:      amount,
		IsPaid:      isPaid,
		TypePayment: typePayment,
		CategoryID:  category.ID,
		Category:    category,
	}
}

// simplifiedCategoryRow strips SubCategories so table expectations don't have to
// hand-derive Go's nil-vs-empty-slice zero-value rules for the untested subcategory path.
type simplifiedCategoryRow struct {
	EstimateID   *uuid.UUID
	CategoryID   *uuid.UUID
	CategoryName string
	IsIncome     bool
	IsPlanned    bool
	Budgeted     float64
	Realized     float64
	RealizedPaid float64
	Result       float64
	Progress     *float64
}

func simplifyCategories(categories []domain.EstimateSummaryCategory) []simplifiedCategoryRow {
	if categories == nil {
		return nil
	}
	rows := make([]simplifiedCategoryRow, len(categories))
	for i, category := range categories {
		rows[i] = simplifiedCategoryRow{
			EstimateID:   category.EstimateID,
			CategoryID:   category.CategoryID,
			CategoryName: category.CategoryName,
			IsIncome:     category.IsIncome,
			IsPlanned:    category.IsPlanned,
			Budgeted:     category.Budgeted,
			Realized:     category.Realized,
			RealizedPaid: category.RealizedPaid,
			Result:       category.Result,
			Progress:     category.Progress,
		}
	}
	return rows
}

func TestEstimate_FindSummary(t *testing.T) {
	type (
		input struct {
			month int
			year  int
		}
		expected struct {
			totals     domain.EstimateSummaryTotals
			categories []simplifiedCategoryRow
			err        error
		}
	)

	period := testMonthPeriod(8, 2026)

	salaryID := uuid.MustParse("11111111-0000-0000-0000-000000000001")
	foodID := uuid.MustParse("11111111-0000-0000-0000-000000000002")
	cardPaymentCatID := uuid.MustParse("11111111-0000-0000-0000-000000000003")
	transferCatID := uuid.MustParse("11111111-0000-0000-0000-000000000004")
	cardFixedCatID := uuid.MustParse("11111111-0000-0000-0000-000000000005")
	miscCatID := uuid.MustParse("11111111-0000-0000-0000-000000000006")
	bonusID := uuid.MustParse("11111111-0000-0000-0000-000000000007")
	funID := uuid.MustParse("11111111-0000-0000-0000-000000000008")

	salaryCat := domain.Category{ID: &salaryID, Description: "Salário", IsIncome: true}
	foodCat := domain.Category{ID: &foodID, Description: "Alimentação", IsIncome: false}
	cardPaymentCat := domain.Category{ID: &cardPaymentCatID, Description: "Cartão", IsIncome: false}
	transferCat := domain.Category{ID: &transferCatID, Description: "Transferência", IsIncome: false}
	cardFixedCat := domain.Category{ID: &cardFixedCatID, Description: "Fatura futura", IsIncome: false}
	miscCat := domain.Category{ID: &miscCatID, Description: "Diversos", IsIncome: false}
	bonusCat := domain.Category{ID: &bonusID, Description: "Bônus", IsIncome: true}
	funCat := domain.Category{ID: &funID, Description: "Lazer", IsIncome: false}

	salaryEstID := uuid.MustParse("22222222-0000-0000-0000-000000000001")
	foodEstID := uuid.MustParse("22222222-0000-0000-0000-000000000002")

	tests := map[string]struct {
		// input
		input input
		// mocks
		mockSetup func(mockMovRepo *usecase.MockMovementRepository, mockInvoiceUC *usecase.MockInvoice, mockEstimateRepo *usecase.MockEstimateRepository)
		// expected
		expected expected
	}{
		"should return 400 when month is out of range": {
			input:     input{month: 13, year: 2026},
			mockSetup: func(_ *usecase.MockMovementRepository, _ *usecase.MockInvoice, _ *usecase.MockEstimateRepository) {},
			expected: expected{
				categories: nil,
				err:        domain.ErrInvalidInput,
			},
		},
		"should match the golden file totals for the Impacto medido scenario": {
			input: input{month: 8, year: 2026},
			mockSetup: func(mockMovRepo *usecase.MockMovementRepository, mockInvoiceUC *usecase.MockInvoice, mockEstimateRepo *usecase.MockEstimateRepository) {
				mockMovRepo.On("FindByPeriod", period).Return(domain.MovementList{
					movementWithCategory(5000, true, domain.TypePaymentDebit, salaryCat),
					movementWithCategory(-600, true, domain.TypePaymentDebit, foodCat),
					movementWithCategory(-300, false, domain.TypePaymentDebit, foodCat),
					movementWithCategory(-400, true, domain.TypePaymentInvoicePayment, cardPaymentCat),
					movementWithCategory(1000, true, domain.TypePaymentInternalTransfer, transferCat),
					movementWithCategory(-1000, true, domain.TypePaymentInternalTransfer, transferCat),
				}, nil)
				mockInvoiceUC.On("FindDetailedInvoicesByPeriod", context.Background(), period).Return([]domain.DetailedInvoice{
					{Movements: domain.MovementList{
						movementWithCategory(-400, true, domain.TypePaymentCreditCard, foodCat),
					}},
				}, nil)
				mockEstimateRepo.On("FindCategoriesByMonth", 8, 2026).Return([]domain.EstimateCategories{
					{ID: &salaryEstID, CategoryID: &salaryID, CategoryName: "Salário", IsCategoryIncome: true, Amount: 5000},
					{ID: &foodEstID, CategoryID: &foodID, CategoryName: "Alimentação", IsCategoryIncome: false, Amount: -1000},
				}, nil)
				mockEstimateRepo.On("FindSubcategoriesByMonth", 8, 2026).Return([]domain.EstimateSubCategories{}, nil)
			},
			expected: expected{
				totals: domain.EstimateSummaryTotals{
					Income:        domain.EstimateTotalsBucket{Budgeted: 5000, Realized: 5000, RealizedPaid: 5000, Consolidated: 5000},
					Expense:       domain.EstimateTotalsBucket{Budgeted: -1000, Realized: -1300, RealizedPaid: -1000, Consolidated: -1300},
					PeriodBalance: 3700,
				},
				categories: []simplifiedCategoryRow{
					{
						EstimateID: &salaryEstID, CategoryID: &salaryID, CategoryName: "Salário",
						IsIncome: true, IsPlanned: true, Budgeted: 5000, Realized: 5000, RealizedPaid: 5000,
						Result: 0, Progress: floatPtr(1.0),
					},
					{
						EstimateID: &foodEstID, CategoryID: &foodID, CategoryName: "Alimentação",
						IsIncome: false, IsPlanned: true, Budgeted: -1000, Realized: -1300, RealizedPaid: -1000,
						Result: -300, Progress: floatPtr(1.3),
					},
				},
			},
		},
		"should include a pending movement in realized but exclude it from realized_paid": {
			input: input{month: 8, year: 2026},
			mockSetup: func(mockMovRepo *usecase.MockMovementRepository, mockInvoiceUC *usecase.MockInvoice, mockEstimateRepo *usecase.MockEstimateRepository) {
				mockMovRepo.On("FindByPeriod", period).Return(domain.MovementList{
					movementWithCategory(-300, false, domain.TypePaymentDebit, foodCat),
				}, nil)
				mockInvoiceUC.On("FindDetailedInvoicesByPeriod", context.Background(), period).Return([]domain.DetailedInvoice{}, nil)
				mockEstimateRepo.On("FindCategoriesByMonth", 8, 2026).Return([]domain.EstimateCategories{
					{ID: &foodEstID, CategoryID: &foodID, CategoryName: "Alimentação", IsCategoryIncome: false, Amount: -1000},
				}, nil)
				mockEstimateRepo.On("FindSubcategoriesByMonth", 8, 2026).Return([]domain.EstimateSubCategories{}, nil)
			},
			expected: expected{
				totals: domain.EstimateSummaryTotals{
					Expense:       domain.EstimateTotalsBucket{Budgeted: -1000, Realized: -300, RealizedPaid: 0, Consolidated: -1000},
					PeriodBalance: -1000,
				},
				categories: []simplifiedCategoryRow{
					{
						EstimateID: &foodEstID, CategoryID: &foodID, CategoryName: "Alimentação",
						IsIncome: false, IsPlanned: true, Budgeted: -1000, Realized: -300, RealizedPaid: 0,
						Result: 700, Progress: floatPtr(0.3),
					},
				},
			},
		},
		"should exclude both legs of an internal transfer from every category": {
			input: input{month: 8, year: 2026},
			mockSetup: func(mockMovRepo *usecase.MockMovementRepository, mockInvoiceUC *usecase.MockInvoice, mockEstimateRepo *usecase.MockEstimateRepository) {
				mockMovRepo.On("FindByPeriod", period).Return(domain.MovementList{
					movementWithCategory(1000, true, domain.TypePaymentInternalTransfer, transferCat),
					movementWithCategory(-1000, true, domain.TypePaymentInternalTransfer, transferCat),
				}, nil)
				mockInvoiceUC.On("FindDetailedInvoicesByPeriod", context.Background(), period).Return([]domain.DetailedInvoice{}, nil)
				mockEstimateRepo.On("FindCategoriesByMonth", 8, 2026).Return([]domain.EstimateCategories{}, nil)
				mockEstimateRepo.On("FindSubcategoriesByMonth", 8, 2026).Return([]domain.EstimateSubCategories{}, nil)
			},
			expected: expected{
				categories: []simplifiedCategoryRow{},
			},
		},
		"should exclude invoice_payment from every category": {
			input: input{month: 8, year: 2026},
			mockSetup: func(mockMovRepo *usecase.MockMovementRepository, mockInvoiceUC *usecase.MockInvoice, mockEstimateRepo *usecase.MockEstimateRepository) {
				mockMovRepo.On("FindByPeriod", period).Return(domain.MovementList{
					movementWithCategory(-400, true, domain.TypePaymentInvoicePayment, cardPaymentCat),
				}, nil)
				mockInvoiceUC.On("FindDetailedInvoicesByPeriod", context.Background(), period).Return([]domain.DetailedInvoice{}, nil)
				mockEstimateRepo.On("FindCategoriesByMonth", 8, 2026).Return([]domain.EstimateCategories{}, nil)
				mockEstimateRepo.On("FindSubcategoriesByMonth", 8, 2026).Return([]domain.EstimateSubCategories{}, nil)
			},
			expected: expected{
				categories: []simplifiedCategoryRow{},
			},
		},
		"should count a credit card purchase via the invoice items": {
			input: input{month: 8, year: 2026},
			mockSetup: func(mockMovRepo *usecase.MockMovementRepository, mockInvoiceUC *usecase.MockInvoice, mockEstimateRepo *usecase.MockEstimateRepository) {
				mockMovRepo.On("FindByPeriod", period).Return(domain.MovementList{}, nil)
				mockInvoiceUC.On("FindDetailedInvoicesByPeriod", context.Background(), period).Return([]domain.DetailedInvoice{
					{Movements: domain.MovementList{
						movementWithCategory(-400, true, domain.TypePaymentCreditCard, foodCat),
					}},
				}, nil)
				mockEstimateRepo.On("FindCategoriesByMonth", 8, 2026).Return([]domain.EstimateCategories{
					{ID: &foodEstID, CategoryID: &foodID, CategoryName: "Alimentação", IsCategoryIncome: false, Amount: -1000},
				}, nil)
				mockEstimateRepo.On("FindSubcategoriesByMonth", 8, 2026).Return([]domain.EstimateSubCategories{}, nil)
			},
			expected: expected{
				totals: domain.EstimateSummaryTotals{
					Expense:       domain.EstimateTotalsBucket{Budgeted: -1000, Realized: -400, RealizedPaid: -400, Consolidated: -1000},
					PeriodBalance: -1000,
				},
				categories: []simplifiedCategoryRow{
					{
						EstimateID: &foodEstID, CategoryID: &foodID, CategoryName: "Alimentação",
						IsIncome: false, IsPlanned: true, Budgeted: -1000, Realized: -400, RealizedPaid: -400,
						Result: 600, Progress: floatPtr(0.4),
					},
				},
			},
		},
		"should count an invoice_remainder in the month of the invoice that receives it, as an unplanned line": {
			input: input{month: 8, year: 2026},
			mockSetup: func(mockMovRepo *usecase.MockMovementRepository, mockInvoiceUC *usecase.MockInvoice, mockEstimateRepo *usecase.MockEstimateRepository) {
				mockMovRepo.On("FindByPeriod", period).Return(domain.MovementList{}, nil)
				mockInvoiceUC.On("FindDetailedInvoicesByPeriod", context.Background(), period).Return([]domain.DetailedInvoice{
					{Movements: domain.MovementList{
						movementWithCategory(-300, false, domain.TypePaymentInvoiceRemainder, cardFixedCat),
					}},
				}, nil)
				mockEstimateRepo.On("FindCategoriesByMonth", 8, 2026).Return([]domain.EstimateCategories{}, nil)
				mockEstimateRepo.On("FindSubcategoriesByMonth", 8, 2026).Return([]domain.EstimateSubCategories{}, nil)
			},
			expected: expected{
				totals: domain.EstimateSummaryTotals{
					Expense:       domain.EstimateTotalsBucket{Realized: -300, Consolidated: -300},
					PeriodBalance: -300,
				},
				categories: []simplifiedCategoryRow{
					{
						CategoryID: &cardFixedCatID, CategoryName: "Fatura futura",
						IsIncome: false, IsPlanned: false, Budgeted: 0, Realized: -300, RealizedPaid: 0,
						Result: -300, Progress: nil,
					},
				},
			},
		},
		"should render a virtual line with null progress for a category with movements but no estimate": {
			input: input{month: 8, year: 2026},
			mockSetup: func(mockMovRepo *usecase.MockMovementRepository, mockInvoiceUC *usecase.MockInvoice, mockEstimateRepo *usecase.MockEstimateRepository) {
				mockMovRepo.On("FindByPeriod", period).Return(domain.MovementList{
					movementWithCategory(-250, true, domain.TypePaymentDebit, miscCat),
				}, nil)
				mockInvoiceUC.On("FindDetailedInvoicesByPeriod", context.Background(), period).Return([]domain.DetailedInvoice{}, nil)
				mockEstimateRepo.On("FindCategoriesByMonth", 8, 2026).Return([]domain.EstimateCategories{}, nil)
				mockEstimateRepo.On("FindSubcategoriesByMonth", 8, 2026).Return([]domain.EstimateSubCategories{}, nil)
			},
			expected: expected{
				totals: domain.EstimateSummaryTotals{
					Expense:       domain.EstimateTotalsBucket{Realized: -250, RealizedPaid: -250, Consolidated: -250},
					PeriodBalance: -250,
				},
				categories: []simplifiedCategoryRow{
					{
						CategoryID: &miscCatID, CategoryName: "Diversos",
						IsIncome: false, IsPlanned: false, Budgeted: 0, Realized: -250, RealizedPaid: -250,
						Result: -250, Progress: nil,
					},
				},
			},
		},
		"should keep is_income from the Category flag when a refund has a positive amount in an expense category": {
			input: input{month: 8, year: 2026},
			mockSetup: func(mockMovRepo *usecase.MockMovementRepository, mockInvoiceUC *usecase.MockInvoice, mockEstimateRepo *usecase.MockEstimateRepository) {
				mockMovRepo.On("FindByPeriod", period).Return(domain.MovementList{
					movementWithCategory(150, true, domain.TypePaymentDebit, foodCat),
				}, nil)
				mockInvoiceUC.On("FindDetailedInvoicesByPeriod", context.Background(), period).Return([]domain.DetailedInvoice{}, nil)
				mockEstimateRepo.On("FindCategoriesByMonth", 8, 2026).Return([]domain.EstimateCategories{
					{ID: &foodEstID, CategoryID: &foodID, CategoryName: "Alimentação", IsCategoryIncome: false, Amount: -1000},
				}, nil)
				mockEstimateRepo.On("FindSubcategoriesByMonth", 8, 2026).Return([]domain.EstimateSubCategories{}, nil)
			},
			expected: expected{
				totals: domain.EstimateSummaryTotals{
					Expense:       domain.EstimateTotalsBucket{Budgeted: -1000, Realized: 150, RealizedPaid: 150, Consolidated: -1000},
					PeriodBalance: -1000,
				},
				categories: []simplifiedCategoryRow{
					{
						EstimateID: &foodEstID, CategoryID: &foodID, CategoryName: "Alimentação",
						IsIncome: false, IsPlanned: true, Budgeted: -1000, Realized: 150, RealizedPaid: 150,
						Result: 1150, Progress: floatPtr(0.15),
					},
				},
			},
		},
		"should order categories income before expense, planned before unplanned, then alphabetically": {
			input: input{month: 8, year: 2026},
			mockSetup: func(mockMovRepo *usecase.MockMovementRepository, mockInvoiceUC *usecase.MockInvoice, mockEstimateRepo *usecase.MockEstimateRepository) {
				mockMovRepo.On("FindByPeriod", period).Return(domain.MovementList{
					movementWithCategory(5000, true, domain.TypePaymentDebit, salaryCat),
					movementWithCategory(200, true, domain.TypePaymentDebit, bonusCat),
					movementWithCategory(-500, true, domain.TypePaymentDebit, foodCat),
					movementWithCategory(-100, true, domain.TypePaymentDebit, funCat),
				}, nil)
				mockInvoiceUC.On("FindDetailedInvoicesByPeriod", context.Background(), period).Return([]domain.DetailedInvoice{}, nil)
				mockEstimateRepo.On("FindCategoriesByMonth", 8, 2026).Return([]domain.EstimateCategories{
					{ID: &salaryEstID, CategoryID: &salaryID, CategoryName: "Salário", IsCategoryIncome: true, Amount: 5000},
					{ID: &foodEstID, CategoryID: &foodID, CategoryName: "Alimentação", IsCategoryIncome: false, Amount: -1000},
				}, nil)
				mockEstimateRepo.On("FindSubcategoriesByMonth", 8, 2026).Return([]domain.EstimateSubCategories{}, nil)
			},
			expected: expected{
				totals: domain.EstimateSummaryTotals{
					Income:        domain.EstimateTotalsBucket{Budgeted: 5000, Realized: 5200, RealizedPaid: 5200, Consolidated: 5200},
					Expense:       domain.EstimateTotalsBucket{Budgeted: -1000, Realized: -600, RealizedPaid: -600, Consolidated: -1100},
					PeriodBalance: 4100,
				},
				categories: []simplifiedCategoryRow{
					{
						EstimateID: &salaryEstID, CategoryID: &salaryID, CategoryName: "Salário",
						IsIncome: true, IsPlanned: true, Budgeted: 5000, Realized: 5000, RealizedPaid: 5000,
						Result: 0, Progress: floatPtr(1.0),
					},
					{
						CategoryID: &bonusID, CategoryName: "Bônus",
						IsIncome: true, IsPlanned: false, Budgeted: 0, Realized: 200, RealizedPaid: 200,
						Result: 200, Progress: nil,
					},
					{
						EstimateID: &foodEstID, CategoryID: &foodID, CategoryName: "Alimentação",
						IsIncome: false, IsPlanned: true, Budgeted: -1000, Realized: -500, RealizedPaid: -500,
						Result: 500, Progress: floatPtr(0.5),
					},
					{
						CategoryID: &funID, CategoryName: "Lazer",
						IsIncome: false, IsPlanned: false, Budgeted: 0, Realized: -100, RealizedPaid: -100,
						Result: -100, Progress: nil,
					},
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Arrange
			var (
				mockMovRepo      = &usecase.MockMovementRepository{}
				mockInvoiceUC    = &usecase.MockInvoice{}
				mockEstimateRepo = &usecase.MockEstimateRepository{}
				mockCategoryRepo = &usecase.MockCategoryRepository{}
				mockSubCatRepo   = &usecase.MockSubCategory{}
				svc              = usecase.NewEstimate(mockEstimateRepo, mockCategoryRepo, mockSubCatRepo, mockMovRepo, mockInvoiceUC)
			)
			defer mockMovRepo.AssertExpectations(t)
			defer mockInvoiceUC.AssertExpectations(t)
			defer mockEstimateRepo.AssertExpectations(t)
			tc.mockSetup(mockMovRepo, mockInvoiceUC, mockEstimateRepo)

			// Act
			summary, err := svc.FindSummary(context.Background(), tc.input.month, tc.input.year)

			// Assert
			assert.ErrorIs(t, err, tc.expected.err)
			assert.Equal(t, tc.expected.totals, summary.Totals)
			assert.Equal(t, tc.expected.categories, simplifyCategories(summary.Categories))
		})
	}
}
