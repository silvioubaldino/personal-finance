package usecase

import (
	"context"
	"fmt"
	"time"

	"personal-finance/internal/domain"
)

type DashboardMovementRepository interface {
	FindByPeriod(ctx context.Context, period domain.Period) (domain.MovementList, error)
}

type DashboardEstimateRepository interface {
	FindCategoriesByMonth(ctx context.Context, month int, year int) ([]domain.EstimateCategories, error)
}

type DashboardUseCase interface {
	CalculateSummary(ctx context.Context, period domain.Period) (domain.DashboardSummary, error)
}

type dashboardUseCase struct {
	movementRepo DashboardMovementRepository
	estimateRepo DashboardEstimateRepository
}

func NewDashboard(movementRepo DashboardMovementRepository, estimateRepo DashboardEstimateRepository) DashboardUseCase {
	return dashboardUseCase{
		movementRepo: movementRepo,
		estimateRepo: estimateRepo,
	}
}

func (uc dashboardUseCase) CalculateSummary(ctx context.Context, period domain.Period) (domain.DashboardSummary, error) {
	if err := period.Validate(); err != nil {
		return domain.DashboardSummary{}, fmt.Errorf("período inválido: %w", err)
	}

	movements, err := uc.movementRepo.FindByPeriod(ctx, period)
	if err != nil {
		return domain.DashboardSummary{}, fmt.Errorf("error finding movements: %w", err)
	}

	paid := movements.GetPaidMovements()

	monthlySeries := buildMonthlySeries(period, paid)

	currentMonth, err := uc.buildCurrentMonth(ctx, period, paid)
	if err != nil {
		return domain.DashboardSummary{}, err
	}

	kpis := buildKPIs(monthlySeries)

	return domain.DashboardSummary{
		MonthlySeries: monthlySeries,
		CurrentMonth:  currentMonth,
		KPIs:          kpis,
	}, nil
}

// monthKey uniquely identifies a calendar month.
type monthKey struct {
	month int
	year  int
}

// buildMonthlySeries produces one ordered entry per calendar month in [from,to],
// filling months with no activity as zeros so the chart axis stays continuous.
func buildMonthlySeries(period domain.Period, paid domain.MovementList) []domain.MonthlyPoint {
	incomeByMonth := make(map[monthKey]float64)
	expenseByMonth := make(map[monthKey]float64)

	for _, m := range paid.GetIncomeMovements() {
		k := keyFromTime(*m.Date)
		incomeByMonth[k] += m.Amount
	}
	for _, m := range paid.GetExpenseMovements() {
		k := keyFromTime(*m.Date)
		expenseByMonth[k] += m.Amount
	}

	var series []domain.MonthlyPoint
	cursor := time.Date(period.From.Year(), period.From.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(period.To.Year(), period.To.Month(), 1, 0, 0, 0, 0, time.UTC)
	for !cursor.After(end) {
		k := monthKey{month: int(cursor.Month()), year: cursor.Year()}
		income := incomeByMonth[k]
		expense := expenseByMonth[k]
		series = append(series, domain.MonthlyPoint{
			Month:   k.month,
			Year:    k.year,
			Income:  income,
			Expense: expense,
			Net:     income + expense,
		})
		cursor = cursor.AddDate(0, 1, 0)
	}

	return series
}

// buildCurrentMonth computes budgeted vs realized for the month of period.To.
func (uc dashboardUseCase) buildCurrentMonth(
	ctx context.Context,
	period domain.Period,
	paid domain.MovementList,
) (domain.BudgetComparison, error) {
	month := int(period.To.Month())
	year := period.To.Year()

	estimates, err := uc.estimateRepo.FindCategoriesByMonth(ctx, month, year)
	if err != nil {
		return domain.BudgetComparison{}, fmt.Errorf("error finding estimates: %w", err)
	}

	estimateList := domain.EstimateCategoriesList(estimates)

	paid = filterByMonth(paid, month, year)

	expenseSumByCategory := paid.GetExpenseMovements().GetSumByCategory()
	expenseEstimates := estimateList.GetExpenseEstimates().GetEstimateByCategory()
	expenseBudgeted := sumMapValues(expenseEstimates)
	expenseRealized := getBalanceSum(expenseEstimates, expenseSumByCategory, false)

	incomeSumByCategory := paid.GetIncomeMovements().GetSumByCategory()
	incomeEstimates := estimateList.GetIncomeEstimates().GetEstimateByCategory()
	incomeBudgeted := sumMapValues(incomeEstimates)
	incomeRealized := getBalanceSum(incomeEstimates, incomeSumByCategory, true)

	return domain.BudgetComparison{
		Month: month,
		Year:  year,
		Budget: domain.DashboardBudget{
			Income: domain.BudgetLine{
				Budgeted: incomeBudgeted,
				Realized: incomeRealized,
			},
			Expense: domain.BudgetLine{
				Budgeted: expenseBudgeted,
				Realized: expenseRealized,
			},
		},
	}, nil
}

// buildKPIs aggregates the monthly series into period-wide totals and averages.
func buildKPIs(series []domain.MonthlyPoint) domain.DashboardKPIs {
	var totalIncome, totalExpense float64
	for _, p := range series {
		totalIncome += p.Income
		totalExpense += p.Expense
	}

	months := float64(len(series))
	var avgIncome, avgExpense float64
	if months > 0 {
		avgIncome = totalIncome / months
		avgExpense = totalExpense / months
	}

	periodNet := totalIncome + totalExpense

	var savingsRate float64
	if totalIncome > 0 {
		savingsRate = periodNet / totalIncome
	}

	return domain.DashboardKPIs{
		TotalIncome:       totalIncome,
		TotalExpense:      totalExpense,
		AvgMonthlyIncome:  avgIncome,
		AvgMonthlyExpense: avgExpense,
		PeriodNet:         periodNet,
		SavingsRate:       savingsRate,
	}
}

// filterByMonth returns only movements whose date falls in the given month/year.
func filterByMonth(movements domain.MovementList, month, year int) domain.MovementList {
	var filtered domain.MovementList
	for _, m := range movements {
		if m.Date != nil && int(m.Date.Month()) == month && m.Date.Year() == year {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

func keyFromTime(t time.Time) monthKey {
	return monthKey{month: int(t.Month()), year: t.Year()}
}

func sumMapValues[K comparable](m map[K]float64) float64 {
	var total float64
	for _, v := range m {
		total += v
	}
	return total
}
