package domain

import "github.com/google/uuid"

type (
	DashboardSummary struct {
		MonthlySeries              []MonthlyPoint           `json:"monthly_series"`
		CurrentMonth               BudgetComparison         `json:"current_month"`
		CreditCardInvoices         CreditCardInvoiceSummary `json:"credit_card_invoices"`
		ExpenseWeekdayDistribution []ExpenseWeekdayPoint    `json:"expense_weekday_distribution"`
		ExpenseByCategory          []CategoryExpensePoint   `json:"expense_by_category"`
		KPIs                       DashboardKPIs            `json:"kpis"`
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
