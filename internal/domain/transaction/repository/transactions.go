package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	movementRepo "personal-finance/internal/domain/movement/repository"
	"personal-finance/internal/domain/wallet/repository"
	"personal-finance/internal/model"
)

type Transaction interface {
	AddConsistent(ctx context.Context, transaction model.Transaction) (model.Transaction, error)
}
type pgRepository struct {
	gorm               *gorm.DB
	movementRepository movementRepo.Repository
	walletRepository   repository.Repository
}

func NewPgRepository(gorm *gorm.DB, movRepo movementRepo.Repository, walletRepo repository.Repository) Transaction {
	return pgRepository{
		gorm:               gorm,
		movementRepository: movRepo,
		walletRepository:   walletRepo,
	}
}

func (r pgRepository) AddConsistent(ctx context.Context, transaction model.Transaction) (model.Transaction, error) {
	gormTransactionErr := r.gorm.Transaction(func(tx *gorm.DB) error {
		estimateResult, err := r.movementRepository.AddConsistent(ctx, tx, *transaction.Estimate, "userID")
		if err != nil {
			return err
		}

		var doneListResult model.MovementList
		for i := range transaction.DoneList {
			transaction.DoneList[i].TransactionID = estimateResult.ID

			doneResult, err := r.movementRepository.AddUpdatingWallet(ctx, tx, transaction.DoneList[i], "userID")
			if err != nil {
				return err
			}

			doneListResult = append(doneListResult, doneResult)
		}

		transaction = model.BuildTransaction(estimateResult, doneListResult)
		return nil
	})
	if gormTransactionErr != nil {
		return model.Transaction{}, errors.New("repository error")
	}
	return transaction, nil
}
