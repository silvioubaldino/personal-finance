package output

import (
	"personal-finance/internal/domain"
)

type BalanceOutput struct {
	Expense       float64 `json:"expense"`
	Income        float64 `json:"income"`
	PeriodBalance float64 `json:"period_balance"`
}

func ToBalanceOutput(input domain.Balance) BalanceOutput {
	return BalanceOutput{
		Expense:       input.Expense,
		Income:        input.Income,
		PeriodBalance: input.PeriodBalance,
	}
}
