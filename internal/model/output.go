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

	TransactionOutput struct {
		ID                  *uuid.UUID        `json:"id,omitempty" gorm:"primaryKey"`
		Description         string            `json:"description,omitempty"`
		Amount              float64           `json:"amount"`
		Date                *time.Time        `json:"date,omitempty"`
		ParentTransactionID *uuid.UUID        `json:"parent_transaction_id"`
		Wallet              WalletOutput      `json:"wallets,omitempty"`
		TypePayment         TypePaymentOutput `json:"type_payments,omitempty"`
		Category            CategoryOutput    `json:"categories,omitempty"`
		DateUpdate          *time.Time        `json:"date_update,omitempty"`
	}

	TransactionListOutput []TransactionOutput

	ConsolidatedTransactionOutput struct {
		ParentTransaction *TransactionOutput    `json:"parent_transaction,omitempty"`
		Consolidation     *Consolidation        `json:"consolidation,omitempty"`
		TransactionList   TransactionListOutput `json:"transaction_list"`
	}
)
