package usecase

import (
	"context"
	"testing"
	"time"

	"personal-finance/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func dashboardDate(year int, month time.Month, day int) *time.Time {
	t := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	return &t
}

func dashboardMovement(amount float64, date *time.Time, isPaid bool) domain.Movement {
	categoryID := uuid.New()
	return dashboardMovementWithCategory(amount, date, isPaid, &categoryID)
}

func dashboardMovementWithCategory(amount float64, date *time.Time, isPaid bool, categoryID *uuid.UUID) domain.Movement {
	return domain.Movement{
		Amount:     amount,
		Date:       date,
		IsPaid:     isPaid,
		CategoryID: categoryID,
		Category:   domain.Category{ID: categoryID},
	}
}

func dashboardMovementWithPayment(
	amount float64,
	date *time.Time,
	isPaid bool,
	typePayment domain.TypePayment,
) domain.Movement {
	movement := dashboardMovement(amount, date, isPaid)
	movement.TypePayment = typePayment
	return movement
}

func dashboardEstimate(amount float64, isIncome bool, categoryID *uuid.UUID) domain.EstimateCategories {
	return domain.EstimateCategories{
		CategoryID:       categoryID,
		IsCategoryIncome: isIncome,
		Amount:           amount,
	}
}

func dashboardInvoice(
	cardID *uuid.UUID,
	cardName string,
	cardColor string,
	dueDate *time.Time,
	amount float64,
) domain.Invoice {
	return domain.Invoice{
		CreditCardID: cardID,
		CreditCard:   domain.CreditCard{ID: cardID, Name: cardName, Color: cardColor},
		DueDate:      *dueDate,
		Amount:       amount,
	}
}

func expenseWeekdays(dates ...*time.Time) []domain.ExpenseWeekdayPoint {
	counts := make([]int, 7)
	for _, d := range dates {
		counts[int(d.Weekday())]++
	}

	distribution := make([]domain.ExpenseWeekdayPoint, 7)
	for weekday, count := range counts {
		var percentage float64
		if len(dates) > 0 {
			percentage = float64(count) / float64(len(dates))
		}
		distribution[weekday] = domain.ExpenseWeekdayPoint{
			Weekday:    weekday,
			Count:      count,
			Percentage: percentage,
		}
	}
	return distribution
}

func emptyInvoiceSummary(year int, months ...time.Month) domain.CreditCardInvoiceSummary {
	series := make([]domain.CreditCardInvoicePoint, 0, len(months))
	for _, month := range months {
		series = append(series, domain.CreditCardInvoicePoint{
			Month:  int(month),
			Year:   year,
			ByCard: []domain.CreditCardInvoiceSlice{},
		})
	}
	return domain.CreditCardInvoiceSummary{
		Cards:  []domain.CreditCardRef{},
		Series: series,
	}
}

