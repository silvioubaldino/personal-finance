package usecase

import (
	"context"
	"errors"
	"fmt"

	"personal-finance/internal/domain"
	"personal-finance/internal/infrastructure/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (u *Movement) UpdateAllNext(ctx context.Context, id uuid.UUID, newMovement domain.Movement) (domain.Movement, error) {
	if newMovement.Date == nil {
		return domain.Movement{}, fmt.Errorf("movement date is required")
	}

	err := u.validateSubCategory(ctx, newMovement.SubCategoryID, newMovement.CategoryID)
	if err != nil {
		return domain.Movement{}, err
	}

	var result domain.Movement
	err = u.txManager.WithTransaction(ctx, func(tx *gorm.DB) error {
		existingMovement, findErr := u.movementRepo.FindByID(ctx, id)
		if findErr != nil && !errors.Is(findErr, repository.ErrMovementNotFound) {
			return findErr
		}

		recurrent, recErr := u.resolveRecurrent(ctx, &existingMovement, id)
		if recErr != nil {
			return recErr
		}

		if recurrent.ID != nil {
			return u.updateAllNextRecurrent(ctx, tx, &existingMovement, &recurrent, newMovement, &result)
		}

		if existingMovement.ID == nil {
			return fmt.Errorf("movement not found")
		}

		return u.updateSingleMovement(ctx, tx, id, &existingMovement, newMovement, &result)
	})
	if err != nil {
		return domain.Movement{}, err
	}

	return result, nil
}

func (u *Movement) resolveRecurrent(ctx context.Context, existingMovement *domain.Movement, id uuid.UUID) (domain.RecurrentMovement, error) {
	if existingMovement.ID != nil && existingMovement.RecurrentID != nil {
		recurrent, err := u.recurrentRepo.FindByID(ctx, *existingMovement.RecurrentID)
		if err != nil && !errors.Is(err, repository.ErrRecurrentMovementNotFound) {
			return domain.RecurrentMovement{}, fmt.Errorf("error finding recurrent: %w", err)
		}
		return recurrent, nil
	}

	if existingMovement.ID == nil {
		recurrent, err := u.recurrentRepo.FindByID(ctx, id)
		if err != nil {
			return domain.RecurrentMovement{}, fmt.Errorf("error finding recurrent: %w", err)
		}
		return recurrent, nil
	}

	return domain.RecurrentMovement{}, nil
}

