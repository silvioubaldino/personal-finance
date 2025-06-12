package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/infrastructure/repository/transaction"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type (
	MovementRepository interface {
		Add(ctx context.Context, tx *gorm.DB, movement domain.Movement) (domain.Movement, error)
		FindByPeriod(ctx context.Context, period domain.Period) (domain.MovementList, error)
		FindByID(ctx context.Context, id uuid.UUID) (domain.Movement, error)
		UpdateIsPaid(ctx context.Context, tx *gorm.DB, id uuid.UUID, movement domain.Movement) (domain.Movement, error)
		UpdateOne(ctx context.Context, tx *gorm.DB, id uuid.UUID, movement domain.Movement) (domain.Movement, error)
	}

	RecurrentRepository interface {
		Add(ctx context.Context, tx *gorm.DB, recurrent domain.RecurrentMovement) (domain.RecurrentMovement, error)
		FindByMonth(ctx context.Context, month time.Time) ([]domain.RecurrentMovement, error)
		FindByID(ctx context.Context, id uuid.UUID) (domain.RecurrentMovement, error)
	}

	Movement struct {
		movementRepo    MovementRepository
		recurrentRepo   RecurrentRepository
		walletRepo      WalletRepository
		subCategoryRepo SubCategoryRepository
		txManager       transaction.Manager
	}
)

func NewMovement(
	movementRepo MovementRepository,
	recurrentRepo RecurrentRepository,
	walletRepo WalletRepository,
	subCategoryRepo SubCategoryRepository,
	txManager transaction.Manager,
) Movement {
	return Movement{
		movementRepo:    movementRepo,
		recurrentRepo:   recurrentRepo,
		walletRepo:      walletRepo,
		subCategoryRepo: subCategoryRepo,
		txManager:       txManager,
	}
}

func (u *Movement) isSubCategoryValid(ctx context.Context, subCategoryID, categoryID *uuid.UUID) error {
	if subCategoryID == nil {
		return nil
	}

	isSubCategoryValid, err := u.subCategoryRepo.IsSubCategoryBelongsToCategory(ctx, *subCategoryID, *categoryID)
	if err != nil {
		return err
	}

	if !isSubCategoryValid {
		return domain.WrapInvalidInput(
			domain.New("subcategory does not belong to the provided category"),
			"validate subcategory",
		)
	}

	return nil
}

func (u *Movement) updateWalletBalance(ctx context.Context, tx *gorm.DB, walletID *uuid.UUID, amount float64) error {
	wallet, err := u.walletRepo.FindByID(ctx, walletID)
	if err != nil {
		return err
	}

	if !wallet.HasSufficientBalance(amount) {
		return ErrInsufficientBalance
	}

	wallet.Balance += amount

	return u.walletRepo.UpdateAmount(ctx, tx, wallet.ID, wallet.Balance)
}

func (u *Movement) Add(ctx context.Context, movement domain.Movement) (domain.Movement, error) {
	err := u.isSubCategoryValid(ctx, movement.SubCategoryID, movement.CategoryID)
	if err != nil {
		return domain.Movement{}, err
	}

	var result domain.Movement

	err = u.txManager.WithTransaction(ctx, func(tx *gorm.DB) error {
		if movement.ShouldCreateRecurrent() {
			recurrent := domain.ToRecurrentMovement(movement)

			createdRecurrent, err := u.recurrentRepo.Add(ctx, tx, recurrent)
			if err != nil {
				return err
			}

			movement.RecurrentID = createdRecurrent.ID
		}

		createdMovement, err := u.movementRepo.Add(ctx, tx, movement)
		if err != nil {
			return err
		}

		if movement.IsPaid {
			err = u.updateWalletBalance(ctx, tx, movement.WalletID, movement.Amount)
			if err != nil {
				return err
			}
		}

		result = createdMovement
		return nil
	})
	if err != nil {
		return domain.Movement{}, err
	}

	return result, nil
}

