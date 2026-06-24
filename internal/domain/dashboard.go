package domain

type (
	// DashboardSummary aggregates financial data for the analytics screen.
	DashboardSummary struct {
		MonthlySeries []MonthlyPoint   `json:"monthly_series"`
		CurrentMonth  BudgetComparison `json:"current_month"`
		KPIs          DashboardKPIs    `json:"kpis"`
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

	// DashboardKPIs summarizes the whole period.
	DashboardKPIs struct {
		TotalIncome       float64 `json:"total_income"`
		TotalExpense      float64 `json:"total_expense"`
		AvgMonthlyIncome  float64 `json:"avg_monthly_income"`
		AvgMonthlyExpense float64 `json:"avg_monthly_expense"`
		PeriodNet         float64 `json:"period_net"`
		SavingsRate       float64 `json:"savings_rate"`
	}
)
