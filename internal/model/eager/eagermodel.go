package eager

import (
	"time"

	"github.com/google/uuid"

	"personal-finance/internal/model"
)

type (
	Transaction struct {
		ID                  *uuid.UUID        `json:"id,omitempty" gorm:"primaryKey"`
		Description         string            `json:"description,omitempty"`
		Amount              float64           `json:"amount"`
		Date                time.Time         `json:"date"`
		ParentTransactionID *uuid.UUID        `json:"-"`
		WalletID            int               `json:"-"`
		Wallet              model.Wallet      `json:"wallets,omitempty"`
		TypePaymentID       int               `json:"-"`
		TypePayment         model.TypePayment `json:"type_payments,omitempty"`
		CategoryID          int               `json:"-"`
		Category            model.Category    `json:"categories,omitempty"`
		DateCreate          time.Time         `json:"date_create"`
		DateUpdate          time.Time         `json:"date_update"`
	}

	TransactionList []Transaction

	Consolidation struct {
		Estimated float64 `json:"estimated,omitempty"`
		Realized  float64 `json:"realized,omitempty"`
		Remaining float64 `json:"remaining,omitempty"`
	}

	ConsolidatedTransaction struct {
		ParentTransaction *Transaction    `json:"parent_transaction,omitempty"`
		Consolidation     *Consolidation  `json:"consolidation,omitempty"`
		TransactionList   TransactionList `json:"transaction_list"`
	}
)

func BuildParentTransactionEager(transaction Transaction, list TransactionList) ConsolidatedTransaction {
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
