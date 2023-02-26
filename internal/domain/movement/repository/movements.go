package repository

import (
	"context"
	"errors"
	"net/http"
	"time"

	"personal-finance/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	Add(ctx context.Context, transaction model.Movement) (model.Movement, error)
	FindByID(_ context.Context, id uuid.UUID) (model.Movement, error)
	FindByPeriod(ctx context.Context, period model.Period) ([]model.Movement, error)
	Update(ctx context.Context, id uuid.UUID, transaction model.Movement) (model.Movement, error)
	Delete(ctx context.Context, id uuid.UUID) error
	FindByTransactionID(_ context.Context, parentID uuid.UUID, transactionStatusID int) (model.MovementList, error)
	FindByStatusByPeriod(_ context.Context, transactionStatusID int, period model.Period) ([]model.Movement, error)
	FindSingleTransactionByPeriod(_ context.Context, transactionStatusID int, period model.Period) ([]model.Movement, error)
}

type PgRepository struct {
	Gorm *gorm.DB
}

func NewPgRepository(gorm *gorm.DB) Repository {
	return PgRepository{Gorm: gorm}
}

func (p PgRepository) Add(_ context.Context, transaction model.Movement) (model.Movement, error) {
	now := time.Now()
	id := uuid.New()

	transaction.ID = &id
	transaction.DateCreate = now
	transaction.DateUpdate = now

	if transaction.TransactionID == &uuid.Nil {
		transaction.TransactionID = transaction.ID
	}

	result := p.Gorm.Create(&transaction)
	if err := result.Error; err != nil {
		return model.Movement{}, handleError("repository error", err)
	}
	return transaction, nil
}

func (p PgRepository) FindByID(_ context.Context, id uuid.UUID) (model.Movement, error) {
	var transaction model.Movement
	result := p.Gorm.First(&transaction, id)
	if err := result.Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.Movement{}, model.BuildErrNotfound("resource not found")
		}
		return model.Movement{}, handleError("repository error", err)
	}
	return transaction, nil
}

func (p PgRepository) FindByPeriod(_ context.Context, period model.Period) ([]model.Movement, error) {
	var transaction []model.Movement
	result := p.Gorm.Where("date BETWEEN ? AND ?", period.From, period.To).Find(&transaction)
	if err := result.Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []model.Movement{}, model.BuildErrNotfound("resource not found")
		}
		return []model.Movement{}, handleError("repository error", err)
	}
	return transaction, nil
}

func (p PgRepository) Update(_ context.Context, id uuid.UUID, transaction model.Movement) (model.Movement, error) {
	transactionFound, err := p.FindByID(context.Background(), id)
	if err != nil {
		return model.Movement{}, err
	}
	var updated bool
	if transaction.Description != "" {
		transactionFound.Description = transaction.Description
		updated = true
	}
	if transaction.Amount != 0 {
		transactionFound.Amount = transaction.Amount
		updated = true
	}
	if transaction.Date != nil {
		transactionFound.Date = transaction.Date
		updated = true
	}
	if transaction.WalletID != 0 {
		transactionFound.WalletID = transaction.WalletID
		updated = true
	}
	if transaction.TypePaymentID != 0 {
		transactionFound.TypePaymentID = transaction.TypePaymentID
		updated = true
	}
	if transaction.CategoryID != 0 {
		transactionFound.CategoryID = transaction.CategoryID
		updated = true
	}
	if !updated {
		return model.Movement{}, handleError("no changes", errors.New("no changes"))
	}
	transactionFound.DateUpdate = time.Now()
	result := p.Gorm.Updates(&transactionFound)
	if err = result.Error; err != nil {
		return model.Movement{}, handleError("repository error", err)
	}
	return transactionFound, nil
}

func (p PgRepository) Delete(_ context.Context, id uuid.UUID) error {
	if err := p.Gorm.Delete(&model.Movement{}, id).Error; err != nil {
		return handleError("repository error", err)
	}
	return nil
}

func (p PgRepository) FindSingleTransactionByPeriod(_ context.Context, transactionStatusID int, period model.Period) ([]model.Movement, error) {
	var transactions []model.Movement
	result := p.Gorm.
		Where("movement_status_id = ?", transactionStatusID).
		Where("transaction_id = id").
		Where("date BETWEEN ? AND ?", period.From, period.To).
		Find(&transactions)
	if err := result.Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []model.Movement{}, model.BuildErrNotfound("resource not found")
		}
		return []model.Movement{}, handleError("repository error", err)
	}
	return transactions, nil
}

func (p PgRepository) FindByTransactionID(_ context.Context, parentID uuid.UUID, transactionStatusID int) (model.MovementList, error) {
	var transactions model.MovementList
	result := p.Gorm.
		Where("transaction_id = ?", parentID).
		Where("movement_status_id = ?", transactionStatusID).
		Joins("Wallet").
		Joins("Category").
		Joins("TypePayment").
		Find(&transactions)
	if err := result.Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []model.Movement{}, model.BuildErrNotfound("resource not found")
		}
		return []model.Movement{}, handleError("repository error", err)
	}
	return transactions, nil
}

func (p PgRepository) FindByStatusByPeriod(_ context.Context, transactionStatusID int, period model.Period) ([]model.Movement, error) {
	var transactions []model.Movement
	result := p.Gorm.
		Where("movement_status_id = ?", transactionStatusID).
		Where("date BETWEEN ? AND ?", period.From, period.To).
		Joins("Wallet").
		Joins("Category").
		Joins("TypePayment").
		Find(&transactions)
	if err := result.Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []model.Movement{}, model.BuildErrNotfound("resource not found")
		}
		return []model.Movement{}, handleError("repository error", err)
	}
	return transactions, nil
}

func handleError(msg string, err error) error {
	businessErr := model.BusinessError{}
	if ok := errors.As(err, &businessErr); ok {
		return businessErr
	}
	return model.BuildBusinessError(msg, http.StatusInternalServerError, err)
}
