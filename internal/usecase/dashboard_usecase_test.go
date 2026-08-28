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

// dashboardMovementWithCategory monta um Movement de despesa (Category.IsIncome = false).
// A classificação receita × despesa vem da flag da Category, não do sinal
// (AYD-005@context) — para receita use dashboardIncomeMovement.
func dashboardMovementWithCategory(amount float64, date *time.Time, isPaid bool, categoryID *uuid.UUID) domain.Movement {
	return domain.Movement{
		Amount:     amount,
		Date:       date,
		IsPaid:     isPaid,
		CategoryID: categoryID,
		Category:   domain.Category{ID: categoryID},
	}
}

func dashboardIncomeMovement(amount float64, date *time.Time, isPaid bool, categoryID *uuid.UUID) domain.Movement {
	movement := dashboardMovementWithCategory(amount, date, isPaid, categoryID)
	movement.Category.IsIncome = true
	return movement
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

func dashboardDetailedInvoice(invoice domain.Invoice, movements ...domain.Movement) domain.DetailedInvoice {
	return domain.DetailedInvoice{Invoice: invoice, Movements: movements}
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
			mockInvoiceUC *MockInvoice,
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
				mockInvoiceUC *MockInvoice,
			) {
				period := domain.Period{
					From: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
					To:   time.Date(2026, time.March, 31, 0, 0, 0, 0, time.UTC),
				}
				janIncomeCat := uuid.New()
				marIncomeCat := uuid.New()
				movements := domain.MovementList{
					dashboardIncomeMovement(5000, dashboardDate(2026, time.January, 10), true, &janIncomeCat),
					dashboardMovementWithCategory(-3000, dashboardDate(2026, time.January, 15), true, &janExpenseCategoryID),
					dashboardIncomeMovement(4000, dashboardDate(2026, time.March, 5), true, &marIncomeCat),
					dashboardMovementWithCategory(-1000, dashboardDate(2026, time.March, 8), true, &marExpenseCategoryID),
				}
				mockMovRepo.On("FindByPeriod", period).Return(movements, nil)
				mockEstRepo.On("FindCategoriesByMonth", 3, 2026).
					Return([]domain.EstimateCategories{}, nil)
				mockInvoiceUC.On("FindDetailedInvoicesByPeriod", context.Background(), period).
					Return([]domain.DetailedInvoice{}, nil)
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
		"should report realized as the plain paid sum, without the legacy budget floor": {
			input: input{period: domain.Period{
				From: time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2026, time.June, 30, 0, 0, 0, 0, time.UTC),
			}},
			mockSetup: func(
				mockMovRepo *MockMovementRepository,
				mockEstRepo *MockEstimateRepository,
				mockInvoiceUC *MockInvoice,
			) {
				period := domain.Period{
					From: time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC),
					To:   time.Date(2026, time.June, 30, 0, 0, 0, 0, time.UTC),
				}
				movements := domain.MovementList{
					dashboardIncomeMovement(4800, dashboardDate(2026, time.June, 10), true, &incomeCat),
					dashboardMovementWithCategory(-3200, dashboardDate(2026, time.June, 12), true, &expenseCat),
					dashboardMovementWithCategory(-1000, dashboardDate(2026, time.June, 20), false, &expenseCat),
				}
				mockMovRepo.On("FindByPeriod", period).Return(movements, nil)
				mockEstRepo.On("FindCategoriesByMonth", 6, 2026).Return([]domain.EstimateCategories{
					dashboardEstimate(5000, true, &incomeCat),
					dashboardEstimate(-3000, false, &expenseCat),
				}, nil)
				mockInvoiceUC.On("FindDetailedInvoicesByPeriod", context.Background(), period).
					Return([]domain.DetailedInvoice{}, nil)
			},
			expected: expected{
				output: domain.DashboardSummary{
					MonthlySeries: []domain.MonthlyPoint{
						{Month: 6, Year: 2026, Income: 4800, Expense: -3200, Net: 1600},
					},
					CurrentMonth: domain.BudgetComparison{
						Month: 6, Year: 2026,
						Budget: domain.DashboardBudget{
							// 4800 realizado contra 5000 orçado: o teto/piso do Balance
							// mostrava 5000 aqui; realized_paid mostra o que entrou.
							Income:  domain.BudgetLine{Budgeted: 5000, Realized: 4800},
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
				mockInvoiceUC *MockInvoice,
			) {
				period := domain.Period{
					From: time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC),
					To:   time.Date(2026, time.April, 30, 0, 0, 0, 0, time.UTC),
				}
				mockMovRepo.On("FindByPeriod", period).Return(domain.MovementList{}, nil)
				mockEstRepo.On("FindCategoriesByMonth", 4, 2026).
					Return([]domain.EstimateCategories{}, nil)
				mockInvoiceUC.On("FindDetailedInvoicesByPeriod", context.Background(), period).
					Return([]domain.DetailedInvoice{}, nil)
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
				mockInvoiceUC *MockInvoice,
			) {
				period := domain.Period{
					From: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
					To:   time.Date(2026, time.February, 28, 0, 0, 0, 0, time.UTC),
				}
				mockMovRepo.On("FindByPeriod", period).Return(domain.MovementList{}, nil)
				mockEstRepo.On("FindCategoriesByMonth", 2, 2026).
					Return([]domain.EstimateCategories{}, nil)
				mockInvoiceUC.On("FindDetailedInvoicesByPeriod", context.Background(), period).
					Return([]domain.DetailedInvoice{
						dashboardDetailedInvoice(dashboardInvoice(&nubankID, "Nubank", "#820ad1", dashboardDate(2026, time.January, 10), -1400)),
						dashboardDetailedInvoice(dashboardInvoice(&itauID, "Itau", "", dashboardDate(2026, time.January, 15), -700)),
						dashboardDetailedInvoice(dashboardInvoice(&nubankID, "Nubank", "#820ad1", dashboardDate(2026, time.February, 10), -900)),
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
				mockInvoiceUC *MockInvoice,
			) {
				period := domain.Period{
					From: time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC),
					To:   time.Date(2026, time.February, 28, 0, 0, 0, 0, time.UTC),
				}
				febIncomeCat := uuid.New()
				movements := domain.MovementList{
					dashboardMovementWithCategory(-100, dashboardDate(2026, time.February, 6), true, &weekdayCat1ID),
					dashboardMovementWithCategory(-300, dashboardDate(2026, time.February, 20), true, &weekdayCat2ID),
					dashboardMovementWithCategory(-400, dashboardDate(2026, time.February, 17), true, &weekdayCat3ID),
					dashboardMovementWithPayment(
						-500, dashboardDate(2026, time.February, 17), true, domain.TypePaymentInternalTransfer,
					),
					dashboardIncomeMovement(1000, dashboardDate(2026, time.February, 6), true, &febIncomeCat),
				}
				mockMovRepo.On("FindByPeriod", period).Return(movements, nil)
				mockEstRepo.On("FindCategoriesByMonth", 2, 2026).
					Return([]domain.EstimateCategories{}, nil)
				// Compra no cartão de uma fatura ainda aberta: entra no dia da semana
				// (comportamento de compra) e fica fora do realizado, que é só pago.
				mockInvoiceUC.On("FindDetailedInvoicesByPeriod", context.Background(), period).
					Return([]domain.DetailedInvoice{
						dashboardDetailedInvoice(
							dashboardInvoice(&nubankID, "Nubank", "#820ad1", dashboardDate(2026, time.February, 10), -200),
							dashboardMovementWithCategory(-200, dashboardDate(2026, time.February, 13), false, &foodCategoryID),
						),
					}, nil)
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
					CreditCardInvoices: domain.CreditCardInvoiceSummary{
						Cards: []domain.CreditCardRef{
							{CreditCardID: &nubankID, Name: "Nubank", Color: "#820ad1"},
						},
						Series: []domain.CreditCardInvoicePoint{
							{
								Month: 2, Year: 2026, Total: -200,
								ByCard: []domain.CreditCardInvoiceSlice{
									{CreditCardID: &nubankID, Amount: -200},
								},
							},
						},
					},
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
				mockInvoiceUC *MockInvoice,
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
				mockInvoiceUC.On("FindDetailedInvoicesByPeriod", context.Background(), period).
					Return([]domain.DetailedInvoice{}, nil)
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
		"should count the itemized card purchases once, never the invoice_payment on top of them": {
			input: input{period: domain.Period{
				From: time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2026, time.February, 28, 0, 0, 0, 0, time.UTC),
			}},
			mockSetup: func(
				mockMovRepo *MockMovementRepository,
				mockEstRepo *MockEstimateRepository,
				mockInvoiceUC *MockInvoice,
			) {
				period := domain.Period{
					From: time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC),
					To:   time.Date(2026, time.February, 28, 0, 0, 0, 0, time.UTC),
				}
				// A fatura paga deixa no banco as duas pontas do mesmo dinheiro: as compras
				// (marcadas como pagas) e o invoice_payment de -350. Só as compras contam.
				invoicePayment := domain.Movement{
					Amount: -350, Date: dashboardDate(2026, time.February, 10), IsPaid: true,
					CategoryID: &invoiceCategoryID, Category: domain.Category{ID: &invoiceCategoryID},
					TypePayment: domain.TypePaymentInvoicePayment,
				}
				mockMovRepo.On("FindByPeriod", period).Return(domain.MovementList{invoicePayment}, nil)
				mockEstRepo.On("FindCategoriesByMonth", 2, 2026).
					Return([]domain.EstimateCategories{}, nil)
				mockInvoiceUC.On("FindDetailedInvoicesByPeriod", context.Background(), period).
					Return([]domain.DetailedInvoice{
						dashboardDetailedInvoice(
							dashboardInvoice(&nubankID, "Nubank", "#820ad1", dashboardDate(2026, time.February, 10), -350),
							domain.Movement{
								Amount: -300, Date: dashboardDate(2026, time.February, 5), IsPaid: true,
								CategoryID: &foodCategoryID, Category: domain.Category{ID: &foodCategoryID},
								TypePayment: domain.TypePaymentCreditCard,
							},
							// Remanescente da fatura anterior: nunca fica pago, então conta
							// no dia da semana? não — não é compra — e fica fora do realizado.
							domain.Movement{
								Amount: -50, Date: dashboardDate(2026, time.February, 6), IsPaid: false,
								CategoryID: &invoiceCategoryID, Category: domain.Category{ID: &invoiceCategoryID},
								TypePayment: domain.TypePaymentInvoiceRemainder,
							},
						),
					}, nil)
			},
			expected: expected{
				output: domain.DashboardSummary{
					MonthlySeries: []domain.MonthlyPoint{
						{Month: 2, Year: 2026, Income: 0, Expense: -300, Net: -300},
					},
					CurrentMonth: domain.BudgetComparison{
						Month: 2, Year: 2026,
						Budget: domain.DashboardBudget{
							Income:  domain.BudgetLine{Budgeted: 0, Realized: 0},
							Expense: domain.BudgetLine{Budgeted: 0, Realized: -300},
						},
					},
					CreditCardInvoices: domain.CreditCardInvoiceSummary{
						Cards: []domain.CreditCardRef{
							{CreditCardID: &nubankID, Name: "Nubank", Color: "#820ad1"},
						},
						Series: []domain.CreditCardInvoicePoint{
							{
								Month: 2, Year: 2026, Total: -350,
								ByCard: []domain.CreditCardInvoiceSlice{
									{CreditCardID: &nubankID, Amount: -350},
								},
							},
						},
					},
					ExpenseWeekdayDistribution: expenseWeekdays(
						dashboardDate(2026, time.February, 5),
					),
					ExpenseByCategory: []domain.CategoryExpensePoint{
						{CategoryID: &foodCategoryID, Total: -300},
					},
					KPIs: domain.DashboardKPIs{
						TotalIncome:  0,
						TotalExpense: -300,
					},
				},
				err: nil,
			},
		},
		"should count a card purchase in the month of its invoice due date, not of the purchase": {
			input: input{period: domain.Period{
				From: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2026, time.February, 28, 0, 0, 0, 0, time.UTC),
			}},
			mockSetup: func(
				mockMovRepo *MockMovementRepository,
				mockEstRepo *MockEstimateRepository,
				mockInvoiceUC *MockInvoice,
			) {
				period := domain.Period{
					From: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
					To:   time.Date(2026, time.February, 28, 0, 0, 0, 0, time.UTC),
				}
				mockMovRepo.On("FindByPeriod", period).Return(domain.MovementList{}, nil)
				mockEstRepo.On("FindCategoriesByMonth", 2, 2026).
					Return([]domain.EstimateCategories{}, nil)
				mockInvoiceUC.On("FindDetailedInvoicesByPeriod", context.Background(), period).
					Return([]domain.DetailedInvoice{
						dashboardDetailedInvoice(
							dashboardInvoice(&nubankID, "Nubank", "#820ad1", dashboardDate(2026, time.February, 10), -300),
							domain.Movement{
								Amount: -300, Date: dashboardDate(2026, time.January, 30), IsPaid: true,
								CategoryID: &foodCategoryID, Category: domain.Category{ID: &foodCategoryID},
								TypePayment: domain.TypePaymentCreditCard,
							},
						),
					}, nil)
			},
			expected: expected{
				output: domain.DashboardSummary{
					MonthlySeries: []domain.MonthlyPoint{
						{Month: 1, Year: 2026, Income: 0, Expense: 0, Net: 0},
						{Month: 2, Year: 2026, Income: 0, Expense: -300, Net: -300},
					},
					CurrentMonth: domain.BudgetComparison{
						Month: 2, Year: 2026,
						Budget: domain.DashboardBudget{
							Income:  domain.BudgetLine{Budgeted: 0, Realized: 0},
							Expense: domain.BudgetLine{Budgeted: 0, Realized: -300},
						},
					},
					CreditCardInvoices: domain.CreditCardInvoiceSummary{
						Cards: []domain.CreditCardRef{
							{CreditCardID: &nubankID, Name: "Nubank", Color: "#820ad1"},
						},
						Series: []domain.CreditCardInvoicePoint{
							{Month: 1, Year: 2026, Total: 0, ByCard: []domain.CreditCardInvoiceSlice{
								{CreditCardID: &nubankID, Amount: 0},
							}},
							{Month: 2, Year: 2026, Total: -300, ByCard: []domain.CreditCardInvoiceSlice{
								{CreditCardID: &nubankID, Amount: -300},
							}},
						},
					},
					// O dia da semana continua sendo o da compra, não o do vencimento.
					ExpenseWeekdayDistribution: expenseWeekdays(
						dashboardDate(2026, time.January, 30),
					),
					ExpenseByCategory: []domain.CategoryExpensePoint{
						{CategoryID: &foodCategoryID, Total: -300},
					},
					KPIs: domain.DashboardKPIs{
						TotalIncome:  0,
						TotalExpense: -300,
					},
				},
				err: nil,
			},
		},
		"should exclude movements tagged with the internal transfer category IDs from every aggregate": {
			input: input{period: domain.Period{
				From: time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2026, time.February, 28, 0, 0, 0, 0, time.UTC),
			}},
			mockSetup: func(
				mockMovRepo *MockMovementRepository,
				mockEstRepo *MockEstimateRepository,
				mockInvoiceUC *MockInvoice,
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
				mockInvoiceUC.On("FindDetailedInvoicesByPeriod", context.Background(), period).
					Return([]domain.DetailedInvoice{}, nil)
			},
			expected: expected{
				output: domain.DashboardSummary{
					MonthlySeries: []domain.MonthlyPoint{
						{Month: 2, Year: 2026, Income: 0, Expense: -300, Net: -300},
					},
					CurrentMonth: domain.BudgetComparison{
						Month: 2, Year: 2026,
						Budget: domain.DashboardBudget{
							Income:  domain.BudgetLine{Budgeted: 0, Realized: 0},
							Expense: domain.BudgetLine{Budgeted: 0, Realized: -300},
						},
					},
					CreditCardInvoices: emptyInvoiceSummary(2026, time.February),
					ExpenseWeekdayDistribution: expenseWeekdays(
						dashboardDate(2026, time.February, 5),
					),
					ExpenseByCategory: []domain.CategoryExpensePoint{
						{CategoryID: &foodCategoryID, Total: -300},
					},
					KPIs: domain.DashboardKPIs{
						TotalIncome:  0,
						TotalExpense: -300,
					},
				},
				err: nil,
			},
		},
		"should classify weekday distribution by category is_income, not by amount sign": {
			input: input{period: domain.Period{
				From: time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2026, time.February, 28, 0, 0, 0, 0, time.UTC),
			}},
			mockSetup: func(
				mockMovRepo *MockMovementRepository,
				mockEstRepo *MockEstimateRepository,
				mockInvoiceUC *MockInvoice,
			) {
				period := domain.Period{
					From: time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC),
					To:   time.Date(2026, time.February, 28, 0, 0, 0, 0, time.UTC),
				}
				incomeCategoryID := uuid.New()
				movements := domain.MovementList{
					// entrada sem categoria caiu no fallback de despesa (AYD-006) mas
					// continua entrada: is_income decide, não o sinal positivo.
					{
						Amount: 300, Date: dashboardDate(2026, time.February, 5), IsPaid: true,
						CategoryID: &foodCategoryID,
						Category:   domain.Category{ID: &foodCategoryID, IsIncome: false},
					},
					// estorno em categoria de receita: sinal negativo não vira despesa.
					{
						Amount: -900, Date: dashboardDate(2026, time.February, 6), IsPaid: true,
						CategoryID: &incomeCategoryID,
						Category:   domain.Category{ID: &incomeCategoryID, IsIncome: true},
					},
				}
				mockMovRepo.On("FindByPeriod", period).Return(movements, nil)
				mockEstRepo.On("FindCategoriesByMonth", 2, 2026).
					Return([]domain.EstimateCategories{}, nil)
				mockInvoiceUC.On("FindDetailedInvoicesByPeriod", context.Background(), period).
					Return([]domain.DetailedInvoice{}, nil)
			},
			expected: expected{
				output: domain.DashboardSummary{
					MonthlySeries: []domain.MonthlyPoint{
						{Month: 2, Year: 2026, Income: -900, Expense: 300, Net: -600},
					},
					CurrentMonth: domain.BudgetComparison{
						Month: 2, Year: 2026,
						Budget: domain.DashboardBudget{
							Income:  domain.BudgetLine{Budgeted: 0, Realized: -900},
							Expense: domain.BudgetLine{Budgeted: 0, Realized: 300},
						},
					},
					CreditCardInvoices: emptyInvoiceSummary(2026, time.February),
					ExpenseWeekdayDistribution: expenseWeekdays(
						dashboardDate(2026, time.February, 5),
					),
					ExpenseByCategory: []domain.CategoryExpensePoint{
						{CategoryID: &foodCategoryID, Total: 300},
					},
					KPIs: domain.DashboardKPIs{
						TotalIncome:  -900,
						TotalExpense: 300,
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
				_ *MockInvoice,
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
		"should return error when invoice lookup fails": {
			input: input{period: domain.Period{
				From: time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2026, time.August, 31, 0, 0, 0, 0, time.UTC),
			}},
			mockSetup: func(
				mockMovRepo *MockMovementRepository,
				_ *MockEstimateRepository,
				mockInvoiceUC *MockInvoice,
			) {
				period := domain.Period{
					From: time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC),
					To:   time.Date(2026, time.August, 31, 0, 0, 0, 0, time.UTC),
				}
				mockMovRepo.On("FindByPeriod", period).Return(domain.MovementList{}, nil)
				mockInvoiceUC.On("FindDetailedInvoicesByPeriod", context.Background(), period).
					Return([]domain.DetailedInvoice{}, assert.AnError)
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
				mockMovRepo   = &MockMovementRepository{}
				mockEstRepo   = &MockEstimateRepository{}
				mockInvoiceUC = &MockInvoice{}
				uc            = NewDashboard(mockMovRepo, mockEstRepo, mockInvoiceUC)
			)
			defer mockMovRepo.AssertExpectations(t)
			defer mockEstRepo.AssertExpectations(t)
			defer mockInvoiceUC.AssertExpectations(t)
			tc.mockSetup(mockMovRepo, mockEstRepo, mockInvoiceUC)

			output, err := uc.CalculateSummary(context.Background(), tc.input.period)

			assert.ErrorIs(t, err, tc.expected.err)
			assert.Equal(t, tc.expected.output, output)
		})
	}
}