func TestDashboard_CalculateSummary(t *testing.T) {
	type (
		input struct {
			period domain.Period
		}
		expected struct {
			output domain.DashboardSummary
			err    error
		}
	)

	nubankID := uuid.New()
	itauID := uuid.New()
	janExpenseCategoryID := uuid.New()
	marExpenseCategoryID := uuid.New()
	weekdayCat1ID := uuid.New()
	weekdayCat2ID := uuid.New()
	weekdayCat3ID := uuid.New()
	foodCategoryID := uuid.New()
	transportCategoryID := uuid.New()
	invoiceCategoryID := uuid.New()
	incomeCat := uuid.New()
	expenseCat := uuid.New()

	tests := map[string]struct {
		input     input
		mockSetup func(
			mockMovRepo *MockMovementRepository,
			mockEstRepo *MockEstimateRepository,
			mockInvRepo *MockInvoiceRepository,
		)
		expected expected
	}{
		"should build multi-month series filling gaps with zeros when months have no activity": {
			input: input{period: domain.Period{
				From: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2026, time.March, 31, 0, 0, 0, 0, time.UTC),
			}},
			mockSetup: func(
				mockMovRepo *MockMovementRepository,
				mockEstRepo *MockEstimateRepository,
				mockInvRepo *MockInvoiceRepository,
			) {
				period := domain.Period{
					From: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
					To:   time.Date(2026, time.March, 31, 0, 0, 0, 0, time.UTC),
				}
				movements := domain.MovementList{
					dashboardMovement(5000, dashboardDate(2026, time.January, 10), true),
					dashboardMovementWithCategory(-3000, dashboardDate(2026, time.January, 15), true, &janExpenseCategoryID),
					dashboardMovement(4000, dashboardDate(2026, time.March, 5), true),
					dashboardMovementWithCategory(-1000, dashboardDate(2026, time.March, 8), true, &marExpenseCategoryID),
				}
				mockMovRepo.On("FindByPeriod", period).Return(movements, nil)
				mockEstRepo.On("FindCategoriesByMonth", 3, 2026).
					Return([]domain.EstimateCategories{}, nil)
				mockInvRepo.On("FindByPeriod", period).Return([]domain.Invoice{}, nil)
			},
			expected: expected{
				output: domain.DashboardSummary{
					MonthlySeries: []domain.MonthlyPoint{
						{Month: 1, Year: 2026, Income: 5000, Expense: -3000, Net: 2000},
						{Month: 2, Year: 2026, Income: 0, Expense: 0, Net: 0},
						{Month: 3, Year: 2026, Income: 4000, Expense: -1000, Net: 3000},
					},
					CurrentMonth: domain.BudgetComparison{
						Month: 3, Year: 2026,
						Budget: domain.DashboardBudget{
							Income:  domain.BudgetLine{Budgeted: 0, Realized: 4000},
							Expense: domain.BudgetLine{Budgeted: 0, Realized: -1000},
						},
					},
					CreditCardInvoices: emptyInvoiceSummary(2026, time.January, time.February, time.March),
					ExpenseWeekdayDistribution: expenseWeekdays(
						dashboardDate(2026, time.January, 15),
						dashboardDate(2026, time.March, 8),
					),
					ExpenseByCategory: []domain.CategoryExpensePoint{
						{CategoryID: &janExpenseCategoryID, Total: -3000},
						{CategoryID: &marExpenseCategoryID, Total: -1000},
					},
					KPIs: domain.DashboardKPIs{
						TotalIncome:  9000,
						TotalExpense: -4000,
					},
				},
				err: nil,
			},
		},
		"should compute budget vs realized for current month using paid movements": {
			input: input{period: domain.Period{
				From: time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2026, time.June, 30, 0, 0, 0, 0, time.UTC),
			}},
			mockSetup: func(
				mockMovRepo *MockMovementRepository,
				mockEstRepo *MockEstimateRepository,
				mockInvRepo *MockInvoiceRepository,
			) {
				period := domain.Period{
					From: time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC),
					To:   time.Date(2026, time.June, 30, 0, 0, 0, 0, time.UTC),
				}
				movements := domain.MovementList{
					dashboardMovementWithCategory(4800, dashboardDate(2026, time.June, 10), true, &incomeCat),
					dashboardMovementWithCategory(-3200, dashboardDate(2026, time.June, 12), true, &expenseCat),
					dashboardMovementWithCategory(-1000, dashboardDate(2026, time.June, 20), false, &expenseCat),
				}
				mockMovRepo.On("FindByPeriod", period).Return(movements, nil)
				mockEstRepo.On("FindCategoriesByMonth", 6, 2026).Return([]domain.EstimateCategories{
					dashboardEstimate(5000, true, &incomeCat),
					dashboardEstimate(-3000, false, &expenseCat),
				}, nil)
				mockInvRepo.On("FindByPeriod", period).Return([]domain.Invoice{}, nil)
			},
			expected: expected{
				output: domain.DashboardSummary{
					MonthlySeries: []domain.MonthlyPoint{
						{Month: 6, Year: 2026, Income: 4800, Expense: -3200, Net: 1600},
					},
					CurrentMonth: domain.BudgetComparison{
						Month: 6, Year: 2026,
						Budget: domain.DashboardBudget{
							Income:  domain.BudgetLine{Budgeted: 5000, Realized: 5000},
							Expense: domain.BudgetLine{Budgeted: -3000, Realized: -3200},
						},
					},
					CreditCardInvoices: emptyInvoiceSummary(2026, time.June),
					ExpenseWeekdayDistribution: expenseWeekdays(
						dashboardDate(2026, time.June, 12),
						dashboardDate(2026, time.June, 20),
					),
					ExpenseByCategory: []domain.CategoryExpensePoint{
						{CategoryID: &expenseCat, Total: -3200},
					},
					KPIs: domain.DashboardKPIs{
						TotalIncome:  4800,
						TotalExpense: -3200,
					},
				},
				err: nil,
			},
		},
		"should return zeroed summary when period has no movements": {
			input: input{period: domain.Period{
				From: time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2026, time.April, 30, 0, 0, 0, 0, time.UTC),
			}},
			mockSetup: func(
				mockMovRepo *MockMovementRepository,
				mockEstRepo *MockEstimateRepository,
				mockInvRepo *MockInvoiceRepository,
			) {
				period := domain.Period{
					From: time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC),
					To:   time.Date(2026, time.April, 30, 0, 0, 0, 0, time.UTC),
				}
				mockMovRepo.On("FindByPeriod", period).Return(domain.MovementList{}, nil)
				mockEstRepo.On("FindCategoriesByMonth", 4, 2026).
					Return([]domain.EstimateCategories{}, nil)
				mockInvRepo.On("FindByPeriod", period).Return([]domain.Invoice{}, nil)
			},
			expected: expected{
				output: domain.DashboardSummary{
					MonthlySeries: []domain.MonthlyPoint{
						{Month: 4, Year: 2026, Income: 0, Expense: 0, Net: 0},
					},
					CurrentMonth: domain.BudgetComparison{
						Month: 4, Year: 2026,
						Budget: domain.DashboardBudget{
							Income:  domain.BudgetLine{Budgeted: 0, Realized: 0},
							Expense: domain.BudgetLine{Budgeted: 0, Realized: 0},
						},
					},
					CreditCardInvoices:         emptyInvoiceSummary(2026, time.April),
					ExpenseWeekdayDistribution: expenseWeekdays(),
					ExpenseByCategory:          []domain.CategoryExpensePoint{},
					KPIs: domain.DashboardKPIs{
						TotalIncome:  0,
						TotalExpense: 0,
					},
				},
				err: nil,
			},
		},
		"should stack invoice totals by card, carrying each card's own color and zero-filling missing cards": {
			input: input{period: domain.Period{
				From: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2026, time.February, 28, 0, 0, 0, 0, time.UTC),
			}},
			mockSetup: func(
				mockMovRepo *MockMovementRepository,
				mockEstRepo *MockEstimateRepository,
				mockInvRepo *MockInvoiceRepository,
			) {
				period := domain.Period{
					From: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
					To:   time.Date(2026, time.February, 28, 0, 0, 0, 0, time.UTC),
				}
				mockMovRepo.On("FindByPeriod", period).Return(domain.MovementList{}, nil)
				mockEstRepo.On("FindCategoriesByMonth", 2, 2026).
					Return([]domain.EstimateCategories{}, nil)
				mockInvRepo.On("FindByPeriod", period).Return([]domain.Invoice{
					dashboardInvoice(&nubankID, "Nubank", "#820ad1", dashboardDate(2026, time.January, 10), -1400),
					dashboardInvoice(&itauID, "Itau", "", dashboardDate(2026, time.January, 15), -700),
					dashboardInvoice(&nubankID, "Nubank", "#820ad1", dashboardDate(2026, time.February, 10), -900),
				}, nil)
			},
			expected: expected{
				output: domain.DashboardSummary{
					MonthlySeries: []domain.MonthlyPoint{
						{Month: 1, Year: 2026, Income: 0, Expense: 0, Net: 0},
						{Month: 2, Year: 2026, Income: 0, Expense: 0, Net: 0},
					},
					CurrentMonth: domain.BudgetComparison{
						Month: 2, Year: 2026,
						Budget: domain.DashboardBudget{
							Income:  domain.BudgetLine{Budgeted: 0, Realized: 0},
							Expense: domain.BudgetLine{Budgeted: 0, Realized: 0},
						},
					},
					CreditCardInvoices: domain.CreditCardInvoiceSummary{
						Cards: []domain.CreditCardRef{
							{CreditCardID: &itauID, Name: "Itau", Color: ""},
							{CreditCardID: &nubankID, Name: "Nubank", Color: "#820ad1"},
						},
						Series: []domain.CreditCardInvoicePoint{
							{
								Month: 1, Year: 2026, Total: -2100,
								ByCard: []domain.CreditCardInvoiceSlice{
									{CreditCardID: &itauID, Amount: -700},
									{CreditCardID: &nubankID, Amount: -1400},
								},
							},
							{
								Month: 2, Year: 2026, Total: -900,
								ByCard: []domain.CreditCardInvoiceSlice{
									{CreditCardID: &itauID, Amount: 0},
									{CreditCardID: &nubankID, Amount: -900},
								},
							},
						},
					},
					ExpenseWeekdayDistribution: expenseWeekdays(),
					ExpenseByCategory:          []domain.CategoryExpensePoint{},
					KPIs:                       domain.DashboardKPIs{},
				},
				err: nil,
			},
		},
		"should count unpaid expenses and exclude internal transfers from all aggregates": {
			input: input{period: domain.Period{
				From: time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2026, time.February, 28, 0, 0, 0, 0, time.UTC),
			}},
			mockSetup: func(
				mockMovRepo *MockMovementRepository,
				mockEstRepo *MockEstimateRepository,
				mockInvRepo *MockInvoiceRepository,
			) {
				period := domain.Period{
					From: time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC),
					To:   time.Date(2026, time.February, 28, 0, 0, 0, 0, time.UTC),
				}
				movements := domain.MovementList{
					dashboardMovementWithCategory(-100, dashboardDate(2026, time.February, 6), true, &weekdayCat1ID),
					dashboardMovementWithPayment(
						-200, dashboardDate(2026, time.February, 13), false, domain.TypePaymentCreditCard,
					),
					dashboardMovementWithCategory(-300, dashboardDate(2026, time.February, 20), true, &weekdayCat2ID),
					dashboardMovementWithCategory(-400, dashboardDate(2026, time.February, 17), true, &weekdayCat3ID),
					dashboardMovementWithPayment(
						-500, dashboardDate(2026, time.February, 17), true, domain.TypePaymentInternalTransfer,
					),
					dashboardMovement(1000, dashboardDate(2026, time.February, 6), true),
				}
				mockMovRepo.On("FindByPeriod", period).Return(movements, nil)
				mockEstRepo.On("FindCategoriesByMonth", 2, 2026).
					Return([]domain.EstimateCategories{}, nil)
				mockInvRepo.On("FindByPeriod", period).Return([]domain.Invoice{}, nil)
			},
			expected: expected{
				output: domain.DashboardSummary{
					MonthlySeries: []domain.MonthlyPoint{
						{Month: 2, Year: 2026, Income: 1000, Expense: -800, Net: 200},
					},
					CurrentMonth: domain.BudgetComparison{
						Month: 2, Year: 2026,
						Budget: domain.DashboardBudget{
							Income:  domain.BudgetLine{Budgeted: 0, Realized: 1000},
							Expense: domain.BudgetLine{Budgeted: 0, Realized: -800},
						},
					},
					CreditCardInvoices: emptyInvoiceSummary(2026, time.February),
					ExpenseWeekdayDistribution: []domain.ExpenseWeekdayPoint{
						{Weekday: 0, Count: 0, Percentage: 0},
						{Weekday: 1, Count: 0, Percentage: 0},
						{Weekday: 2, Count: 1, Percentage: 0.25},
						{Weekday: 3, Count: 0, Percentage: 0},
						{Weekday: 4, Count: 0, Percentage: 0},
						{Weekday: 5, Count: 3, Percentage: 0.75},
						{Weekday: 6, Count: 0, Percentage: 0},
					},
					ExpenseByCategory: []domain.CategoryExpensePoint{
						{CategoryID: &weekdayCat3ID, Total: -400},
						{CategoryID: &weekdayCat2ID, Total: -300},
						{CategoryID: &weekdayCat1ID, Total: -100},
					},
					KPIs: domain.DashboardKPIs{
						TotalIncome:  1000,
						TotalExpense: -800,
					},
				},
				err: nil,
			},
		},
		"should sum expenses per category across months, carrying name/color, sorted by largest spend first": {
			input: input{period: domain.Period{
				From: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2026, time.February, 28, 0, 0, 0, 0, time.UTC),
			}},
			mockSetup: func(
				mockMovRepo *MockMovementRepository,
				mockEstRepo *MockEstimateRepository,
				mockInvRepo *MockInvoiceRepository,
			) {
				period := domain.Period{
					From: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
					To:   time.Date(2026, time.February, 28, 0, 0, 0, 0, time.UTC),
				}
				foodCategory := domain.Category{ID: &foodCategoryID, Description: "Alimentação", Color: "#f97316"}
				transportCategory := domain.Category{ID: &transportCategoryID, Description: "Transporte", Color: ""}
				movements := domain.MovementList{
					{Amount: -100, Date: dashboardDate(2026, time.January, 5), IsPaid: true, CategoryID: &foodCategoryID, Category: foodCategory},
					{Amount: -150, Date: dashboardDate(2026, time.February, 5), IsPaid: true, CategoryID: &foodCategoryID, Category: foodCategory},
					{Amount: -400, Date: dashboardDate(2026, time.January, 10), IsPaid: true, CategoryID: &transportCategoryID, Category: transportCategory},
					{Amount: -900, Date: dashboardDate(2026, time.February, 10), IsPaid: false, CategoryID: &transportCategoryID, Category: transportCategory},
				}
				mockMovRepo.On("FindByPeriod", period).Return(movements, nil)
				mockEstRepo.On("FindCategoriesByMonth", 2, 2026).
					Return([]domain.EstimateCategories{}, nil)
				mockInvRepo.On("FindByPeriod", period).Return([]domain.Invoice{}, nil)
			},
			expected: expected{
				output: domain.DashboardSummary{
					MonthlySeries: []domain.MonthlyPoint{
						{Month: 1, Year: 2026, Income: 0, Expense: -500, Net: -500},
						{Month: 2, Year: 2026, Income: 0, Expense: -150, Net: -150},
					},
					CurrentMonth: domain.BudgetComparison{
						Month: 2, Year: 2026,
						Budget: domain.DashboardBudget{
							Income:  domain.BudgetLine{Budgeted: 0, Realized: 0},
							Expense: domain.BudgetLine{Budgeted: 0, Realized: -150},
						},
					},
					CreditCardInvoices: emptyInvoiceSummary(2026, time.January, time.February),
					ExpenseWeekdayDistribution: expenseWeekdays(
						dashboardDate(2026, time.January, 5),
						dashboardDate(2026, time.February, 5),
						dashboardDate(2026, time.January, 10),
						dashboardDate(2026, time.February, 10),
					),
					ExpenseByCategory: []domain.CategoryExpensePoint{
						{CategoryID: &transportCategoryID, Name: "Transporte", Color: "", Total: -400},
						{CategoryID: &foodCategoryID, Name: "Alimentação", Color: "#f97316", Total: -250},
					},
					KPIs: domain.DashboardKPIs{
						TotalIncome:  0,
						TotalExpense: -650,
					},
				},
				err: nil,
			},
		},
		"should exclude invoice_payment from expense by category, keeping credit_card and invoice_remainder": {
			input: input{period: domain.Period{
				From: time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2026, time.February, 28, 0, 0, 0, 0, time.UTC),
			}},
			mockSetup: func(
				mockMovRepo *MockMovementRepository,
				mockEstRepo *MockEstimateRepository,
				mockInvRepo *MockInvoiceRepository,
			) {
				period := domain.Period{
					From: time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC),
					To:   time.Date(2026, time.February, 28, 0, 0, 0, 0, time.UTC),
				}
				invoiceCategory := domain.Category{ID: &invoiceCategoryID}
				movements := domain.MovementList{
					{
						Amount: -300, Date: dashboardDate(2026, time.February, 5), IsPaid: true,
						CategoryID: &foodCategoryID, Category: domain.Category{ID: &foodCategoryID},
						TypePayment: domain.TypePaymentCreditCard,
					},
					{
						Amount: -50, Date: dashboardDate(2026, time.February, 6), IsPaid: true,
						CategoryID: &invoiceCategoryID, Category: invoiceCategory,
						TypePayment: domain.TypePaymentInvoiceRemainder,
					},
					{
						Amount: -350, Date: dashboardDate(2026, time.February, 10), IsPaid: true,
						CategoryID: &invoiceCategoryID, Category: invoiceCategory,
						TypePayment: domain.TypePaymentInvoicePayment,
					},
				}
				mockMovRepo.On("FindByPeriod", period).Return(movements, nil)
				mockEstRepo.On("FindCategoriesByMonth", 2, 2026).
					Return([]domain.EstimateCategories{}, nil)
				mockInvRepo.On("FindByPeriod", period).Return([]domain.Invoice{}, nil)
			},
			expected: expected{
				output: domain.DashboardSummary{
					MonthlySeries: []domain.MonthlyPoint{
						{Month: 2, Year: 2026, Income: 0, Expense: -700, Net: -700},
					},
					CurrentMonth: domain.BudgetComparison{
						Month: 2, Year: 2026,
						Budget: domain.DashboardBudget{
							Income:  domain.BudgetLine{Budgeted: 0, Realized: 0},
							Expense: domain.BudgetLine{Budgeted: 0, Realized: -700},
						},
					},
					CreditCardInvoices: emptyInvoiceSummary(2026, time.February),
					ExpenseWeekdayDistribution: expenseWeekdays(
						dashboardDate(2026, time.February, 5),
						dashboardDate(2026, time.February, 6),
						dashboardDate(2026, time.February, 10),
					),
					ExpenseByCategory: []domain.CategoryExpensePoint{
						{CategoryID: &foodCategoryID, Total: -300},
						{CategoryID: &invoiceCategoryID, Total: -50},
					},
					KPIs: domain.DashboardKPIs{
						TotalIncome:  0,
						TotalExpense: -700,
					},
				},
				err: nil,
			},
		},
		"should exclude movements tagged with the internal transfer category IDs from expense by category": {
			input: input{period: domain.Period{
				From: time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2026, time.February, 28, 0, 0, 0, 0, time.UTC),
			}},
			mockSetup: func(
				mockMovRepo *MockMovementRepository,
				mockEstRepo *MockEstimateRepository,
				mockInvRepo *MockInvoiceRepository,
			) {
				period := domain.Period{
					From: time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC),
					To:   time.Date(2026, time.February, 28, 0, 0, 0, 0, time.UTC),
				}
				transferOutID := uuid.MustParse(domain.InternalTransferOutCategoryID)
				transferInID := uuid.MustParse(domain.InternalTransferInCategoryID)
				movements := domain.MovementList{
					{
						Amount: -300, Date: dashboardDate(2026, time.February, 5), IsPaid: true,
						CategoryID: &foodCategoryID, Category: domain.Category{ID: &foodCategoryID},
					},
					{
						Amount: -700, Date: dashboardDate(2026, time.February, 6), IsPaid: true,
						CategoryID: &transferOutID, Category: domain.Category{ID: &transferOutID},
					},
					{
						Amount: -900, Date: dashboardDate(2026, time.February, 7), IsPaid: true,
						CategoryID: &transferInID, Category: domain.Category{ID: &transferInID},
					},
				}
				mockMovRepo.On("FindByPeriod", period).Return(movements, nil)
				mockEstRepo.On("FindCategoriesByMonth", 2, 2026).
					Return([]domain.EstimateCategories{}, nil)
				mockInvRepo.On("FindByPeriod", period).Return([]domain.Invoice{}, nil)
			},
			expected: expected{
				output: domain.DashboardSummary{
					MonthlySeries: []domain.MonthlyPoint{
						{Month: 2, Year: 2026, Income: 0, Expense: -1900, Net: -1900},
					},
					CurrentMonth: domain.BudgetComparison{
						Month: 2, Year: 2026,
						Budget: domain.DashboardBudget{
							Income:  domain.BudgetLine{Budgeted: 0, Realized: 0},
							Expense: domain.BudgetLine{Budgeted: 0, Realized: -1900},
						},
					},
					CreditCardInvoices: emptyInvoiceSummary(2026, time.February),
					ExpenseWeekdayDistribution: expenseWeekdays(
						dashboardDate(2026, time.February, 5),
						dashboardDate(2026, time.February, 6),
						dashboardDate(2026, time.February, 7),
					),
					ExpenseByCategory: []domain.CategoryExpensePoint{
						{CategoryID: &foodCategoryID, Total: -300},
					},
					KPIs: domain.DashboardKPIs{
						TotalIncome:  0,
						TotalExpense: -1900,
					},
				},
				err: nil,
			},
		},
		"should return error when movement repository fails": {
			input: input{period: domain.Period{
				From: time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2026, time.July, 31, 0, 0, 0, 0, time.UTC),
			}},
			mockSetup: func(
				mockMovRepo *MockMovementRepository,
				_ *MockEstimateRepository,
				_ *MockInvoiceRepository,
			) {
				period := domain.Period{
					From: time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC),
					To:   time.Date(2026, time.July, 31, 0, 0, 0, 0, time.UTC),
				}
				mockMovRepo.On("FindByPeriod", period).
					Return(domain.MovementList{}, assert.AnError)
			},
			expected: expected{
				output: domain.DashboardSummary{},
				err:    assert.AnError,
			},
		},
		"should return error when invoice repository fails": {
			input: input{period: domain.Period{
				From: time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2026, time.August, 31, 0, 0, 0, 0, time.UTC),
			}},
			mockSetup: func(
				mockMovRepo *MockMovementRepository,
				mockEstRepo *MockEstimateRepository,
				mockInvRepo *MockInvoiceRepository,
			) {
				period := domain.Period{
					From: time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC),
					To:   time.Date(2026, time.August, 31, 0, 0, 0, 0, time.UTC),
				}
				mockMovRepo.On("FindByPeriod", period).Return(domain.MovementList{}, nil)
				mockEstRepo.On("FindCategoriesByMonth", 8, 2026).
					Return([]domain.EstimateCategories{}, nil)
				mockInvRepo.On("FindByPeriod", period).
					Return([]domain.Invoice{}, assert.AnError)
			},
			expected: expected{
				output: domain.DashboardSummary{},
				err:    assert.AnError,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var (
				mockMovRepo = &MockMovementRepository{}
				mockEstRepo = &MockEstimateRepository{}
				mockInvRepo = &MockInvoiceRepository{}
				uc          = NewDashboard(mockMovRepo, mockEstRepo, mockInvRepo)
			)
			defer mockMovRepo.AssertExpectations(t)
			defer mockEstRepo.AssertExpectations(t)
			defer mockInvRepo.AssertExpectations(t)
			tc.mockSetup(mockMovRepo, mockEstRepo, mockInvRepo)

			output, err := uc.CalculateSummary(context.Background(), tc.input.period)

			assert.ErrorIs(t, err, tc.expected.err)
			assert.Equal(t, tc.expected.output, output)
		})
	}
}
