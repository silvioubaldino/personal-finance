package repository

import (
	"context"
	"personal-finance/internal/model"
	"personal-finance/internal/model/eager"
	"time"

	"gorm.io/gorm"
)

type Repository interface {
	Add(ctx context.Context, transaction model.Transaction) (model.Transaction, error)
	FindAll(ctx context.Context) ([]model.Transaction, error)
	FindByID(ctx context.Context, id int) (model.Transaction, error)
	FindByIDEager(ctx context.Context, id int) (eager.Transaction, error)
	Update(ctx context.Context, id int, transaction model.Transaction) (model.Transaction, error)
	Delete(ctx context.Context, id int) error
}

type PgRepository struct {
	Gorm *gorm.DB
}

func NewPgRepository(gorm *gorm.DB) Repository {
	return PgRepository{Gorm: gorm}
}

func (p PgRepository) Add(_ context.Context, transaction model.Transaction) (model.Transaction, error) {
	now := time.Now()
	transaction.DateCreate = now
	transaction.DateUpdate = now
	result := p.Gorm.Create(&transaction)
	if err := result.Error; err != nil {
		return model.Transaction{}, err
	}
	return transaction, nil
}

func (p PgRepository) FindAll(_ context.Context) ([]model.Transaction, error) {
	var transactions []model.Transaction
	result := p.Gorm.Find(&transactions)
	if err := result.Error; err != nil {
		return []model.Transaction{}, err
	}
	return transactions, nil
}

func (p PgRepository) FindByID(_ context.Context, id int) (model.Transaction, error) {
	var transaction model.Transaction
	result := p.Gorm.First(&transaction, id)
	if err := result.Error; err != nil {
		return model.Transaction{}, err
	}
	return transaction, nil
}

func (p PgRepository) FindByIDEager(_ context.Context, id int) (eager.Transaction, error) {
	var eagerTransaction eager.Transaction
	result := p.Gorm.Joins("Wallet").Joins("TypePayment").Joins("Category").First(&eagerTransaction, id)
	if err := result.Error; err != nil {
		return eager.Transaction{}, err
	}
	return eagerTransaction, nil
}

func (p PgRepository) Update(_ context.Context, id int, transaction model.Transaction) (model.Transaction, error) {
	transactionFound, err := p.FindByID(context.Background(), id)
	if err != nil {
		return model.Transaction{}, err
	}
	transactionFound.Description = transaction.Description
	transactionFound.Amount = transaction.Amount
	transactionFound.WalletID = transaction.WalletID
	transactionFound.TypePaymentID = transaction.TypePaymentID
	transactionFound.CategoryID = transaction.CategoryID
	transactionFound.DateUpdate = time.Now()
	result := p.Gorm.Save(&transactionFound)
	if result.Error != nil {
		return model.Transaction{}, result.Error
	}
	return transactionFound, nil
}

func (p PgRepository) Delete(_ context.Context, id int) error {
	if err := p.Gorm.Delete(&eager.Transaction{}, id).Error; err != nil {
		return err
	}
	return nil
}
