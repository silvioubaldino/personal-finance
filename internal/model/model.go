package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type (
	Wallet struct {
		ID          int       `json:"id,omitempty" gorm:"primaryKey"`
		Description string    `json:"description,omitempty"`
		Balance     float64   `json:"balance"`
		DateCreate  time.Time `json:"date_create"`
		DateUpdate  time.Time `json:"date_update"`
	}

	TypePayment struct {
		ID          int       `json:"id,omitempty" gorm:"primaryKey"`
		Description string    `json:"description,omitempty"`
		DateCreate  time.Time `json:"date_create"`
		DateUpdate  time.Time `json:"date_update"`
	}

	Category struct {
		ID          int       `json:"id,omitempty" gorm:"primaryKey"`
		Description string    `json:"description,omitempty"`
		DateCreate  time.Time `json:"date_create"`
		DateUpdate  time.Time `json:"date_update"`
	}

	TransactionStatus struct {
		ID          int       `json:"id,omitempty" gorm:"primaryKey"`
		Description string    `json:"description,omitempty"`
		DateCreate  time.Time `json:"date_create"`
		DateUpdate  time.Time `json:"date_update"`
	}

	Transaction struct {
		ID                  uuid.UUID `json:"id,omitempty" gorm:"primaryKey"`
		Description         string    `json:"description,omitempty"`
		Amount              float64   `json:"amount"`
		Date                time.Time `json:"date"`
		ParentTransactionID uuid.UUID `json:"parent_transaction_id"`
		WalletID            int       `json:"wallet_id"`
		TypePaymentID       int       `json:"type_payment_id"`
		CategoryID          int       `json:"category_id"`
		TransactionStatusID int       `json:"transaction_status_id"`
		DateCreate          time.Time `json:"date_create"`
		DateUpdate          time.Time `json:"date_update"`
	}

	TransactionList []Transaction

	ConsolidatedTransaction struct {
		ParentTransaction Transaction     `json:"parent_transaction"`
		Consolidation     Consolidation   `json:"consolidation"`
		TransactionList   TransactionList `json:"transaction_list"`
	}

	Consolidation struct {
		Estimated float64 `json:"estimated"`
		Realized  float64 `json:"realized"`
		Remaining float64 `json:"remaining"`
	}

	Balance struct {
		Period  Period  `json:"period"`
		Expense float64 `json:"expense"`
		Income  float64 `json:"income"`
	}

	Period struct {
		From time.Time `json:"from"`
		To   time.Time `json:"to"`
	}
)

func (t TransactionStatus) TableName() string {
	return "transaction_status"
}

func (p *Period) Validate() error {
	now := time.Now()
	if p.From == p.To {
		return errors.New("date must be informed")
	}

	if p.From.IsZero() {
		p.From = now
	}
	if p.To.IsZero() {
		p.To = now
	}

	if p.From.After(p.To) {
		return errors.New("'from' must be before 'to'")
	}

	return nil
}

func BuildParentTransaction(transaction Transaction, list TransactionList) ConsolidatedTransaction {
	pt := ConsolidatedTransaction{
		ParentTransaction: transaction,
		TransactionList:   list,
	}
	pt.Consolidate()
	return pt
}

func (pt *ConsolidatedTransaction) Consolidate() {
	var realized float64
	for _, transaction := range pt.TransactionList {
		realized += transaction.Amount
	}
	pt.Consolidation.Estimated = pt.ParentTransaction.Amount
	pt.Consolidation.Realized = realized
	pt.Consolidation.Remaining = pt.ParentTransaction.Amount - realized
}
