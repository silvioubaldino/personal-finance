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
		TransactionID *uuid.UUID         `json:"transaction_id"`
		Estimate      *MovementOutput    `json:"estimate,omitempty"`
		Consolidation *Consolidation     `json:"consolidation,omitempty"`
		DoneList      MovementListOutput `json:"done_list"`
	}
)

func ToOutput(input Transaction) TransactionOutput {
	output := TransactionOutput{
		TransactionID: input.TransactionID,
		Estimate:      ToMovementOutput(input.Estimate),
		Consolidation: input.Consolidation,
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
		Wallet:        toWalletOutput(input.Wallet),
		TypePayment:   toTypePaymentOutput(input.TypePayment),
		Category:      toCategoryOutput(input.Category),
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

func toWalletOutput(input Wallet) WalletOutput {
	return WalletOutput{
		ID:          input.ID,
		Description: input.Description,
		Balance:     input.Balance,
	}
}

func toTypePaymentOutput(input TypePayment) TypePaymentOutput {
	return TypePaymentOutput{
		ID:          input.ID,
		Description: input.Description,
	}
}

func toCategoryOutput(input Category) CategoryOutput {
	return CategoryOutput{
		ID:          input.ID,
		Description: input.Description,
	}
}
