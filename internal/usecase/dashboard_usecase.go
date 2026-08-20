package usecase

import (
	"context"
	"fmt"
	"math"
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
		MonthlySeries:              monthlySeries,
		CurrentMonth:               currentMonth,
		CreditCardInvoices:         buildCreditCardInvoices(period, invoices),
		ExpenseWeekdayDistribution: buildExpenseWeekdayDistribution(movements),
		ExpenseByCategory:          buildExpenseByCategory(paid),
		KPIs:                       kpis,
	}, nil
}

type monthKey struct {
	month int
	year  int
}

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

func buildCreditCardInvoices(period domain.Period, invoices []domain.Invoice) domain.CreditCardInvoiceSummary {
	amountByMonthAndCard := make(map[monthKey]map[uuid.UUID]float64)
	cardsByID := make(map[uuid.UUID]domain.CreditCardRef)

	for _, invoice := range invoices {
		if invoice.CreditCardID == nil {
			continue
		}

		cardID := *invoice.CreditCardID
		if _, seen := cardsByID[cardID]; !seen {
			id := cardID
			cardsByID[cardID] = domain.CreditCardRef{
				CreditCardID: &id,
				Name:         invoice.CreditCard.Name,
				Color:        invoice.CreditCard.Color,
			}
		}

		k := keyFromTime(invoice.DueDate)
		if amountByMonthAndCard[k] == nil {
			amountByMonthAndCard[k] = make(map[uuid.UUID]float64)
		}
		amountByMonthAndCard[k][cardID] += invoice.Amount
	}

	cards := buildCardLegend(cardsByID)

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

func buildCardLegend(cardsByID map[uuid.UUID]domain.CreditCardRef) []domain.CreditCardRef {
	cards := make([]domain.CreditCardRef, 0, len(cardsByID))
	for _, card := range cardsByID {
		cards = append(cards, card)
	}
	sort.Slice(cards, func(i, j int) bool {
		if cards[i].Name == cards[j].Name {
			return cards[i].CreditCardID.String() < cards[j].CreditCardID.String()
		}
		return cards[i].Name < cards[j].Name
	})
	return cards
}

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

var internalTransferCategoryIDs = map[uuid.UUID]bool{
	uuid.MustParse(domain.InternalTransferOutCategoryID): true,
	uuid.MustParse(domain.InternalTransferInCategoryID):  true,
}

func buildExpenseByCategory(paid domain.MovementList) []domain.CategoryExpensePoint {
	totalByCategory := make(map[uuid.UUID]float64)
	categoryByID := make(map[uuid.UUID]domain.Category)

	for _, m := range paid.GetExpenseMovements() {
		if m.CategoryID == nil ||
			m.TypePayment == domain.TypePaymentInternalTransfer ||
			m.TypePayment == domain.TypePaymentInvoicePayment ||
			internalTransferCategoryIDs[*m.CategoryID] {
			continue
		}
		id := *m.CategoryID
		totalByCategory[id] += m.Amount
		if _, seen := categoryByID[id]; !seen {
			categoryByID[id] = m.Category
		}
	}

	points := make([]domain.CategoryExpensePoint, 0, len(totalByCategory))
	for id, total := range totalByCategory {
		category := categoryByID[id]
		categoryID := id
		points = append(points, domain.CategoryExpensePoint{
			CategoryID: &categoryID,
			Name:       category.Description,
			Color:      category.Color,
			Total:      total,
		})
	}

	sort.Slice(points, func(i, j int) bool {
		absI, absJ := math.Abs(points[i].Total), math.Abs(points[j].Total)
		if absI == absJ {
			return points[i].Name < points[j].Name
		}
		return absI > absJ
	})

	return points
}

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
