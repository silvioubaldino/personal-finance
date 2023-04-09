package service

import (
	"context"
	"fmt"

	"personal-finance/internal/domain/movement/repository"
	transactionRepository "personal-finance/internal/domain/transaction/repository"
	"personal-finance/internal/model"

	"github.com/google/uuid"
)

type Transaction interface {
	FindByID(ctx context.Context, id uuid.UUID, userID string) (model.Transaction, error)
	FindByPeriod(ctx context.Context, period model.Period, userID string) ([]model.Transaction, error)
	AddDoneTransaction(ctx context.Context, doneMovement model.Movement) (model.Transaction, error)
}

type transaction struct {
	transactionRepo transactionRepository.Transaction
	movementRepo    repository.Repository
}

func NewTransactionService(transactionRepo transactionRepository.Transaction, movementRepo repository.Repository) Transaction {
	return transaction{
		transactionRepo: transactionRepo,
		movementRepo:    movementRepo,
	}
}

func (s transaction) FindByID(ctx context.Context, id uuid.UUID, userID string) (model.Transaction, error) {
	estimate, err := s.movementRepo.FindByID(ctx, id, userID)
	if err != nil {
		return model.Transaction{}, fmt.Errorf("error to find estimate transactions: %w", err)
	}

	var doneList []model.Movement
	if estimate.StatusID == model.TransactionStatusPlannedID {
		doneList, err = s.movementRepo.FindByTransactionID(ctx, *estimate.ID, model.TransactionStatusPaidID, userID)
		if err != nil {
			return model.Transaction{}, fmt.Errorf("error to find done transactions: %w", err)
		}
	}

	return model.BuildTransaction(estimate, doneList), nil
}

func (s transaction) FindByPeriod(ctx context.Context, period model.Period, userID string) ([]model.Transaction, error) {
	estimates, err := s.movementRepo.FindByStatusByPeriod(ctx, model.TransactionStatusPlannedID, period, userID)
	if err != nil {
		return []model.Transaction{}, fmt.Errorf("error to find planned transactions: %w", err)
	}

	var transactions []model.Transaction
	for _, estimate := range estimates {
		doneList, err := s.movementRepo.FindByTransactionID(ctx, *estimate.ID, model.TransactionStatusPaidID, userID)
		if err != nil {
			return []model.Transaction{}, fmt.Errorf("error to find realized transactions: %w", err)
		}
		transactions = append(transactions, model.BuildTransaction(estimate, doneList))
	}

	if len(transactions) == 0 {
		return []model.Transaction{}, model.BuildErrNotfound("resource not found")
	}
	return transactions, nil
}

func (s transaction) AddDoneTransaction(ctx context.Context, doneMovement model.Movement) (model.Transaction, error) {
	estimate := doneMovement
	estimate.StatusID = model.TransactionStatusPlannedID

	var done model.MovementList
	done[0] = doneMovement
	done[0].StatusID = model.TransactionStatusPaidID
	transaction := model.BuildTransaction(estimate, done)

	transaction, err := s.transactionRepo.AddConsistent(ctx, transaction)
	if err != nil {
		return model.Transaction{}, err
	}
	return transaction, nil
}
