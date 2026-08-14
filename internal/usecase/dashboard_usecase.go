package usecase

import (
	"context"
	"fmt"
	"sort"
	"time"

	"personal-finance/internal/domain"

	"github.com/google/uuid"
)

type DashboardMovementRepository interface {
	FindByPeriod(ctx context.Context, period domain.Period) (domain.MovementList, error)
}

type DashboardEstimateRepository interface {
	FindCategoriesByMonth(ctx context.Context, month int, year int) ([]domain.EstimateCategories, error)
}

type DashboardInvoiceRepository interface {
	FindByPeriod(ctx context.Context, period domain.Period) ([]domain.Invoice, error)
}

type DashboardUseCase interface {
	CalculateSummary(ctx context.Context, period domain.Period) (domain.DashboardSummary, error)
}

type dashboardUseCase struct {
	movementRepo DashboardMovementRepository
	estimateRepo DashboardEstimateRepository
	invoiceRepo  DashboardInvoiceRepository
}

func NewDashboard(
	movementRepo DashboardMovementRepository,
	estimateRepo DashboardEstimateRepository,
	invoiceRepo DashboardInvoiceRepository,
) DashboardUseCase {
	return dashboardUseCase{
		movementRepo: movementRepo,
		estimateRepo: estimateRepo,
		invoiceRepo:  invoiceRepo,
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

	invoices, err := uc.invoiceRepo.FindByPeriod(ctx, period)
	if err != nil {
		return domain.DashboardSummary{}, fmt.Errorf("error finding invoices: %w", err)
	}

	kpis := buildKPIs(monthlySeries)

	return domain.DashboardSummary{
		MonthlySeries:      monthlySeries,
		CurrentMonth:       currentMonth,
		CreditCardInvoices: buildCreditCardInvoices(period, invoices),
		// A distribuição por dia da semana mede comportamento de compra, não caixa
		// realizado: usa todas as despesas do período (pagas e pendentes), porque
		// compra no cartão só fica paga quando a fatura é paga (AYD-003, decisão #7).
		ExpenseWeekdayDistribution: buildExpenseWeekdayDistribution(movements),
		KPIs:                       kpis,
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

// buildKPIs aggregates the monthly series into period-wide totals.
func buildKPIs(series []domain.MonthlyPoint) domain.DashboardKPIs {
	var totalIncome, totalExpense float64
	for _, p := range series {
		totalIncome += p.Income
		totalExpense += p.Expense
	}

	return domain.DashboardKPIs{
		TotalIncome:  totalIncome,
		TotalExpense: totalExpense,
	}
}

// buildCreditCardInvoices groups invoice totals per calendar month, stacked by
// credit card. An invoice belongs to the month of its due_date, the same
// convention InvoiceRepository.FindByMonth uses.
func buildCreditCardInvoices(period domain.Period, invoices []domain.Invoice) domain.CreditCardInvoiceSummary {
	amountByMonthAndCard := make(map[monthKey]map[uuid.UUID]float64)
	cardNames := make(map[uuid.UUID]string)

	for _, invoice := range invoices {
		if invoice.CreditCardID == nil {
			continue
		}

		cardID := *invoice.CreditCardID
		if _, seen := cardNames[cardID]; !seen {
			cardNames[cardID] = invoice.CreditCard.Name
		}

		k := keyFromTime(invoice.DueDate)
		if amountByMonthAndCard[k] == nil {
			amountByMonthAndCard[k] = make(map[uuid.UUID]float64)
		}
		amountByMonthAndCard[k][cardID] += invoice.Amount
	}

	cards := buildCardLegend(cardNames)

	// Uma entrada por mês do span, alinhada com monthly_series; by_card sempre
	// traz todos os cartões (0 onde não houve fatura) para o empilhamento não
	// trocar de cor entre meses.
	series := make([]domain.CreditCardInvoicePoint, 0)
	cursor := time.Date(period.From.Year(), period.From.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(period.To.Year(), period.To.Month(), 1, 0, 0, 0, 0, time.UTC)
	for !cursor.After(end) {
		k := monthKey{month: int(cursor.Month()), year: cursor.Year()}

		byCard := make([]domain.CreditCardInvoiceSlice, 0, len(cards))
		var total float64
		for _, card := range cards {
			amount := amountByMonthAndCard[k][*card.CreditCardID]
			total += amount
			byCard = append(byCard, domain.CreditCardInvoiceSlice{
				CreditCardID: card.CreditCardID,
				Amount:       amount,
			})
		}

		series = append(series, domain.CreditCardInvoicePoint{
			Month:  k.month,
			Year:   k.year,
			Total:  total,
			ByCard: byCard,
		})
		cursor = cursor.AddDate(0, 1, 0)
	}

	return domain.CreditCardInvoiceSummary{Cards: cards, Series: series}
}

// buildCardLegend orders the cards by name so the stacking order is stable.
func buildCardLegend(cardNames map[uuid.UUID]string) []domain.CreditCardRef {
	cards := make([]domain.CreditCardRef, 0, len(cardNames))
	for id, name := range cardNames {
		cardID := id
		cards = append(cards, domain.CreditCardRef{CreditCardID: &cardID, Name: name})
	}
	sort.Slice(cards, func(i, j int) bool {
		if cards[i].Name == cards[j].Name {
			return cards[i].CreditCardID.String() < cards[j].CreditCardID.String()
		}
		return cards[i].Name < cards[j].Name
	})
	return cards
}

// buildExpenseWeekdayDistribution measures purchase behaviour: how the *count*
// of expense movements spreads over the days of the week. It deliberately
// counts unpaid movements too — a credit card purchase stays unpaid until the
// invoice is paid, so filtering by paid would erase card purchases. Internal
// transfers are excluded: they move money between the user's own wallets and
// are not purchases (AYD-003, decisão #7).
func buildExpenseWeekdayDistribution(movements domain.MovementList) []domain.ExpenseWeekdayPoint {
	counts := make([]int, 7)
	total := 0

	for _, m := range movements.GetExpenseMovements() {
		if m.Date == nil || m.TypePayment == domain.TypePaymentInternalTransfer {
			continue
		}
		counts[int(m.Date.Weekday())]++
		total++
	}

	distribution := make([]domain.ExpenseWeekdayPoint, 7)
	for weekday, count := range counts {
		var percentage float64
		if total > 0 {
			percentage = float64(count) / float64(total)
		}
		distribution[weekday] = domain.ExpenseWeekdayPoint{
			Weekday:    weekday,
			Count:      count,
			Percentage: percentage,
		}
	}

	return distribution
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
