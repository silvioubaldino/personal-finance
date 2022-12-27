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
		ID                  *uuid.UUID `json:"id,omitempty" gorm:"primaryKey"`
		Description         string     `json:"description,omitempty"`
		Amount              float64    `json:"amount,omitempty"`
		Date                *time.Time `json:"date,omitempty"`
		ParentTransactionID *uuid.UUID `json:"parent_transaction_id,omitempty"`
		WalletID            int        `json:"wallet_id,omitempty"`
		TypePaymentID       int        `json:"type_payment_id,omitempty"`
		CategoryID          int        `json:"category_id,omitempty"`
		TransactionStatusID int        `json:"transaction_status_id,omitempty"`
		DateCreate          time.Time  `json:"-"`
		DateUpdate          time.Time  `json:"-"`
	}

	TransactionList []Transaction

	ConsolidatedTransaction struct {
		ParentTransaction *Transaction    `json:"parent_transaction,omitempty"`
		Consolidation     *Consolidation  `json:"consolidation,omitempty"`
		TransactionList   TransactionList `json:"transaction_list"`
	}

	Consolidation struct {
		Estimated float64 `json:"estimated,omitempty"`
		Realized  float64 `json:"realized,omitempty"`
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
		ParentTransaction: &transaction,
		Consolidation:     &Consolidation{},
		TransactionList:   list,
	}
	pt.Consolidate()
	return pt
}

func (pt *ConsolidatedTransaction) Consolidate() {
	emptyTransaction := Transaction{}
	if *pt.ParentTransaction == emptyTransaction {
		return
	}

	var realized float64
	for _, transaction := range pt.TransactionList {
		realized += transaction.Amount
	}
	pt.Consolidation.Estimated = pt.ParentTransaction.Amount
	pt.Consolidation.Realized = realized
	pt.Consolidation.Remaining = pt.ParentTransaction.Amount - realized
}