func (u *Movement) updateAllNextRecurrent(
	ctx context.Context,
	tx *gorm.DB,
	existingMovement *domain.Movement,
	recurrent *domain.RecurrentMovement,
	newMovement domain.Movement,
	result *domain.Movement,
) error {
	// endDate = last valid month of old chain (one before the updated month)
	endDate := domain.SetMonthYear(*recurrent.InitialDate, newMovement.Date.Month()-1, newMovement.Date.Year())
	deletingOldChain := endDate.Before(*recurrent.InitialDate)

	// 1. Create new recurrent chain starting from the updated month
	newRecurrent := domain.ToRecurrentMovement(newMovement)
	newInitialDate := domain.SetMonthYear(*recurrent.InitialDate, newMovement.Date.Month(), newMovement.Date.Year())
	newRecurrent.InitialDate = &newInitialDate
	newRecurrent.EndDate = recurrent.EndDate

	createdRecurrent, err := u.recurrentRepo.Add(ctx, tx, newRecurrent)
	if err != nil {
		return fmt.Errorf("error creating new recurrent: %w", err)
	}

	// 2. Update or create the physical movement for the current month
	if existingMovement.ID != nil {
		if existingMovement.IsCreditCardMovement() {
			if err := u.handleCreditCardMovementUpdate(ctx, tx, existingMovement, &newMovement); err != nil {
				return err
			}
		} else if existingMovement.IsPaid {
			if err := u.handlePaid(ctx, tx, *existingMovement, newMovement); err != nil {
				return err
			}
		}

		newMovement.RecurrentID = createdRecurrent.ID
		updated, err := u.movementRepo.Update(ctx, tx, *existingMovement.ID, newMovement)
		if err != nil {
			return fmt.Errorf("error updating movement: %w", err)
		}
		*result = updated
	} else {
		*result = newMovement
	}

	// 3. Update subsequent physical credit card movements in the old chain
	if existingMovement.IsCreditCardMovement() && !deletingOldChain && existingMovement.ID != nil {
		if err := u.updateFollowingCreditCardMovementsInChain(ctx, tx, *recurrent.ID, *existingMovement.ID, *createdRecurrent.ID, existingMovement.Amount, newMovement.Amount); err != nil {
			return err
		}
	}

	// 4. Retire old chain — current movement FK already updated, clean up any other orphaned movements
	if deletingOldChain {
		if err := u.movementRepo.DeleteAllByRecurrentID(ctx, tx, *recurrent.ID); err != nil {
			return fmt.Errorf("error deleting movements for old recurrent: %w", err)
		}
		if err := u.recurrentRepo.Delete(ctx, tx, recurrent.ID); err != nil {
			return fmt.Errorf("error deleting old recurrent: %w", err)
		}
	} else {
		updatedRecurrent := *recurrent
		updatedRecurrent.EndDate = &endDate

		_, err := u.recurrentRepo.Update(ctx, tx, recurrent.ID, updatedRecurrent)
		if err != nil {
			return fmt.Errorf("error updating recurrent end date: %w", err)
		}
	}

	return nil
}

func (u *Movement) updateFollowingCreditCardMovementsInChain(
	ctx context.Context,
	tx *gorm.DB,
	oldRecurrentID, excludeMovementID, newRecurrentID uuid.UUID,
	oldAmount, newAmount float64,
) error {
	movements, err := u.movementRepo.FindAllByRecurrentID(ctx, oldRecurrentID)
	if err != nil {
		return fmt.Errorf("error finding following credit card movements: %w", err)
	}

	delta := newAmount - oldAmount

	for i := range movements {
		m := &movements[i]
		if m.ID == nil || *m.ID == excludeMovementID {
			continue
		}

		if delta != 0 && m.CreditCardInfo != nil && m.CreditCardInfo.InvoiceID != nil {
			invoice, err := u.invoiceRepo.FindByID(ctx, *m.CreditCardInfo.InvoiceID)
			if err != nil {
				return fmt.Errorf("error finding invoice for movement %s: %w", m.ID, err)
			}
			if !invoice.IsPaid {
				_, err = u.invoiceRepo.UpdateAmount(ctx, tx, *m.CreditCardInfo.InvoiceID, invoice.Amount+delta)
				if err != nil {
					return fmt.Errorf("error updating invoice amount: %w", err)
				}
				_, err = u.creditCardRepo.UpdateLimitDelta(ctx, tx, *m.CreditCardInfo.CreditCardID, delta)
				if err != nil {
					return fmt.Errorf("error updating credit card limit: %w", err)
				}
			}
		}

		m.Amount = newAmount
		m.RecurrentID = &newRecurrentID
		if _, err := u.movementRepo.Update(ctx, tx, *m.ID, *m); err != nil {
			return fmt.Errorf("error updating following movement: %w", err)
		}
	}

	return nil
}

func (u *Movement) updateSingleMovement(
	ctx context.Context,
	tx *gorm.DB,
	id uuid.UUID,
	existingMovement *domain.Movement,
	newMovement domain.Movement,
	result *domain.Movement,
) error {
	if existingMovement.IsPaid {
		if err := u.handlePaid(ctx, tx, *existingMovement, newMovement); err != nil {
			return err
		}
	}

	updated, err := u.movementRepo.Update(ctx, tx, id, newMovement)
	if err != nil {
		return fmt.Errorf("error updating movement: %w", err)
	}
	*result = updated
	return nil
}
