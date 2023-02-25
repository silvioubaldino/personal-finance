package eager

/*
import (
	"time"

	"github.com/google/uuid"

	"personal-finance/internal/model"
)

type (
	Movement struct {
		ID            *uuid.UUID        `json:"id,omitempty" gorm:"primaryKey"`
		Description   string            `json:"description,omitempty"`
		Amount        float64           `json:"amount"`
		Date          time.Time         `json:"date"`
		TransactionID *uuid.UUID        `json:"-"`
		WalletID      int               `json:"-"`
		Wallet        model.Wallet      `json:"wallets,omitempty"`
		TypePaymentID int               `json:"-"`
		TypePayment   model.TypePayment `json:"type_payments,omitempty"`
		CategoryID    int               `json:"-"`
		Category      model.Category    `json:"categories,omitempty"`
		DateCreate    time.Time         `json:"date_create"`
		DateUpdate    time.Time         `json:"date_update"`
	}

	MovementList []Movement

	Transaction struct {
		TransactionID *uuid.UUID           `json:"transaction_id"`
		Estimate      *Movement            `json:"estimate,omitempty"`
		Consolidation *model.Consolidation `json:"consolidation,omitempty"`
		DoneList      MovementList         `json:"done_list"`
	}
)

func BuildTransactionEager(estimate Movement, doneList MovementList) Transaction {
	pt := Transaction{
		TransactionID: estimate.TransactionID,
		Estimate:      &estimate,
		Consolidation: &model.Consolidation{},
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

func ToOutput(input Transaction) model.TransactionOutput {
	output := model.TransactionOutput{
		Estimate:      toTransactionOutput(input.Estimate),
		Consolidation: input.Consolidation,
		DoneList:      toTransactionListOutput(input.DoneList),
	}
	return output
}

func toTransactionOutput(input *Movement) *model.MovementOutput {
	output := &model.MovementOutput{
		ID:            input.ID,
		Description:   input.Description,
		Amount:        input.Amount,
		Date:          &input.Date,
		TransactionID: input.TransactionID,
		Wallet:        toWalletOutput(input.Wallet),
		TypePayment:   toTypePaymentOutput(input.TypePayment),
		Category:      toCategoryOutput(input.Category),
		DateUpdate:    &input.DateUpdate,
	}
	return output
}

func toTransactionListOutput(input MovementList) model.MovementListOutput {
	output := make(model.MovementListOutput, len(input))
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
*/