func (u *Movement) FindByPeriod(ctx context.Context, period domain.Period) (domain.MovementList, error) {
	movements, err := u.movementRepo.FindByPeriod(ctx, period)
	if err != nil {
		return []domain.Movement{}, err
	}

	recurrents, err := u.recurrentRepo.FindByMonth(ctx, period.To)
	if err != nil {
		return nil, domain.WrapInternalError(err, "error to find recurrents")
	}

	return mergeMovementsWithRecurrents(movements, recurrents, period.To), nil
}

func (u *Movement) Pay(ctx context.Context, id uuid.UUID, date time.Time) (domain.Movement, error) {
	var result domain.Movement
	var err error

	txError := u.txManager.WithTransaction(ctx, func(tx *gorm.DB) error {
		if result, err = u.payMovement(ctx, tx, id, date); err != nil {
			return fmt.Errorf("error paying movement with id: %s: %w", id, err)
		}

		if err = u.updateWalletBalance(ctx, tx, result.WalletID, result.Amount); err != nil {
			return fmt.Errorf("error updating wallet: %w", err)
		}
		return nil
	})

	if txError != nil {
		return domain.Movement{}, txError
	}
	return result, nil
}

func (u *Movement) payMovement(ctx context.Context, tx *gorm.DB, id uuid.UUID, date time.Time) (domain.Movement, error) {
	movement, err := u.movementRepo.FindByID(ctx, id)
	if err != nil {
		if !errors.Is(err, domain.ErrNotFound) {
			return domain.Movement{}, err
		}

		recurrent, err := u.recurrentRepo.FindByID(ctx, id)
		if err != nil {
			return domain.Movement{}, err
		}

		if date.IsZero() {
			return domain.Movement{}, ErrDateRequired
		}

		mov := domain.FromRecurrentMovement(recurrent, date)
		mov.IsPaid = true

		createdMovement, err := u.movementRepo.Add(ctx, tx, mov)
		if err != nil {
			return domain.Movement{}, err
		}

		return createdMovement, nil
	}

	if movement.IsPaid {
		return domain.Movement{}, ErrMovementAlreadyPaid
	}

	movement.IsPaid = true

	updatedMovement, err := u.movementRepo.UpdateIsPaid(ctx, tx, id, movement)
	if err != nil {
		return domain.Movement{}, err
	}

	return updatedMovement, nil
}

func (u *Movement) RevertPay(ctx context.Context, id uuid.UUID) (domain.Movement, error) {
	var result domain.Movement

	txError := u.txManager.WithTransaction(ctx, func(tx *gorm.DB) error {
		movement, err := u.movementRepo.FindByID(ctx, id)
		if err != nil {
			return fmt.Errorf("error finding movement with id: %s: %w", id, err)
		}

		if !movement.IsPaid {
			return ErrMovementNotPaid
		}

		movement.IsPaid = false

		result, err = u.movementRepo.UpdateIsPaid(ctx, tx, id, movement)
		if err != nil {
			return fmt.Errorf("error updating movement: %w", err)
		}

		if err = u.updateWalletBalance(ctx, tx, result.WalletID, result.ReverseAmount()); err != nil {
			return fmt.Errorf("error updating wallet: %w", err)
		}

		return nil
	})

	if txError != nil {
		return domain.Movement{}, txError
	}
	return result, nil
}

