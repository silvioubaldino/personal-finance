package repository

import (
	"context"
	"fmt"
	"math"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/plataform/authentication"

	"gorm.io/gorm"
)

const defaultMovementsLimit = 50

// AgentFinancialRepository provides optimized read-only queries for the AI agent tools.
// It avoids N+1 queries and returns only the minimal fields needed by the LLM.
type AgentFinancialRepository struct {
	db *gorm.DB
}

func NewAgentFinancialRepository(db *gorm.DB) *AgentFinancialRepository {
	return &AgentFinancialRepository{db: db}
}

// --- Internal helpers ---

type periodBounds struct {
	start time.Time
	end   time.Time
	label string
}

func buildPeriodBounds(month, year int) periodBounds {
	now := time.Now()
	if month == 0 {
		month = int(now.Month())
	}
	if year == 0 {
		year = now.Year()
	}
	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)
	return periodBounds{
		start: start,
		end:   end,
		label: fmt.Sprintf("%d-%02d", year, month),
	}
}

// --- GetFinancialOverview ---

type overviewRow struct {
	Kind   string  `gorm:"column:kind"`
	Amount float64 `gorm:"column:amount"`
}

type walletRow struct {
	Name    string  `gorm:"column:name"`
	Balance float64 `gorm:"column:balance"`
}

// GetFinancialOverview returns income, expenses, net and wallet balances for the given period.
func (r *AgentFinancialRepository) GetFinancialOverview(ctx context.Context, month, year int) (domain.AgentFinancialOverview, error) {
	userID := authentication.UserIDFromContext(ctx)
	p := buildPeriodBounds(month, year)

	var rows []overviewRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			CASE WHEN amount > 0 THEN 'income' ELSE 'expense' END AS kind,
			SUM(amount) AS amount
		FROM movements
		WHERE user_id = ?
		  AND is_paid = true
		  AND date >= ?
		  AND date < ?
		  AND type_payment NOT IN ('invoice_payment', 'internal_transfer')
		GROUP BY kind
	`, userID, p.start, p.end).Scan(&rows).Error
	if err != nil {
		return domain.AgentFinancialOverview{}, fmt.Errorf("overview income/expense query: %w", err)
	}

	var income, expenses float64
	for _, row := range rows {
		if row.Kind == "income" {
			income = row.Amount
		} else {
			expenses = math.Abs(row.Amount)
		}
	}

	var walletRows []walletRow
	err = r.db.WithContext(ctx).Raw(`
		SELECT description AS name, balance
		FROM wallets
		WHERE user_id = ?
		ORDER BY balance DESC
	`, userID).Scan(&walletRows).Error
	if err != nil {
		return domain.AgentFinancialOverview{}, fmt.Errorf("overview wallets query: %w", err)
	}

	wallets := make([]domain.AgentWalletItem, 0, len(walletRows))
	for _, w := range walletRows {
		wallets = append(wallets, domain.AgentWalletItem{Name: w.Name, Balance: w.Balance})
	}

	return domain.AgentFinancialOverview{
		Period:   p.label,
		Income:   income,
		Expenses: expenses,
		Net:      income - expenses,
		Wallets:  wallets,
	}, nil
}

// --- GetSpendingBreakdown ---

type spendingRow struct {
	CategoryName string  `gorm:"column:category_name"`
	IsIncome     bool    `gorm:"column:is_income"`
	Amount       float64 `gorm:"column:amount"`
}

// GetSpendingBreakdown returns spending and income grouped by category for the given period.
func (r *AgentFinancialRepository) GetSpendingBreakdown(ctx context.Context, month, year int) (domain.AgentSpendingBreakdown, error) {
	userID := authentication.UserIDFromContext(ctx)
	p := buildPeriodBounds(month, year)

	var rows []spendingRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			COALESCE(c.description, 'Sem categoria') AS category_name,
			COALESCE(c.is_income, false) AS is_income,
			SUM(m.amount) AS amount
		FROM movements m
		LEFT JOIN categories c ON c.id = m.category_id
		WHERE m.user_id = ?
		  AND m.is_paid = true
		  AND m.date >= ?
		  AND m.date < ?
		  AND m.type_payment NOT IN ('invoice_payment', 'internal_transfer')
		GROUP BY c.description, c.is_income
		ORDER BY ABS(SUM(m.amount)) DESC
	`, userID, p.start, p.end).Scan(&rows).Error
	if err != nil {
		return domain.AgentSpendingBreakdown{}, fmt.Errorf("spending breakdown query: %w", err)
	}

	var totalIncome, totalExpenses float64
	for _, row := range rows {
		if row.IsIncome {
			totalIncome += row.Amount
		} else {
			totalExpenses += math.Abs(row.Amount)
		}
	}

	categories := make([]domain.AgentCategoryItem, 0, len(rows))
	for _, row := range rows {
		abs := math.Abs(row.Amount)
		var pct float64
		if row.IsIncome && totalIncome > 0 {
			pct = math.Round((abs/totalIncome)*1000) / 10
		} else if !row.IsIncome && totalExpenses > 0 {
			pct = math.Round((abs/totalExpenses)*1000) / 10
		}
		categories = append(categories, domain.AgentCategoryItem{
			Name:     row.CategoryName,
			Amount:   abs,
			Pct:      pct,
			IsIncome: row.IsIncome,
		})
	}

	return domain.AgentSpendingBreakdown{
		Period:        p.label,
		TotalIncome:   totalIncome,
		TotalExpenses: totalExpenses,
		Categories:    categories,
	}, nil
}

