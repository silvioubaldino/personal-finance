package repository

import (
	"context"
	"errors"
	"net/http"
	"time"

	"personal-finance/internal/domain/wallet/repository"
	"personal-finance/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	Add(ctx context.Context, movement model.Movement, userID string) (model.Movement, error)
	AddConsistent(_ context.Context, tx *gorm.DB, movement model.Movement, userID string) (model.Movement, error)
	AddUpdatingWallet(ctx context.Context, tx *gorm.DB, movement model.Movement, userID string) (model.Movement, error)
	FindByID(_ context.Context, id uuid.UUID, userID string) (model.Movement, error)
	FindByPeriod(ctx context.Context, period model.Period, userID string) ([]model.Movement, error)
	Update(ctx context.Context, id uuid.UUID, transaction model.Movement, userID string) (model.Movement, error)
	Delete(ctx context.Context, id uuid.UUID, userID string) error
	FindByTransactionID(_ context.Context, parentID uuid.UUID, transactionStatusID int, userID string) (model.MovementList, error)
	FindByStatusByPeriod(_ context.Context, transactionStatusID int, period model.Period, userID string) ([]model.Movement, error)
}

type PgRepository struct {
	gorm       *gorm.DB
	walletRepo repository.Repository
}

func NewPgRepository(gorm *gorm.DB, walletRepo repository.Repository) Repository {
	return PgRepository{
		gorm:       gorm,
		walletRepo: walletRepo,
	}
}

func (p PgRepository) Add(_ context.Context, movement model.Movement, userID string) (model.Movement, error) {
	now := time.Now()
	id := uuid.New()

	movement.ID = &id
	movement.DateCreate = now
	movement.DateUpdate = now
	movement.UserID = userID

	if movement.TransactionID == &uuid.Nil {
		movement.TransactionID = movement.ID
	}

	result := p.gorm.Create(&movement)
	if err := result.Error; err != nil {
		return model.Movement{}, handleError("repository error", err)
	}
	return movement, nil
}

func (p PgRepository) FindByID(_ context.Context, id uuid.UUID, userID string) (model.Movement, error) {
	var transaction model.Movement
	result := p.gorm.Where("user_id=?", userID).First(&transaction, id)
	if err := result.Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.Movement{}, model.BuildErrNotfound("resource not found")
		}
		return model.Movement{}, handleError("repository error", err)
	}
	return transaction, nil
}

func (p PgRepository) FindByPeriod(_ context.Context, period model.Period, userID string) ([]model.Movement, error) {
	var transaction []model.Movement
	result := p.gorm.
		Where("user_id=?", userID).
		Where("date BETWEEN ? AND ?", period.From, period.To).
		Find(&transaction)
	if err := result.Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []model.Movement{}, model.BuildErrNotfound("resource not found")
		}
		return []model.Movement{}, handleError("repository error", err)
	}
	return transaction, nil
}

func (p PgRepository) Update(_ context.Context, id uuid.UUID, transaction model.Movement, userID string) (model.Movement, error) {
	transactionFound, err := p.FindByID(context.Background(), id, userID)
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
	result := p.gorm.Updates(&transactionFound)
	if err = result.Error; err != nil {
		return model.Movement{}, handleError("repository error", err)
	}
	return transactionFound, nil
}

func (p PgRepository) Delete(_ context.Context, id uuid.UUID, userID string) error {
	if err := p.gorm.Where("user_id=?", userID).Delete(&model.Movement{}, id).Error; err != nil {
		return handleError("repository error", err)
	}
	return nil
}

func (p PgRepository) FindByTransactionID(_ context.Context, parentID uuid.UUID, transactionStatusID int, userID string) (model.MovementList, error) {
	var transactions model.MovementList
	result := p.gorm.
		Where("movements.user_id=?", userID).
		Where("transaction_id = ?", parentID).
		Where("status_id = ?", transactionStatusID).
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

func (p PgRepository) FindByStatusByPeriod(_ context.Context, transactionStatusID int, period model.Period, userID string) ([]model.Movement, error) {
	var transactions []model.Movement
	result := p.gorm.
		Where("movements.user_id=?", userID).
		Where("status_id = ?", transactionStatusID).
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

func (p PgRepository) AddConsistent(_ context.Context, tx *gorm.DB, movement model.Movement, userID string) (model.Movement, error) {
	now := time.Now()
	id := uuid.New()

	movement.ID = &id
	movement.DateCreate = now
	movement.DateUpdate = now
	movement.UserID = userID

	if movement.TransactionID == &uuid.Nil {
		movement.TransactionID = movement.ID
	}

	result := tx.Create(&movement)
	if err := result.Error; err != nil {
		return model.Movement{}, handleError("repository error", err)
	}
	return movement, nil // TODO recuperar o objeto salvo de result
}

func (p PgRepository) AddUpdatingWallet(ctx context.Context, tx *gorm.DB, movement model.Movement, userID string) (model.Movement, error) {
	if tx != nil {
		mov, err := p.addUpdatingWalletConsistent(ctx, tx, movement, userID)
		if err != nil {
			return model.Movement{}, errors.New("repository error")
		}
		return mov, nil
	}

	gormTransactionErr := p.gorm.Transaction(func(tx *gorm.DB) error {
		_, err := p.addUpdatingWalletConsistent(ctx, tx, movement, userID)
		if err != nil {
			return err
		}
		return nil
	})
	if gormTransactionErr != nil {
		return model.Movement{}, errors.New("repository error")
	}
	return movement, nil
}

func (p PgRepository) addUpdatingWalletConsistent(ctx context.Context, tx *gorm.DB, movement model.Movement, userID string) (model.Movement, error) {
	if movement.StatusID == model.TransactionStatusPlannedID {
		return model.Movement{}, errors.New("estimate can`t update wallet")
	}

	movement, err := p.AddConsistent(ctx, tx, movement, userID)
	if err != nil {
		return model.Movement{}, handleError("repository error", err)
	}

	wallet, err := p.walletRepo.FindByID(ctx, movement.WalletID, userID)
	if err != nil {
		return model.Movement{}, err
	}
	wallet.Balance += movement.Amount
	_, err = p.walletRepo.UpdateConsistent(ctx, tx, wallet, userID)
	if err != nil {
		return model.Movement{}, err
	}

	return movement, nil // TODO recuperar o objeto salvo de result
}