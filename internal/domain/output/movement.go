package output

import (
	"time"

	"personal-finance/internal/domain"

	"github.com/google/uuid"
)

type MovementOutput struct {
	ID           *uuid.UUID        `json:"id,omitempty"`
	Description  string            `json:"description,omitempty"`
	Amount       float64           `json:"amount"`
	Date         *time.Time        `json:"date,omitempty"`
	IsPaid       bool              `json:"is_paid"`
	IsRecurrent  bool              `json:"is_recurrent"`
	RecurrentID  *uuid.UUID        `json:"recurrent_id"`
	InvoiceID    *uuid.UUID        `json:"invoice_id,omitempty"`
	CreditCardID *uuid.UUID        `json:"credit_card_id,omitempty"`
	Wallet       WalletOutput      `json:"wallet,omitempty"`
	TypePayment  string            `json:"type_payment,omitempty"`
	Category     CategoryOutput    `json:"category,omitempty"`
	SubCategory  SubCategoryOutput `json:"sub_category,omitempty"`
	DateUpdate   *time.Time        `json:"date_update,omitempty"`
}

type MovementListOutput []MovementOutput

func ToMovementOutput(input domain.Movement) *MovementOutput {
	output := &MovementOutput{
		ID:           input.ID,
		Description:  input.Description,
		Amount:       input.Amount,
		Date:         input.Date,
		IsPaid:       input.IsPaid,
		IsRecurrent:  input.IsRecurrent,
		RecurrentID:  input.RecurrentID,
		InvoiceID:    input.InvoiceID,
		CreditCardID: input.CreditCardID,
		Wallet:       ToWalletOutput(input.Wallet),
		TypePayment:  string(input.TypePayment),
		Category:     ToCategoryOutput(input.Category),
		SubCategory:  ToSubCategoryOutput(input.SubCategory),
		DateUpdate:   &input.DateUpdate,
	}
	return output
}