// --- GetCreditCardsSummary ---

type creditCardRow struct {
	Name              string  `gorm:"column:name"`
	CreditLimit       float64 `gorm:"column:credit_limit"`
	NextDueDate       *time.Time `gorm:"column:next_due_date"`
	NextDueAmount     float64 `gorm:"column:next_due_amount"`
	OpenInvoicesCount int     `gorm:"column:open_invoices_count"`
}

// GetCreditCardsSummary returns all credit cards with limit, available and next invoice info.
func (r *AgentFinancialRepository) GetCreditCardsSummary(ctx context.Context) (domain.AgentCreditCardsSummary, error) {
	userID := authentication.UserIDFromContext(ctx)

	var rows []creditCardRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			cc.name,
			cc.credit_limit,
			MIN(i.due_date) FILTER (WHERE i.is_paid = false AND i.amount > 0) AS next_due_date,
			COALESCE(
				(SELECT i2.amount FROM invoices i2
				 WHERE i2.credit_card_id = cc.id AND i2.is_paid = false AND i2.amount > 0
				 ORDER BY i2.due_date ASC LIMIT 1),
				0
			) AS next_due_amount,
			COUNT(i.id) FILTER (WHERE i.is_paid = false AND i.amount > 0) AS open_invoices_count
		FROM credit_cards cc
		LEFT JOIN invoices i ON i.credit_card_id = cc.id AND i.user_id = ?
		WHERE cc.user_id = ?
		GROUP BY cc.id, cc.name, cc.credit_limit
		ORDER BY cc.name
	`, userID, userID).Scan(&rows).Error
	if err != nil {
		return domain.AgentCreditCardsSummary{}, fmt.Errorf("credit cards summary query: %w", err)
	}

	// Fetch total open invoice amounts per card to compute available limit
	type usedRow struct {
		Name   string  `gorm:"column:name"`
		InUse  float64 `gorm:"column:in_use"`
	}
	var usedRows []usedRow
	err = r.db.WithContext(ctx).Raw(`
		SELECT cc.name, COALESCE(SUM(i.amount), 0) AS in_use
		FROM credit_cards cc
		LEFT JOIN invoices i ON i.credit_card_id = cc.id AND i.is_paid = false
		WHERE cc.user_id = ?
		GROUP BY cc.name
	`, userID).Scan(&usedRows).Error
	if err != nil {
		return domain.AgentCreditCardsSummary{}, fmt.Errorf("credit cards used query: %w", err)
	}

	usedByName := make(map[string]float64, len(usedRows))
	for _, u := range usedRows {
		usedByName[u.Name] = u.InUse
	}

	cards := make([]domain.AgentCreditCardItem, 0, len(rows))
	for _, row := range rows {
		nextDueDate := ""
		if row.NextDueDate != nil {
			nextDueDate = row.NextDueDate.Format("2006-01-02")
		}
		available := row.CreditLimit - usedByName[row.Name]
		cards = append(cards, domain.AgentCreditCardItem{
			Name:              row.Name,
			Limit:             row.CreditLimit,
			Available:         available,
			NextDueDate:       nextDueDate,
			NextDueAmount:     row.NextDueAmount,
			OpenInvoicesCount: row.OpenInvoicesCount,
		})
	}

	return domain.AgentCreditCardsSummary{Cards: cards}, nil
}

// --- GetMovements ---

type movementRow struct {
	Date        time.Time `gorm:"column:date"`
	Description string    `gorm:"column:description"`
	Amount      float64   `gorm:"column:amount"`
	Category    string    `gorm:"column:category"`
	Wallet      string    `gorm:"column:wallet"`
}

// GetMovements returns a paginated list of movements for the given period.
func (r *AgentFinancialRepository) GetMovements(ctx context.Context, month, year, limit int) (domain.AgentMovementsList, error) {
	userID := authentication.UserIDFromContext(ctx)
	p := buildPeriodBounds(month, year)

	if limit <= 0 || limit > 100 {
		limit = defaultMovementsLimit
	}

	var rows []movementRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			m.date,
			m.description,
			m.amount,
			COALESCE(c.description, 'Sem categoria') AS category,
			COALESCE(w.description, '') AS wallet
		FROM movements m
		LEFT JOIN categories c ON c.id = m.category_id
		LEFT JOIN wallets w ON w.id = m.wallet_id
		WHERE m.user_id = ?
		  AND m.date >= ?
		  AND m.date < ?
		  AND m.type_payment NOT IN ('invoice_payment', 'internal_transfer')
		ORDER BY m.date DESC
		LIMIT ?
	`, userID, p.start, p.end, limit).Scan(&rows).Error
	if err != nil {
		return domain.AgentMovementsList{}, fmt.Errorf("movements query: %w", err)
	}

	// Count total (without limit) and sum
	type summaryRow struct {
		Count int     `gorm:"column:count"`
		Total float64 `gorm:"column:total"`
	}
	var summary summaryRow
	err = r.db.WithContext(ctx).Raw(`
		SELECT COUNT(*) AS count, COALESCE(SUM(amount), 0) AS total
		FROM movements
		WHERE user_id = ?
		  AND date >= ?
		  AND date < ?
		  AND type_payment NOT IN ('invoice_payment', 'internal_transfer')
	`, userID, p.start, p.end).Scan(&summary).Error
	if err != nil {
		return domain.AgentMovementsList{}, fmt.Errorf("movements summary query: %w", err)
	}

	movements := make([]domain.AgentMovementItem, 0, len(rows))
	for _, row := range rows {
		movements = append(movements, domain.AgentMovementItem{
			Date:        row.Date.Format("2006-01-02"),
			Description: row.Description,
			Amount:      row.Amount,
			Category:    row.Category,
			Wallet:      row.Wallet,
		})
	}

	return domain.AgentMovementsList{
		Count:     summary.Count,
		Total:     summary.Total,
		Movements: movements,
	}, nil
}

