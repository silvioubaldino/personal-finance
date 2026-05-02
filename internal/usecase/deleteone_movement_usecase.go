package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/infrastructure/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (u *Movement) DeleteOne(ctx context.Context, id uuid.UUID, date time.Time) error {
	return u.txManager.WithTransaction(ctx, func(tx *gorm.DB) error {
		existingMovement, err := u.movementRepo.FindByID(ctx, id)
		if err != nil {
			if !errors.Is(err, repository.ErrMovementNotFound) {
				return fmt.Errorf("error finding movement: %w", err)
			}

			return u.deleteRecurrentByID(ctx, tx, id, date)
		}

		if existingMovement.IsCreditCardMovement() {
			return u.deleteCreditCardMovement(ctx, tx, id, &existingMovement)
		}

		if existingMovement.RecurrentID != nil {
			return u.deleteOneFromRecurrentChain(ctx, tx, id, &existingMovement, date)
		}

		return u.deleteRegularMovement(ctx, tx, id, &existingMovement)
	})
}

func (u *Movement) deleteCreditCardMovement(ctx context.Context, tx *gorm.DB, id uuid.UUID, movement *domain.Movement) error {
	if err := u.handleCreditCardMovementDelete(ctx, tx, movement); err != nil {
		return err
	}

	return u.movementRepo.Delete(ctx, tx, id)
}

func (u *Movement) deleteOneFromRecurrentChain(ctx context.Context, tx *gorm.DB, id uuid.UUID, movement *domain.Movement, date time.Time) error {
	recurrent, err := u.recurrentRepo.FindByID(ctx, *movement.RecurrentID)
	if err != nil {
		return fmt.Errorf("error finding recurrent movement: %w", err)
	}

	effectiveDate := date
	if effectiveDate.IsZero() {
		if movement.Date == nil {
			return ErrDateRequired
		}
		effectiveDate = *movement.Date
	}

	if movement.IsPaid {
		if err := u.updateWalletBalance(ctx, tx, movement.WalletID, movement.ReverseAmount()); err != nil {
			return fmt.Errorf("error reverting wallet balance: %w", err)
		}
	}

	// Delete physical movement before operating on recurrent to avoid FK constraint violation
	if err := u.movementRepo.Delete(ctx, tx, id); err != nil {
		return fmt.Errorf("error deleting movement: %w", err)
	}

	return u.splitRecurrentChain(ctx, tx, &recurrent, effectiveDate)
}

func (u *Movement) deleteRegularMovement(ctx context.Context, tx *gorm.DB, id uuid.UUID, movement *domain.Movement) error {
	if movement.IsPaid {
		if err := u.updateWalletBalance(ctx, tx, movement.WalletID, movement.ReverseAmount()); err != nil {
			return fmt.Errorf("error reverting wallet balance: %w", err)
		}
	}

	return u.movementRepo.Delete(ctx, tx, id)
}

func (u *Movement) deleteRecurrentByID(ctx context.Context, tx *gorm.DB, id uuid.UUID, date time.Time) error {
	recurrent, err := u.recurrentRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("error finding recurrent movement: %w", err)
	}

	if date.IsZero() {
		return ErrDateRequired
	}

	return u.splitRecurrentChain(ctx, tx, &recurrent, date)
}

func (u *Movement) splitRecurrentChain(ctx context.Context, tx *gorm.DB, recurrent *domain.RecurrentMovement, date time.Time) error {
	// endDate = last valid month (one before the deleted occurrence)
	endDate := domain.SetMonthYear(*recurrent.InitialDate, date.Month()-1, date.Year())
	// newInitialDate = first month of continuation (one after the deleted occurrence)
	newInitialDate := domain.SetMonthYear(*recurrent.InitialDate, date.Month()+1, date.Year())

	if endDate.Before(*recurrent.InitialDate) {
		// Deleting the first occurrence — clean up all referencing movements then delete recurrent
		if err := u.movementRepo.DeleteAllByRecurrentID(ctx, tx, *recurrent.ID); err != nil {
			return fmt.Errorf("error deleting movements for recurrent: %w", err)
		}
		if err := u.recurrentRepo.Delete(ctx, tx, recurrent.ID); err != nil {
			return fmt.Errorf("error deleting recurrent: %w", err)
		}
	} else {
		updatedRecurrent := *recurrent
		updatedRecurrent.EndDate = &endDate

		_, err := u.recurrentRepo.Update(ctx, tx, recurrent.ID, updatedRecurrent)
		if err != nil {
			return fmt.Errorf("error updating recurrent end date: %w", err)
		}
	}

	// Create continuation chain only if there are future months
	if recurrent.EndDate != nil && newInitialDate.After(*recurrent.EndDate) {
		return nil
	}

	newRecurrent := *recurrent
	newRecurrent.ID = nil
	newRecurrent.InitialDate = &newInitialDate

	_, err := u.recurrentRepo.Add(ctx, tx, newRecurrent)
	if err != nil {
		return fmt.Errorf("error creating continuation recurrent: %w", err)
	}

	return nil
}

func (u *Movement) handleCreditCardMovementDelete(
	ctx context.Context,
	tx *gorm.DB,
	existingMovement *domain.Movement,
) error {
	if existingMovement.IsPaid {
		return ErrCreditMovementShouldNotBePaid
	}

	if existingMovement.CreditCardInfo == nil || existingMovement.CreditCardInfo.InvoiceID == nil {
		return ErrUnsupportedMovementTypeV2
	}

	invoice, err := u.invoiceRepo.FindByID(ctx, *existingMovement.CreditCardInfo.InvoiceID)
	if err != nil {
		return fmt.Errorf("error finding invoice: %w", err)
	}

	if invoice.IsPaid {
		return ErrInvoiceAlreadyPaid
	}

	newAmount := invoice.Amount - existingMovement.Amount
	_, err = u.invoiceRepo.UpdateAmount(ctx, tx, *existingMovement.CreditCardInfo.InvoiceID, newAmount)
	if err != nil {
		return fmt.Errorf("error updating invoice amount: %w", err)
	}

	_, err = u.creditCardRepo.UpdateLimitDelta(ctx, tx, *existingMovement.CreditCardInfo.CreditCardID, -existingMovement.Amount)
	if err != nil {
		return fmt.Errorf("error updating credit card limit: %w", err)
	}

	return nil
}
