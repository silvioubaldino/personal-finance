package output

import (
	"time"

	"personal-finance/internal/domain"

	"github.com/google/uuid"
)

type CreditCardOutput struct {
	ID            *uuid.UUID   `json:"id,omitempty"`
	Name          string       `json:"name"`
	CreditLimit   float64      `json:"credit_limit"`
	ClosingDay    int          `json:"closing_day"`
	DueDay        int          `json:"due_day"`
	DefaultWallet WalletOutput `json:"default_wallet,omitempty"`
	DateUpdate    time.Time    `json:"date_update"`
}

func ToCreditCardOutput(input domain.CreditCard) CreditCardOutput {
	return CreditCardOutput{
		ID:            input.ID,
		Name:          input.Name,
		CreditLimit:   input.CreditLimit,
		ClosingDay:    input.ClosingDay,
		DueDay:        input.DueDay,
		DefaultWallet: ToWalletOutput(input.DefaultWallet),
		DateUpdate:    input.DateUpdate,
	}
}

type CreditCardOutputDTO struct {
	ID   *uuid.UUID `json:"id,omitempty"`
	Name string     `json:"name"`
}

func ToCreditCardOutputDTO(input domain.CreditCard) CreditCardOutputDTO {
	return CreditCardOutputDTO{
		ID:   input.ID,
		Name: input.Name,
	}
}