func (u *Movement) UpdateOne(ctx context.Context, id uuid.UUID, newMovement domain.Movement) (domain.Movement, error) {
	err := u.isSubCategoryValid(ctx, newMovement.SubCategoryID, newMovement.CategoryID)
	if err != nil {
		return domain.Movement{}, err
	}

	var result domain.Movement

	err = u.txManager.WithTransaction(ctx, func(tx *gorm.DB) error {
		movementFound, err := u.movementRepo.FindByID(ctx, id)
		if err != nil {
			if !errors.Is(err, domain.ErrNotFound) {
				return err
			}

			recurrent, err := u.recurrentRepo.FindByID(ctx, id)
			if err != nil {
				return fmt.Errorf("error finding recurrent movement: %w", err)
			}

			if newMovement.Date == nil {
				return domain.WrapInvalidInput(
					domain.New("date must be informed for recurrent movement"),
					"validate date",
				)
			}

			mov := domain.FromRecurrentMovement(recurrent, *newMovement.Date)
			mov = u.updateMovementFields(newMovement, mov)
			mov.IsRecurrent = true
			mov.RecurrentID = recurrent.ID

			if mov.IsPaid {
				err = u.updateWalletBalance(ctx, tx, mov.WalletID, mov.Amount)
				if err != nil {
					return err
				}
			}

			createdMovement, err := u.movementRepo.Add(ctx, tx, mov)
			if err != nil {
				return err
			}

			result = createdMovement
			return nil
		}

		updatedMovement := u.updateMovementFields(newMovement, movementFound)

		result, err = u.movementRepo.UpdateOne(ctx, tx, id, updatedMovement)
		if err != nil {
			return fmt.Errorf("error updating movement: %w", err)
		}

		err = u.updateWallet(ctx, tx, movementFound)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return domain.Movement{}, err
	}

	return result, nil
}

func (u *Movement) updateWallet(ctx context.Context, tx *gorm.DB, originalMovement domain.Movement) error {
	if originalMovement.IsPaid {
		err := u.updateWalletBalance(ctx, tx, originalMovement.WalletID, originalMovement.ReverseAmount())
		if err != nil {
			return err
		}
	}

	return nil
}

func (u *Movement) updateMovementFields(newMovement, existingMovement domain.Movement) domain.Movement {
	if newMovement.Description != "" && newMovement.Description != existingMovement.Description {
		existingMovement.Description = newMovement.Description
	}
	if newMovement.Amount != 0 && newMovement.Amount != existingMovement.Amount {
		existingMovement.Amount = newMovement.Amount
	}
	if newMovement.Date != nil && (existingMovement.Date == nil || !newMovement.Date.Equal(*existingMovement.Date)) {
		existingMovement.Date = newMovement.Date
	}
	if newMovement.WalletID != nil && (existingMovement.WalletID == nil || *newMovement.WalletID != *existingMovement.WalletID) {
		existingMovement.WalletID = newMovement.WalletID
	}
	if newMovement.TypePayment != "" && newMovement.TypePayment != existingMovement.TypePayment {
		existingMovement.TypePayment = newMovement.TypePayment
	}
	if newMovement.CategoryID != nil && (existingMovement.CategoryID == nil || *newMovement.CategoryID != *existingMovement.CategoryID) {
		existingMovement.CategoryID = newMovement.CategoryID
	}
	if newMovement.SubCategoryID != nil && (existingMovement.SubCategoryID == nil || *newMovement.SubCategoryID != *existingMovement.SubCategoryID) {
		existingMovement.SubCategoryID = newMovement.SubCategoryID
	}
	if newMovement.IsPaid != existingMovement.IsPaid {
		existingMovement.IsPaid = newMovement.IsPaid
	}
	return existingMovement
}

func mergeMovementsWithRecurrents(
	movements domain.MovementList,
	recurrents []domain.RecurrentMovement,
	date time.Time,
) domain.MovementList {
	recurrentMap := make(map[uuid.UUID]struct{}, len(recurrents))
	for i, mov := range movements {
		if mov.RecurrentID != nil {
			movements[i].IsRecurrent = true
			recurrentMap[*mov.RecurrentID] = struct{}{}
		}
	}

	for _, recurrent := range recurrents {
		if _, ok := recurrentMap[*recurrent.ID]; !ok {
			mov := domain.FromRecurrentMovement(recurrent, date)
			mov.ID = mov.RecurrentID
			movements = append(movements, mov)
		}
	}

	return movements
}
