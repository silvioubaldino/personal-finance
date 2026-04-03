package domain

// AgentWalletItem is a minimal wallet representation for the agent.
type AgentWalletItem struct {
	Name    string  `json:"name"`
	Balance float64 `json:"balance"`
}

// AgentFinancialOverview is the response for get_financial_overview tool.
type AgentFinancialOverview struct {
	Period   string            `json:"period"`
	Income   float64           `json:"income"`
	Expenses float64           `json:"expenses"`
	Net      float64           `json:"net"`
	Wallets  []AgentWalletItem `json:"wallets"`
}

// AgentCategoryItem is a minimal category breakdown item for the agent.
type AgentCategoryItem struct {
	Name     string  `json:"name"`
	Amount   float64 `json:"amount"`
	Pct      float64 `json:"pct"`
	IsIncome bool    `json:"is_income"`
}

// AgentSpendingBreakdown is the response for get_spending_breakdown tool.
type AgentSpendingBreakdown struct {
	Period        string              `json:"period"`
	TotalIncome   float64             `json:"total_income"`
	TotalExpenses float64             `json:"total_expenses"`
	Categories    []AgentCategoryItem `json:"categories"`
}

// AgentCreditCardItem is a minimal credit card representation for the agent.
type AgentCreditCardItem struct {
	Name              string  `json:"name"`
	Limit             float64 `json:"limit"`
	Available         float64 `json:"available"`
	NextDueDate       string  `json:"next_due_date"`
	NextDueAmount     float64 `json:"next_due_amount"`
	OpenInvoicesCount int     `json:"open_invoices_count"`
}

// AgentCreditCardsSummary is the response for get_credit_cards tool.
type AgentCreditCardsSummary struct {
	Cards []AgentCreditCardItem `json:"cards"`
}

// AgentMovementItem is a minimal movement representation for the agent.
type AgentMovementItem struct {
	Date        string  `json:"date"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Category    string  `json:"category"`
	Wallet      string  `json:"wallet"`
}

// AgentMovementsList is the response for get_movements tool.
type AgentMovementsList struct {
	Count     int                 `json:"count"`
	Total     float64             `json:"total"`
	Movements []AgentMovementItem `json:"movements"`
}

// AgentRecurringItem is a minimal recurring expense representation for the agent.
type AgentRecurringItem struct {
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Category    string  `json:"category"`
	Day         int     `json:"day"`
}

// AgentRecurringSummary is the response for get_recurring_expenses tool.
type AgentRecurringSummary struct {
	TotalMonthly float64              `json:"total_monthly"`
	Items        []AgentRecurringItem `json:"items"`
}

// AgentBudgetItem is a minimal budget vs actual item for the agent.
type AgentBudgetItem struct {
	Name        string  `json:"name"`
	Estimated   float64 `json:"estimated"`
	Actual      float64 `json:"actual"`
	Variance    float64 `json:"variance"`
	VariancePct float64 `json:"variance_pct"`
}

// AgentBudgetStatus is the response for get_budget_status tool.
type AgentBudgetStatus struct {
	Period     string            `json:"period"`
	Categories []AgentBudgetItem `json:"categories"`
}
