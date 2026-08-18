package domain

import "github.com/google/uuid"

type (
	// DashboardSummary aggregates financial data for the analytics screen.
	DashboardSummary struct {
		MonthlySeries              []MonthlyPoint           `json:"monthly_series"`
		CurrentMonth               BudgetComparison         `json:"current_month"`
		CreditCardInvoices         CreditCardInvoiceSummary `json:"credit_card_invoices"`
		ExpenseWeekdayDistribution []ExpenseWeekdayPoint    `json:"expense_weekday_distribution"`
		ExpenseByCategory          []CategoryExpensePoint   `json:"expense_by_category"`
		KPIs                       DashboardKPIs            `json:"kpis"`
	}

	// MonthlyPoint is one calendar month entry in the chart series.
	MonthlyPoint struct {
		Month   int     `json:"month"`
		Year    int     `json:"year"`
		Income  float64 `json:"income"`
		Expense float64 `json:"expense"`
		Net     float64 `json:"net"`
	}

	// BudgetComparison holds budgeted vs realized values for a single month.
	BudgetComparison struct {
		Month  int             `json:"month"`
		Year   int             `json:"year"`
		Budget DashboardBudget `json:"budget"`
	}

	// DashboardBudget groups income and expense budget lines.
	DashboardBudget struct {
		Income  BudgetLine `json:"income"`
		Expense BudgetLine `json:"expense"`
	}

	// BudgetLine compares the budgeted value against the realized one.
	BudgetLine struct {
		Budgeted float64 `json:"budgeted"`
		Realized float64 `json:"realized"`
	}

	// CreditCardInvoiceSummary holds the monthly invoice totals stacked by credit card.
	// Cards is the canonical legend/stacking order shared by every entry in Series.
	CreditCardInvoiceSummary struct {
		Cards  []CreditCardRef          `json:"cards"`
		Series []CreditCardInvoicePoint `json:"series"`
	}

	// CreditCardRef identifies a credit card in the invoice chart legend.
	// Color is the card's own color (#RRGGBB) and may be empty when the user
	// never picked one — the client falls back to its palette.
	CreditCardRef struct {
		CreditCardID *uuid.UUID `json:"credit_card_id"`
		Name         string     `json:"name"`
		Color        string     `json:"color"`
	}

	// CreditCardInvoicePoint is one calendar month of invoice totals.
	// ByCard always carries every card in Cards, using 0 where there was no
	// invoice, so the stacking order stays stable across months.
	CreditCardInvoicePoint struct {
		Month  int                      `json:"month"`
		Year   int                      `json:"year"`
		Total  float64                  `json:"total"`
		ByCard []CreditCardInvoiceSlice `json:"by_card"`
	}

	// CreditCardInvoiceSlice is one card's share of a month's invoice total.
	CreditCardInvoiceSlice struct {
		CreditCardID *uuid.UUID `json:"credit_card_id"`
		Amount       float64    `json:"amount"`
	}

	// ExpenseWeekdayPoint is the share of expense movements made on a weekday.
	// Weekday follows time.Weekday: 0 = Sunday ... 6 = Saturday.
	ExpenseWeekdayPoint struct {
		Weekday    int     `json:"weekday"`
		Count      int     `json:"count"`
		Percentage float64 `json:"percentage"`
	}

	// CategoryExpensePoint is the total paid expense in one category for the
	// whole period. Sorted by the caller from largest to smallest total.
	// Color is the category's own color and may be empty — the client falls
	// back to its palette, same as CreditCardRef.
	CategoryExpensePoint struct {
		CategoryID *uuid.UUID `json:"category_id"`
		Name       string     `json:"name"`
		Color      string     `json:"color"`
		Total      float64    `json:"total"`
	}

	// DashboardKPIs summarizes the whole period.
	DashboardKPIs struct {
		TotalIncome  float64 `json:"total_income"`
		TotalExpense float64 `json:"total_expense"`
	}
)
