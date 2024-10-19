package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"personal-finance/internal/domain/movement/repository"
	subCategoryRepository "personal-finance/internal/domain/subcategory/repository"
	"personal-finance/internal/domain/transaction/service"
	"personal-finance/internal/model"
)

type Movement interface {
	Add(ctx context.Context, transaction model.Movement, userID string) (model.Movement, error)
	AddSimple(ctx context.Context, transaction model.Movement, userID string) (model.Movement, error)
	FindByID(ctx context.Context, id uuid.UUID, userID string) (model.Movement, error)
	FindByPeriod(ctx context.Context, period model.Period, userID string) ([]model.Movement, error)
	Pay(ctx context.Context, id uuid.UUID, userID string) (model.Movement, error)
	Update(ctx context.Context, id uuid.UUID, transaction model.Movement, userID string) (model.Movement, error)
	Delete(ctx context.Context, id uuid.UUID, userID string) error
}

type movement struct {
	repo            repository.Repository
	subCategoryRepo subCategoryRepository.Repository
	transactionSvc  service.Transaction
}

func NewMovementService(repo repository.Repository, subCategoryRepo subCategoryRepository.Repository, transactionSvc service.Transaction) Movement {
	return movement{
		repo:            repo,
		subCategoryRepo: subCategoryRepo,
		transactionSvc:  transactionSvc,
	}
}

func (s movement) Add(ctx context.Context, movement model.Movement, userID string) (model.Movement, error) {
	if movement.TransactionID == nil {
		if movement.StatusID == model.TransactionStatusPlannedID {
			movement, err := s.repo.Add(ctx, movement, userID)
			if err != nil {
				return model.Movement{}, fmt.Errorf("error to add transactions: %w", err)
			}
			return movement, nil
		}

		if movement.StatusID == model.TransactionStatusPaidID {
			transaction, err := s.transactionSvc.AddDirectDoneTransaction(ctx, movement, userID)
			if err != nil {
				return model.Movement{}, fmt.Errorf("error to add transactions: %w", err)
			}
			return *transaction.Estimate, nil
		}
	}

	if movement.StatusID == model.TransactionStatusPlannedID {
		return model.Movement{}, errors.New("planned transactions must not have transactionID")
	}

	movement, err := s.repo.AddUpdatingWallet(ctx, nil, movement, userID)
	if err != nil {
		return model.Movement{}, fmt.Errorf("error to add transactions: %w", err)
	}
	return movement, nil
}

func (s movement) AddSimple(ctx context.Context, movement model.Movement, userID string) (model.Movement, error) {
	sub, err := s.subCategoryRepo.FindByID(ctx, movement.SubCategoryID, userID)
	if err != nil {
		return model.Movement{}, fmt.Errorf("error to find subcategory: %w", err)
	}

	if sub.CategoryID != movement.CategoryID {
		return model.Movement{}, errors.New("subcategory does not belong to the category")
	}

	if movement.IsPaid {
		movement.StatusID = 1
		movement, err := s.repo.AddUpdatingWallet(ctx, nil, movement, userID)
		if err != nil {
			return model.Movement{}, fmt.Errorf("error to add transactions: %w", err)
		}
		return movement, nil
	}

	movement.StatusID = 2
	CreatedMovement, err := s.repo.Add(ctx, movement, userID)
	if err != nil {
		return model.Movement{}, fmt.Errorf("error to add transactions: %w", err)
	}
	return CreatedMovement, nil
}

func (s movement) FindByID(ctx context.Context, id uuid.UUID, userID string) (model.Movement, error) {
	result, err := s.repo.FindByID(ctx, id, userID)
	if err != nil {
		return model.Movement{}, fmt.Errorf("error to find transactions: %w", err)
	}
	return result, nil
}

func (s movement) FindByPeriod(ctx context.Context, period model.Period, userID string) ([]model.Movement, error) {
	result, err := s.repo.FindByPeriod(ctx, period, userID)
	if err != nil {
		return []model.Movement{}, fmt.Errorf("error to find transactions: %w", err)
	}
	return result, nil
}

func (s movement) Pay(ctx context.Context, id uuid.UUID, userID string) (model.Movement, error) {
	movement, err := s.repo.FindByID(ctx, id, userID)
	if err != nil {
		return model.Movement{}, fmt.Errorf("error finding transactions: %w", err)
	}

	if movement.StatusID == model.TransactionStatusPaidID || movement.IsPaid {
		return model.Movement{}, errors.New("transaction already paid")
	}

	movement.IsPaid = true

	result, err := s.repo.Update(ctx, id, movement, userID)
	if err != nil {
		return model.Movement{}, fmt.Errorf("error updating transactions: %w", err)
	}
	return result, nil
}

func (s movement) Update(ctx context.Context, id uuid.UUID, transaction model.Movement, userID string) (model.Movement, error) {
	result, err := s.repo.Update(ctx, id, transaction, userID)
	if err != nil {
		return model.Movement{}, fmt.Errorf("error updating transactions: %w", err)
	}
	return result, nil
}

func (s movement) Delete(ctx context.Context, id uuid.UUID, userID string) error {
	err := s.repo.Delete(ctx, id, userID)
	if err != nil {
		return fmt.Errorf("error deleting transactions: %w", err)
	}
	return nil
}
