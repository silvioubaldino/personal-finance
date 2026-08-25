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

type DashboardInvoiceUseCase interface {
	FindDetailedInvoicesByPeriod(ctx context.Context, period domain.Period) ([]domain.DetailedInvoice, error)
}

type DashboardUseCase interface {
	CalculateSummary(ctx context.Context, period domain.Period) (domain.DashboardSummary, error)
}

type dashboardUseCase struct {
	movementRepo   DashboardMovementRepository
	estimateRepo   DashboardEstimateRepository
	invoiceUseCase DashboardInvoiceUseCase
}

func NewDashboard(
	movementRepo DashboardMovementRepository,
	estimateRepo DashboardEstimateRepository,
	invoiceUseCase DashboardInvoiceUseCase,
) DashboardUseCase {
	return dashboardUseCase{
		movementRepo:   movementRepo,
		estimateRepo:   estimateRepo,
		invoiceUseCase: invoiceUseCase,
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

	invoices, err := uc.invoiceUseCase.FindDetailedInvoicesByPeriod(ctx, period)
	if err != nil {
		return domain.DashboardSummary{}, fmt.Errorf("error finding invoices: %w", err)
	}

	realized := buildRealizedEntries(movements, invoices)
	money := realized.forMoney()

	monthlySeries := buildMonthlySeries(period, money)

	currentMonth, err := uc.buildCurrentMonth(ctx, period, money)
	if err != nil {
		return domain.DashboardSummary{}, err
	}

	return domain.DashboardSummary{
		MonthlySeries:              monthlySeries,
		CurrentMonth:               currentMonth,
		CreditCardInvoices:         buildCreditCardInvoices(period, invoices),
		ExpenseWeekdayDistribution: buildExpenseWeekdayDistribution(realized),
		ExpenseByCategory:          buildExpenseByCategory(money),
		KPIs:                       buildKPIs(monthlySeries),
	}, nil
}

type monthKey struct {
	month int
	year  int
}

// realizedEntry é um Movement do recorte canônico junto com o mês em que ele conta.
// Movement avulso conta no mês da própria data; item de Invoice conta no mês do `due_date`
// da fatura que o recebe — a mesma convenção do gráfico de cartões (AYD-003, decisão #8) e
// do /v2/estimate/summary, que seleciona as faturas do mês por `due_date`. É o que mantém
// `sum(monthly_series) == kpis` e `current_month.budget.realized == realized_paid`.
type realizedEntry struct {
	movement domain.Movement
	month    monthKey
}

type realizedEntries []realizedEntry

// forMoney devolve a base **única** dos agregados de dinheiro: recorte canônico, pagas
// (decisão #2) e com `Category` — sem ela não há como classificar receita×despesa nem
// agrupar por categoria, e é o mesmo corte que `aggregateRealized` usa no summary de
// planejamentos.
//
// Todo bloco de dinheiro parte desta lista, sem filtro extra. É isso que sustenta os
// invariantes do contrato:
//
//	sum(expense_by_category[].total) == kpis.total_expense == sum(monthly_series[].expense)
//	sum(monthly_series[].income)     == kpis.total_income
//	current_month.budget.*.realized  == a fatia do mês de `to` dos mesmos números
func (entries realizedEntries) forMoney() realizedEntries {
	money := make(realizedEntries, 0, len(entries))
	for _, entry := range entries {
		if entry.movement.IsPaid && entry.movement.CategoryID != nil {
			money = append(money, entry)
		}
	}
	return money
}

func (entries realizedEntries) movements() domain.MovementList {
	movements := make(domain.MovementList, 0, len(entries))
	for _, entry := range entries {
		movements = append(movements, entry.movement)
	}
	return movements
}

func (entries realizedEntries) inMonth(key monthKey) realizedEntries {
	filtered := make(realizedEntries, 0, len(entries))
	for _, entry := range entries {
		if entry.month == key {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

func buildRealizedEntries(movements domain.MovementList, invoices []domain.DetailedInvoice) realizedEntries {
	entries := make(realizedEntries, 0, len(movements))

	for _, movement := range movements {
		if !isCanonicalRealized(movement) || movement.Date == nil {
			continue
		}
		entries = append(entries, realizedEntry{movement: movement, month: keyFromTime(*movement.Date)})
	}

	for _, invoice := range invoices {
		month := keyFromTime(invoice.DueDate)
		for _, movement := range invoice.Movements {
			entries = append(entries, realizedEntry{movement: movement, month: month})
		}
	}

	return entries
}

// isIncomeMovement classifica pela flag is_income da Category, nunca pelo sinal
// (AYD-005@context): um estorno em categoria de despesa reduz a despesa, não vira receita.
// Só é chamada sobre entradas de forMoney, que já garantem a Category.
func isIncomeMovement(movement domain.Movement) bool {
	return movement.Category.IsIncome
}

func buildMonthlySeries(period domain.Period, money realizedEntries) []domain.MonthlyPoint {
	incomeByMonth := make(map[monthKey]float64)
	expenseByMonth := make(map[monthKey]float64)

	for _, entry := range money {
		if isIncomeMovement(entry.movement) {
			incomeByMonth[entry.month] += entry.movement.Amount
			continue
		}
		expenseByMonth[entry.month] += entry.movement.Amount
	}

	var series []domain.MonthlyPoint
	for _, k := range monthsOf(period) {
		income := incomeByMonth[k]
		expense := expenseByMonth[k]
		series = append(series, domain.MonthlyPoint{
			Month:   k.month,
			Year:    k.year,
			Income:  income,
			Expense: expense,
			Net:     income + expense,
		})
	}

	return series
}

func (uc dashboardUseCase) buildCurrentMonth(
	ctx context.Context,
	period domain.Period,
	paid realizedEntries,
) (domain.BudgetComparison, error) {
	month := int(period.To.Month())
	year := period.To.Year()

	estimates, err := uc.estimateRepo.FindCategoriesByMonth(ctx, month, year)
	if err != nil {
		return domain.BudgetComparison{}, fmt.Errorf("error finding estimates: %w", err)
	}

	// Mesmo cálculo de /v2/estimate/summary: `realized` daqui é o `realized_paid` de lá
	// (AYD-005@context), então as duas telas mostram o mesmo número.
	categoryAggs, _ := aggregateRealized(paid.inMonth(monthKey{month: month, year: year}).movements())
	totals := buildTotals(estimates, categoryAggs)

	return domain.BudgetComparison{
		Month: month,
		Year:  year,
		Budget: domain.DashboardBudget{
			Income: domain.BudgetLine{
				Budgeted: totals.Income.Budgeted,
				Realized: totals.Income.RealizedPaid,
			},
			Expense: domain.BudgetLine{
				Budgeted: totals.Expense.Budgeted,
				Realized: totals.Expense.RealizedPaid,
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

func buildCreditCardInvoices(period domain.Period, invoices []domain.DetailedInvoice) domain.CreditCardInvoiceSummary {
	amountByMonthAndCard := make(map[monthKey]map[uuid.UUID]float64)
	cardsByID := make(map[uuid.UUID]domain.CreditCardRef)

	for _, detailed := range invoices {
		invoice := detailed.Invoice
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
	for _, k := range monthsOf(period) {
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

// buildExpenseWeekdayDistribution mede comportamento de compra (AYD-003, decisão #7): conta
// pagas e pendentes, pelo dia da própria compra — inclusive as do cartão, que só ficam
// `is_paid` quando a Invoice é paga. O invoice_remainder fica fora: é saldo empurrado para a
// fatura seguinte, não uma compra, e sua data é o vencimento anterior + 1 dia.
func buildExpenseWeekdayDistribution(realized realizedEntries) []domain.ExpenseWeekdayPoint {
	counts := make([]int, 7)
	total := 0

	for _, entry := range realized {
		movement := entry.movement
		if movement.Date == nil ||
			isIncomeMovement(movement) ||
			movement.TypePayment == domain.TypePaymentInvoiceRemainder {
			continue
		}
		counts[int(movement.Date.Weekday())]++
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

func buildExpenseByCategory(money realizedEntries) []domain.CategoryExpensePoint {
	totalByCategory := make(map[uuid.UUID]float64)
	categoryByID := make(map[uuid.UUID]domain.Category)

	for _, entry := range money {
		movement := entry.movement
		if isIncomeMovement(movement) {
			continue
		}
		id := *movement.CategoryID
		totalByCategory[id] += movement.Amount
		if _, seen := categoryByID[id]; !seen {
			categoryByID[id] = movement.Category
		}
	}

	points := make([]domain.CategoryExpensePoint, 0, len(totalByCategory))
	for id, total := range totalByCategory {
		// Só o total exatamente zero sai: não move a soma e não tem barra. Categoria que
		// fechou o período **positiva** (estorno maior que o gasto) fica, com o total
		// positivo — tirá-la quebraria sum(expense_by_category) == kpis.total_expense,
		// que é invariante do contrato.
		if total == 0 {
			continue
		}
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

// monthsOf devolve um monthKey por mês do span, para os gráficos manterem eixo contínuo.
func monthsOf(period domain.Period) []monthKey {
	var months []monthKey
	cursor := time.Date(period.From.Year(), period.From.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(period.To.Year(), period.To.Month(), 1, 0, 0, 0, 0, time.UTC)
	for !cursor.After(end) {
		months = append(months, monthKey{month: int(cursor.Month()), year: cursor.Year()})
		cursor = cursor.AddDate(0, 1, 0)
	}
	return months
}

func keyFromTime(t time.Time) monthKey {
	return monthKey{month: int(t.Month()), year: t.Year()}
}