// TestDashboard_CalculateSummary_Reconciles trava os invariantes de conciliação do payload:
// todo bloco de dinheiro sai da mesma base, então os totais têm de fechar entre si. Foi
// justamente um filtro a mais em expense_by_category que abriu uma diferença de R$ 2.650,45
// contra kpis.total_expense em produção.
func TestDashboard_CalculateSummary_Reconciles(t *testing.T) {
	var (
		period = domain.Period{
			From: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
			To:   time.Date(2026, time.March, 31, 0, 0, 0, 0, time.UTC),
		}
		salaryID   = uuid.New()
		foodID     = uuid.New()
		travelID   = uuid.New()
		nubankID   = uuid.New()
		noCatMovID = uuid.New()
	)

	expenseCategory := func(id uuid.UUID) domain.Category {
		return domain.Category{ID: &id, IsIncome: false}
	}

	movements := domain.MovementList{
		dashboardIncomeMovement(9000, dashboardDate(2026, time.January, 5), true, &salaryID),
		dashboardMovementWithCategory(-1200, dashboardDate(2026, time.January, 9), true, &foodID),
		// Viagem: gastou 800 em janeiro e recebeu 1500 de reembolso em março, fechando o
		// período POSITIVA (+700) numa categoria de despesa. É o caso que abriu a
		// divergência em produção.
		dashboardMovementWithCategory(-800, dashboardDate(2026, time.January, 20), true, &travelID),
		{
			Amount: 1500, Date: dashboardDate(2026, time.March, 3), IsPaid: true,
			CategoryID: &travelID, Category: expenseCategory(travelID),
		},
		// Pendente: fora de todo agregado de dinheiro.
		dashboardMovementWithCategory(-5000, dashboardDate(2026, time.February, 2), false, &foodID),
		// Sem Category: não dá para classificar nem agrupar, fica fora de todos.
		{ID: &noCatMovID, Amount: -77, Date: dashboardDate(2026, time.February, 4), IsPaid: true},
	}

	mockMovRepo := &MockMovementRepository{}
	mockEstRepo := &MockEstimateRepository{}
	mockInvoiceUC := &MockInvoice{}

	mockMovRepo.On("FindByPeriod", period).Return(movements, nil)
	mockEstRepo.On("FindCategoriesByMonth", 3, 2026).Return([]domain.EstimateCategories{}, nil)
	mockInvoiceUC.On("FindDetailedInvoicesByPeriod", context.Background(), period).
		Return([]domain.DetailedInvoice{
			dashboardDetailedInvoice(
				dashboardInvoice(&nubankID, "Nubank", "#820ad1", dashboardDate(2026, time.March, 10), -450),
				domain.Movement{
					Amount: -450, Date: dashboardDate(2026, time.February, 25), IsPaid: true,
					CategoryID: &foodID, Category: expenseCategory(foodID),
					TypePayment: domain.TypePaymentCreditCard,
				},
			),
		}, nil)

	summary, err := NewDashboard(mockMovRepo, mockEstRepo, mockInvoiceUC).
		CalculateSummary(context.Background(), period)

	assert.NoError(t, err)

	var seriesIncome, seriesExpense float64
	for _, point := range summary.MonthlySeries {
		seriesIncome += point.Income
		seriesExpense += point.Expense
		assert.InDelta(t, point.Income+point.Expense, point.Net, 0.001, "net de cada mês")
	}

	var categoriesTotal float64
	for _, point := range summary.ExpenseByCategory {
		categoriesTotal += point.Total
	}

	assert.InDelta(t, summary.KPIs.TotalIncome, seriesIncome, 0.001,
		"kpis.total_income tem de ser a soma de monthly_series[].income")
	assert.InDelta(t, summary.KPIs.TotalExpense, seriesExpense, 0.001,
		"kpis.total_expense tem de ser a soma de monthly_series[].expense")
	assert.InDelta(t, summary.KPIs.TotalExpense, categoriesTotal, 0.001,
		"a soma de expense_by_category tem de bater com kpis.total_expense")

	// Março: reembolso de +1500 (categoria de despesa) e a fatura do Nubank de -450.
	marchBudget := summary.CurrentMonth.Budget
	assert.InDelta(t, 0, marchBudget.Income.Realized, 0.001,
		"nenhuma receita em março: o reembolso está em categoria de despesa")
	assert.InDelta(t, 1050, marchBudget.Expense.Realized, 0.001,
		"budget.realized de março tem de ser a fatia de monthly_series do mesmo mês")

	for _, point := range summary.MonthlySeries {
		if point.Month == 3 {
			assert.InDelta(t, marchBudget.Expense.Realized, point.Expense, 0.001,
				"current_month.budget.expense.realized == monthly_series do mês de `to`")
			assert.InDelta(t, marchBudget.Income.Realized, point.Income, 0.001,
				"current_month.budget.income.realized == monthly_series do mês de `to`")
		}
	}

	assert.Equal(t, 9000.0, summary.KPIs.TotalIncome, "só o salário é receita")
	assert.Equal(t, -950.0, summary.KPIs.TotalExpense,
		"-1200 -800 -450 do cartão +1500 de reembolso; o pendente e o sem-categoria ficam fora")

	// A categoria que fechou positiva continua no array, com o total positivo: é ela que
	// faz a soma bater. Tirá-la é o que abria a diferença.
	totalByCategory := make(map[uuid.UUID]float64, len(summary.ExpenseByCategory))
	for _, point := range summary.ExpenseByCategory {
		totalByCategory[*point.CategoryID] = point.Total
	}
	assert.Equal(t, -1650.0, totalByCategory[foodID], "alimentação: -1200 avulso e -450 no cartão")
	assert.Equal(t, 700.0, totalByCategory[travelID], "viagem fechou positiva: -800 +1500")
}
