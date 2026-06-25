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

func dashboardEstimate(amount float64, isIncome bool, categoryID *uuid.UUID) domain.EstimateCategories {
	return domain.EstimateCategories{
		CategoryID:       categoryID,
		IsCategoryIncome: isIncome,
		Amount:           amount,
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

	tests := map[string]struct {
		// input
		input input
		// mocks
		mockSetup func(mockMovRepo *MockMovementRepository, mockEstRepo *MockEstimateRepository)
		// expected
		expected expected
	}{
		"should build multi-month series filling gaps with zeros when months have no activity": {
			input: input{period: domain.Period{
				From: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2026, time.March, 31, 0, 0, 0, 0, time.UTC),
			}},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockEstRepo *MockEstimateRepository) {
				period := domain.Period{
					From: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
					To:   time.Date(2026, time.March, 31, 0, 0, 0, 0, time.UTC),
				}
				movements := domain.MovementList{
					dashboardMovement(5000, dashboardDate(2026, time.January, 10), true),
					dashboardMovement(-3000, dashboardDate(2026, time.January, 15), true),
					// February intentionally empty.
					dashboardMovement(4000, dashboardDate(2026, time.March, 5), true),
					dashboardMovement(-1000, dashboardDate(2026, time.March, 8), true),
				}
				mockMovRepo.On("FindByPeriod", period).Return(movements, nil)
				mockEstRepo.On("FindCategoriesByMonth", 3, 2026).
					Return([]domain.EstimateCategories{}, nil)
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
					KPIs: domain.DashboardKPIs{
						TotalIncome:       9000,
						TotalExpense:      -4000,
						AvgMonthlyIncome:  3000,
						AvgMonthlyExpense: -4000.0 / 3.0,
						PeriodNet:         5000,
						SavingsRate:       5000.0 / 9000.0,
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
			mockSetup: func(mockMovRepo *MockMovementRepository, mockEstRepo *MockEstimateRepository) {
				period := domain.Period{
					From: time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC),
					To:   time.Date(2026, time.June, 30, 0, 0, 0, 0, time.UTC),
				}
				incomeCat := uuid.New()
				expenseCat := uuid.New()
				movements := domain.MovementList{
					dashboardMovementWithCategory(4800, dashboardDate(2026, time.June, 10), true, &incomeCat),
					dashboardMovementWithCategory(-3200, dashboardDate(2026, time.June, 12), true, &expenseCat),
					// Unpaid movement must be ignored.
					dashboardMovementWithCategory(-1000, dashboardDate(2026, time.June, 20), false, &expenseCat),
				}
				mockMovRepo.On("FindByPeriod", period).Return(movements, nil)
				mockEstRepo.On("FindCategoriesByMonth", 6, 2026).Return([]domain.EstimateCategories{
					dashboardEstimate(5000, true, &incomeCat),
					dashboardEstimate(-3000, false, &expenseCat),
				}, nil)
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
					KPIs: domain.DashboardKPIs{
						TotalIncome:       4800,
						TotalExpense:      -3200,
						AvgMonthlyIncome:  4800,
						AvgMonthlyExpense: -3200,
						PeriodNet:         1600,
						SavingsRate:       1600.0 / 4800.0,
					},
				},
				err: nil,
			},
		},
		"should return zero savings rate when total income is zero": {
			input: input{period: domain.Period{
				From: time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2026, time.May, 31, 0, 0, 0, 0, time.UTC),
			}},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockEstRepo *MockEstimateRepository) {
				period := domain.Period{
					From: time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC),
					To:   time.Date(2026, time.May, 31, 0, 0, 0, 0, time.UTC),
				}
				movements := domain.MovementList{
					dashboardMovement(-2000, dashboardDate(2026, time.May, 10), true),
				}
				mockMovRepo.On("FindByPeriod", period).Return(movements, nil)
				mockEstRepo.On("FindCategoriesByMonth", 5, 2026).
					Return([]domain.EstimateCategories{}, nil)
			},
			expected: expected{
				output: domain.DashboardSummary{
					MonthlySeries: []domain.MonthlyPoint{
						{Month: 5, Year: 2026, Income: 0, Expense: -2000, Net: -2000},
					},
					CurrentMonth: domain.BudgetComparison{
						Month: 5, Year: 2026,
						Budget: domain.DashboardBudget{
							Income:  domain.BudgetLine{Budgeted: 0, Realized: 0},
							Expense: domain.BudgetLine{Budgeted: 0, Realized: -2000},
						},
					},
					KPIs: domain.DashboardKPIs{
						TotalIncome:       0,
						TotalExpense:      -2000,
						AvgMonthlyIncome:  0,
						AvgMonthlyExpense: -2000,
						PeriodNet:         -2000,
						SavingsRate:       0,
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
			mockSetup: func(mockMovRepo *MockMovementRepository, mockEstRepo *MockEstimateRepository) {
				period := domain.Period{
					From: time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC),
					To:   time.Date(2026, time.April, 30, 0, 0, 0, 0, time.UTC),
				}
				mockMovRepo.On("FindByPeriod", period).Return(domain.MovementList{}, nil)
				mockEstRepo.On("FindCategoriesByMonth", 4, 2026).
					Return([]domain.EstimateCategories{}, nil)
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
					KPIs: domain.DashboardKPIs{
						TotalIncome:       0,
						TotalExpense:      0,
						AvgMonthlyIncome:  0,
						AvgMonthlyExpense: 0,
						PeriodNet:         0,
						SavingsRate:       0,
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
			mockSetup: func(mockMovRepo *MockMovementRepository, mockEstRepo *MockEstimateRepository) {
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
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Arrange
			var (
				mockMovRepo = &MockMovementRepository{}
				mockEstRepo = &MockEstimateRepository{}
				uc          = NewDashboard(mockMovRepo, mockEstRepo)
			)
			defer mockMovRepo.AssertExpectations(t)
			defer mockEstRepo.AssertExpectations(t)
			tc.mockSetup(mockMovRepo, mockEstRepo)

			// Act
			output, err := uc.CalculateSummary(context.Background(), tc.input.period)

			// Assert
			assert.ErrorIs(t, err, tc.expected.err)
			assert.Equal(t, tc.expected.output, output)
		})
	}
}
