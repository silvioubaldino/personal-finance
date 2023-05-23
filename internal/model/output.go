package model

import (
	"time"

	"github.com/google/uuid"
)

type (
	WalletOutput struct {
		ID          int     `json:"id,omitempty" gorm:"primaryKey"`
		Description string  `json:"description,omitempty"`
		Balance     float64 `json:"balance"`
	}

	TypePaymentOutput struct {
		ID          int    `json:"id,omitempty" gorm:"primaryKey"`
		Description string `json:"description,omitempty"`
	}

	CategoryOutput struct {
		ID          int    `json:"id,omitempty" gorm:"primaryKey"`
		Description string `json:"description,omitempty"`
	}

	TransactionStatusOutput struct {
		ID          int    `json:"id,omitempty" gorm:"primaryKey"`
		Description string `json:"description,omitempty"`
	}

	MovementOutput struct {
		ID            *uuid.UUID        `json:"id,omitempty" gorm:"primaryKey"`
		Description   string            `json:"description,omitempty"`
		Amount        float64           `json:"amount"`
		Date          *time.Time        `json:"date,omitempty"`
		TransactionID *uuid.UUID        `json:"parent_transaction_id"`
		Wallet        WalletOutput      `json:"wallets,omitempty"`
		TypePayment   TypePaymentOutput `json:"type_payments,omitempty"`
		Category      CategoryOutput    `json:"categories,omitempty"`
		DateUpdate    *time.Time        `json:"date_update,omitempty"`
	}

	MovementListOutput []MovementOutput

	TransactionOutput struct {
		TransactionID *uuid.UUID           `json:"transaction_id"`
		Estimate      *MovementOutput      `json:"estimate,omitempty"`
		Consolidation *ConsolidationOutput `json:"consolidation,omitempty"`
		DoneList      MovementListOutput   `json:"done_list"`
	}

	ConsolidationOutput struct {
		Estimated float64 `json:"estimated"`
		Realized  float64 `json:"realized"`
		Remaining float64 `json:"remaining"`
	}
)

func ToTransactionOutput(input Transaction) TransactionOutput {
	output := TransactionOutput{
		TransactionID: input.TransactionID,
		Estimate:      ToMovementOutput(input.Estimate),
		Consolidation: toConsolidationOutput(*input.Consolidation),
		DoneList:      toTransactionListOutput(input.DoneList),
	}
	return output
}

func ToMovementOutput(input *Movement) *MovementOutput {
	output := &MovementOutput{
		ID:            input.ID,
		Description:   input.Description,
		Amount:        input.Amount,
		Date:          input.Date,
		TransactionID: input.TransactionID,
		Wallet:        ToWalletOutput(input.Wallet),
		TypePayment:   ToTypePaymentOutput(input.TypePayment),
		Category:      ToCategoryOutput(input.Category),
		DateUpdate:    &input.DateUpdate,
	}
	return output
}

func toTransactionListOutput(input MovementList) MovementListOutput {
	output := make(MovementListOutput, len(input))
	for i, trx := range input {
		output[i] = *ToMovementOutput(&trx)
	}
	return output
}

func ToWalletOutput(input Wallet) WalletOutput {
	return WalletOutput{
		ID:          input.ID,
		Description: input.Description,
		Balance:     input.Balance,
	}
}

func ToTypePaymentOutput(input TypePayment) TypePaymentOutput {
	return TypePaymentOutput{
		ID:          input.ID,
		Description: input.Description,
	}
}

func ToCategoryOutput(input Category) CategoryOutput {
	return CategoryOutput{
		ID:          input.ID,
		Description: input.Description,
	}
}

func ToTransactionStatusOutput(input TransactionStatus) TransactionStatusOutput {
	return TransactionStatusOutput{
		ID:          input.ID,
		Description: input.Description,
	}
}

func toConsolidationOutput(input Consolidation) *ConsolidationOutput {
	return &ConsolidationOutput{
		Estimated: input.Estimated,
		Realized:  input.Realized,
		Remaining: input.Remaining,
	}
}
