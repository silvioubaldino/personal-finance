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
	InitialBalance float64    `json:"initial_balance,omitempty"`
	InitialDate    *time.Time `json:"initial_date,omitempty"`
}

func ToWalletOutput(input domain.Wallet) WalletOutput {
	var truncated *time.Time
	if !input.InitialDate.IsZero() {
		t := input.InitialDate.Truncate(time.Second)
		truncated = &t
	}
	return WalletOutput{
		ID:             input.ID,
		Description:    input.Description,
		Balance:        input.Balance,
		InitialBalance: input.InitialBalance,
		InitialDate:    truncated,
	}
}

type WalletOutputDTO struct {
	ID          *uuid.UUID `json:"id,omitempty"`
	Description string     `json:"description,omitempty"`
}

func ToWalletOutputDTO(input domain.Wallet) WalletOutputDTO {
	return WalletOutputDTO{
		ID:          input.ID,
		Description: input.Description,
	}
}
