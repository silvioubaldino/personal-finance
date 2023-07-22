package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	TransactionStatusPaidID    = 1
	TransactionStatusPlannedID = 2
)

type (
	Wallet struct {
		ID          int       `json:"id,omitempty" gorm:"primaryKey"`
		Description string    `json:"description,omitempty"`
		Balance     float64   `json:"balance"`
		UserID      string    `json:"user_id"`
		DateCreate  time.Time `json:"date_create"`
		DateUpdate  time.Time `json:"date_update"`
	}

	TypePayment struct {
		ID          int       `json:"id,omitempty" gorm:"primaryKey"`
		Description string    `json:"description,omitempty"`
		UserID      string    `json:"user_id"`
		DateCreate  time.Time `json:"date_create"`
		DateUpdate  time.Time `json:"date_update"`
	}

	Category struct {
		ID          int       `json:"id,omitempty" gorm:"primaryKey"`
		Description string    `json:"description,omitempty"`
		UserID      string    `json:"user_id"`
		DateCreate  time.Time `json:"date_create"`
		DateUpdate  time.Time `json:"date_update"`
	}

	TransactionStatus struct {
		ID          int       `json:"id,omitempty" gorm:"primaryKey"`
		Description string    `json:"description,omitempty"`
		DateCreate  time.Time `json:"date_create"`
		DateUpdate  time.Time `json:"date_update"`
	}

	Movement struct {
		ID            *uuid.UUID  `json:"id,omitempty" gorm:"primaryKey"`
		Description   string      `json:"description,omitempty"`
		Amount        float64     `json:"amount"`
		Date          *time.Time  `json:"date"`
		TransactionID *uuid.UUID  `json:"transaction_id,omitempty"`
		UserID        string      `json:"user_id"`
		StatusID      int         `json:"status_id,omitempty"`
		WalletID      int         `json:"wallet_id,omitempty"`
		Wallet        Wallet      `json:"wallets,omitempty"`
		TypePaymentID int         `json:"type_payment_id,omitempty"`
		TypePayment   TypePayment `json:"type_payments,omitempty"`
		CategoryID    int         `json:"category_id,omitempty"`
		Category      Category    `json:"categories,omitempty"`
		DateCreate    time.Time   `json:"date_create"`
		DateUpdate    time.Time   `json:"date_update"`
	}

	MovementList []Movement

	Transaction struct {
		TransactionID *uuid.UUID     `json:"transaction_id"`
		Estimate      *Movement      `json:"estimate,omitempty"`
		Consolidation *Consolidation `json:"consolidation,omitempty"`
		DoneList      MovementList   `json:"done_list"`
	}

	Consolidation struct {
		Estimated float64 `json:"estimated,omitempty"`
		Realized  float64 `json:"realized,omitempty"`
		Remaining float64 `json:"remaining"`
	}

	Balance struct {
		Expense       float64 `json:"expense"`
		Income        float64 `json:"income"`
		PeriodBalance float64 `json:"period_balance"`
	}

	Period struct {
		From time.Time `json:"from"`
		To   time.Time `json:"to"`
	}
)

func (t TransactionStatus) TableName() string {
	return "movement_status"
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

func BuildTransaction(estimate Movement, doneList MovementList) Transaction {
	pt := Transaction{
		TransactionID: estimate.ID,
		Estimate:      &estimate,
		Consolidation: &Consolidation{},
		DoneList:      doneList,
	}
	pt.Consolidate()
	return pt
}

func (pt *Transaction) Consolidate() {
	emptyMovement := Movement{}
	if *pt.Estimate == emptyMovement {
		return
	}

	var realized float64
	for _, transaction := range pt.DoneList {
		realized += transaction.Amount
	}
	pt.Consolidation.Estimated = pt.Estimate.Amount
	pt.Consolidation.Realized = realized
	pt.Consolidation.Remaining = pt.Estimate.Amount - realized
}

func (b *Balance) Consolidate() {
	b.PeriodBalance = b.Income + b.Expense
}
