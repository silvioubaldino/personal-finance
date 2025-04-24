package output

import (
	"time"

	"personal-finance/internal/domain"

	"github.com/google/uuid"
)

type WalletOutput struct {
	ID             *uuid.UUID `json:"id,omitempty"`
	Description    string     `json:"description,omitempty"`
	Balance        float64    `json:"balance"`
	InitialBalance float64    `json:"initial_balance"`
	InitialDate    time.Time  `json:"initial_date"`
}

func ToWalletOutput(input domain.Wallet) WalletOutput {
	return WalletOutput{
		ID:             input.ID,
		Description:    input.Description,
		Balance:        input.Balance,
		InitialBalance: input.InitialBalance,
		InitialDate:    input.InitialDate.Truncate(time.Second),
	}
}