// --- GetRecurringSummary ---

type recurringRow struct {
	Description  string    `gorm:"column:description"`
	Amount       float64   `gorm:"column:amount"`
	CategoryName string    `gorm:"column:category_name"`
	InitialDate  time.Time `gorm:"column:initial_date"`
}

// GetRecurringSummary returns all active recurring expenses/incomes with total monthly impact.
func (r *AgentFinancialRepository) GetRecurringSummary(ctx context.Context) (domain.AgentRecurringSummary, error) {
	userID := authentication.UserIDFromContext(ctx)
	now := time.Now()

	var rows []recurringRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			rm.description,
			rm.amount,
			COALESCE(c.description, 'Sem categoria') AS category_name,
			rm.initial_date
		FROM recurrent_movements rm
		LEFT JOIN categories c ON c.id = rm.category_id
		WHERE rm.user_id = ?
		  AND rm.initial_date <= ?
		  AND (rm.end_date IS NULL OR rm.end_date >= ?)
		ORDER BY rm.amount ASC
	`, userID, now, now).Scan(&rows).Error
	if err != nil {
		return domain.AgentRecurringSummary{}, fmt.Errorf("recurring summary query: %w", err)
	}

	var total float64
	items := make([]domain.AgentRecurringItem, 0, len(rows))
	for _, row := range rows {
		total += row.Amount
		day := 1
		if !row.InitialDate.IsZero() {
			day = row.InitialDate.Day()
		}
		items = append(items, domain.AgentRecurringItem{
			Description: row.Description,
			Amount:      row.Amount,
			Category:    row.CategoryName,
			Day:         day,
		})
	}

	return domain.AgentRecurringSummary{
		TotalMonthly: total,
		Items:        items,
	}, nil
}

// --- GetBudgetStatus ---

type budgetRow struct {
	CategoryName string  `gorm:"column:category_name"`
	Estimated    float64 `gorm:"column:estimated"`
	Actual       float64 `gorm:"column:actual"`
}

// GetBudgetStatus compares estimated (budget) vs actual spending for the given period.
func (r *AgentFinancialRepository) GetBudgetStatus(ctx context.Context, month, year int) (domain.AgentBudgetStatus, error) {
	userID := authentication.UserIDFromContext(ctx)
	p := buildPeriodBounds(month, year)
	monthNum := p.start.Month()
	yearNum := p.start.Year()

	var rows []budgetRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			ec.category_name,
			ec.amount AS estimated,
			COALESCE(SUM(m.amount), 0) AS actual
		FROM estimate_categories ec
		LEFT JOIN movements m ON m.category_id = ec.category_id
			AND m.user_id = ?
			AND m.is_paid = true
			AND m.date >= ?
			AND m.date < ?
			AND m.type_payment NOT IN ('invoice_payment', 'internal_transfer')
		WHERE ec.user_id = ?
		  AND ec.month = ?
		  AND ec.year = ?
		GROUP BY ec.category_name, ec.amount
		ORDER BY ABS(ec.amount) DESC
	`, userID, p.start, p.end, userID, int(monthNum), yearNum).Scan(&rows).Error
	if err != nil {
		return domain.AgentBudgetStatus{}, fmt.Errorf("budget status query: %w", err)
	}

	categories := make([]domain.AgentBudgetItem, 0, len(rows))
	for _, row := range rows {
		variance := row.Actual - row.Estimated
		var variancePct float64
		if row.Estimated != 0 {
			variancePct = math.Round((variance/math.Abs(row.Estimated))*1000) / 10
		}
		categories = append(categories, domain.AgentBudgetItem{
			Name:        row.CategoryName,
			Estimated:   row.Estimated,
			Actual:      row.Actual,
			Variance:    variance,
			VariancePct: variancePct,
		})
	}

	return domain.AgentBudgetStatus{
		Period:     p.label,
		Categories: categories,
	}, nil
}
