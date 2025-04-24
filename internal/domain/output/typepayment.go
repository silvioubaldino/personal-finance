package output

import (
	"personal-finance/internal/domain"
)

type TypePaymentOutput struct {
	ID          int    `json:"id,omitempty"`
	Description string `json:"description,omitempty"`
}

func ToTypePaymentOutput(input domain.TypePayment) TypePaymentOutput {
	return TypePaymentOutput{
		ID:          input.ID,
		Description: input.Description,
	}
}
