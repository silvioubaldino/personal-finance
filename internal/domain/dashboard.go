package domain

import "github.com/google/uuid"

type (
	DashboardSummary struct {
		MonthlySeries            []MonthlyPoint           `json:"monthly_series"`
		CurrentMonth             BudgetComparison         `json:"current_month"`
		CreditCardInvoices       CreditCardInvoiceSummary `json:"credit_card_invoices"`
		ExpenseDailyDistribution []ExpenseDailyPoint      `json:"expense_daily_distribution"`
		// ExpenseWeekdayDistribution é deprecado (AYD-003@context, decisão #15): substituído
		// por ExpenseDailyDistribution, do qual é a marginal por coluna. Mantido no payload
		// por um ciclo de release porque api e mobile já estão em produção lendo este campo;
		// sai quando os clientes publicados não lerem mais (ver AYD-003@context § "Removido
		// do payload").
		ExpenseWeekdayDistribution []ExpenseWeekdayPoint  `json:"expense_weekday_distribution"`
		ExpenseByCategory          []CategoryExpensePoint `json:"expense_by_category"`
		KPIs                       DashboardKPIs          `json:"kpis"`
	}

	MonthlyPoint struct {
		Month   int     `json:"month"`
		Year    int     `json:"year"`
		Income  float64 `json:"income"`
		Expense float64 `json:"expense"`
		Net     float64 `json:"net"`
	}

	BudgetComparison struct {
		Month  int             `json:"month"`
		Year   int             `json:"year"`
		Budget DashboardBudget `json:"budget"`
	}

	DashboardBudget struct {
		Income  BudgetLine `json:"income"`
		Expense BudgetLine `json:"expense"`
	}

	BudgetLine struct {
		Budgeted float64 `json:"budgeted"`
		Realized float64 `json:"realized"`
	}

	CreditCardInvoiceSummary struct {
		Cards  []CreditCardRef          `json:"cards"`
		Series []CreditCardInvoicePoint `json:"series"`
	}

	CreditCardRef struct {
		CreditCardID *uuid.UUID `json:"credit_card_id"`
		Name         string     `json:"name"`
		Color        string     `json:"color"`
	}

	CreditCardInvoicePoint struct {
		Month  int                      `json:"month"`
		Year   int                      `json:"year"`
		Total  float64                  `json:"total"`
		ByCard []CreditCardInvoiceSlice `json:"by_card"`
	}

	CreditCardInvoiceSlice struct {
		CreditCardID *uuid.UUID `json:"credit_card_id"`
		Amount       float64    `json:"amount"`
	}

	ExpenseWeekdayPoint struct {
		Weekday    int     `json:"weekday"`
		Count      int     `json:"count"`
		Percentage float64 `json:"percentage"`
	}

	// ExpenseDailyPoint é uma célula do mapa de calor diário (AYD-003@context, viz #5): uma
	// entrada por dia do span, zero-preenchida nos dias sem gasto. Date serializa no formato
	// 2006-01-02.
	ExpenseDailyPoint struct {
		Date  string  `json:"date"`
		Count int     `json:"count"`
		Total float64 `json:"total"`
	}

	CategoryExpensePoint struct {
		CategoryID *uuid.UUID `json:"category_id"`
		Name       string     `json:"name"`
		Color      string     `json:"color"`
		Total      float64    `json:"total"`
	}

	DashboardKPIs struct {
		TotalIncome  float64 `json:"total_income"`
		TotalExpense float64 `json:"total_expense"`
	}
)
