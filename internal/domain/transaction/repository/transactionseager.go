package repository

import (
	"context"
	"errors"

	"personal-finance/internal/model"
	"personal-finance/internal/model/eager"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (p PgRepository) FindByParentTransactionIDEager(_ context.Context, parentID uuid.UUID, transactionStatusID int) ([]eager.Transaction, error) {
	var transactions []eager.Transaction
	result := p.Gorm.
		Where("parent_transaction_id = ?", parentID).
		Where("transaction_status_id = ?", transactionStatusID).
		Joins("Wallet").
		Joins("Category").
		Joins("TypePayment").
		Find(&transactions)
	if err := result.Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []eager.Transaction{}, model.BuildErrNotfound("resource not found")
		}
		return []eager.Transaction{}, handleError("repository error", err)
	}
	return transactions, nil
}

func (p PgRepository) FindByStatusByPeriodEager(_ context.Context, transactionStatusID int, period model.Period) ([]eager.Transaction, error) {
	var transactions []eager.Transaction
	result := p.Gorm.
		Where("transaction_status_id = ?", transactionStatusID).
		Where("date BETWEEN ? AND ?", period.From, period.To).
		Joins("Wallet").
		Joins("Category").
		Joins("TypePayment").
		Find(&transactions)
	if err := result.Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []eager.Transaction{}, model.BuildErrNotfound("resource not found")
		}
		return []eager.Transaction{}, handleError("repository error", err)
	}
	return transactions, nil
}
