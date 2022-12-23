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

	ConsolidatedTransaction struct {
		ParentTransaction *Transaction         `json:"parent_transaction,omitempty"`
		Consolidation     *model.Consolidation `json:"consolidation,omitempty"`
		TransactionList   TransactionList      `json:"transaction_list"`
	}
)

func BuildParentTransactionEager(transaction Transaction, list TransactionList) ConsolidatedTransaction {
	pt := ConsolidatedTransaction{
		ParentTransaction: &transaction,
		Consolidation:     &model.Consolidation{},
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

func ToOutput(input ConsolidatedTransaction) model.ConsolidatedTransactionOutput {
	output := model.ConsolidatedTransactionOutput{
		ParentTransaction: toTransactionOutput(input.ParentTransaction),
		Consolidation:     input.Consolidation,
		TransactionList:   toTransactionListOutput(input.TransactionList),
	}
	return output
}

func toTransactionOutput(input *Transaction) *model.TransactionOutput {
	output := &model.TransactionOutput{
		ID:                  input.ID,
		Description:         input.Description,
		Amount:              input.Amount,
		Date:                &input.Date,
		ParentTransactionID: input.ParentTransactionID,
		Wallet:              toWalletOutput(input.Wallet),
		TypePayment:         toTypePaymentOutput(input.TypePayment),
		Category:            toCategoryOutput(input.Category),
		DateUpdate:          &input.DateUpdate,
	}
	return output
}

func toTransactionListOutput(input TransactionList) model.TransactionListOutput {
	output := make(model.TransactionListOutput, len(input))
	for i, trx := range input {
		output[i] = *toTransactionOutput(&trx)
	}
	return output
}

func toWalletOutput(input model.Wallet) model.WalletOutput {
	return model.WalletOutput{
		ID:          input.ID,
		Description: input.Description,
		Balance:     input.Balance,
	}
}

func toTypePaymentOutput(input model.TypePayment) model.TypePaymentOutput {
	return model.TypePaymentOutput{
		ID:          input.ID,
		Description: input.Description,
	}
}

func toCategoryOutput(input model.Category) model.CategoryOutput {
	return model.CategoryOutput{
		ID:          input.ID,
		Description: input.Description,
	}
}
