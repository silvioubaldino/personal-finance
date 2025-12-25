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

func (u *Movement) DeleteOne(ctx context.Context, id uuid.UUID, date *time.Time) error {
	err := u.txManager.WithTransaction(ctx, func(tx *gorm.DB) error {
		existingMovement, err := u.movementRepo.FindByID(ctx, id)
		if err != nil {
			if !errors.Is(err, repository.ErrMovementNotFound) {
				return fmt.Errorf("error finding movement: %w", err)
			}

			recurrent, err := u.recurrentRepo.FindByID(ctx, id)
			if err != nil {
				return fmt.Errorf("error finding recurrent movement: %w", err)
			}

			if date == nil {
				return ErrDateRequired
			}

			return u.handleDeleteOneVirtual(ctx, tx, recurrent, *date)
		}

		if existingMovement.IsCreditCardMovement() {
			return u.handleCreditCardMovementDelete(ctx, tx, &existingMovement)
		}

		if !existingMovement.IsRecurrent && existingMovement.RecurrentID == nil {
			return u.handleDeleteOneNonRecurrent(ctx, tx, existingMovement)
		}

		targetDate := *existingMovement.Date
		if date != nil {
			targetDate = *date
		}

		return u.handleDeleteOneRecurrent(ctx, tx, existingMovement, targetDate)
	})

	return err
}

func (u *Movement) handleDeleteOneNonRecurrent(
	ctx context.Context,
	tx *gorm.DB,
	existingMovement domain.Movement,
) error {
	if existingMovement.IsPaid {
		err := u.updateWalletBalance(ctx, tx, existingMovement.WalletID, existingMovement.ReverseAmount())
		if err != nil {
			return fmt.Errorf("error reverting wallet balance: %w", err)
		}
	}

	err := u.movementRepo.Delete(ctx, tx, *existingMovement.ID)
	if err != nil {
		return fmt.Errorf("error deleting movement: %w", err)
	}

	return nil
}

func (u *Movement) handleDeleteOneRecurrent(
	ctx context.Context,
	tx *gorm.DB,
	existingMovement domain.Movement,
	targetDate time.Time,
) error {
	if existingMovement.RecurrentID == nil {
		return fmt.Errorf("recurrent_id is required for recurrent movement")
	}

	recurrent, err := u.recurrentRepo.FindByID(ctx, *existingMovement.RecurrentID)
	if err != nil {
		return fmt.Errorf("error finding recurrent movement: %w", err)
	}

	splitResult, err := u.splitRecurrenceForDeleteOne(ctx, recurrent, targetDate)
	if err != nil {
		return err
	}

	updatedRecurrent := u.endRecurrence(recurrent, targetDate)
	_, err = u.recurrentRepo.Update(ctx, tx, recurrent.ID, updatedRecurrent)
	if err != nil {
		return fmt.Errorf("error updating recurrent movement: %w", err)
	}

	_, err = u.recurrentRepo.Add(ctx, tx, *splitResult)
	if err != nil {
		return fmt.Errorf("error creating new recurrent movement: %w", err)
	}

	if existingMovement.IsPaid {
		err := u.updateWalletBalance(ctx, tx, existingMovement.WalletID, existingMovement.ReverseAmount())
		if err != nil {
			return fmt.Errorf("error reverting wallet balance: %w", err)
		}
	}

	err = u.movementRepo.Delete(ctx, tx, *existingMovement.ID)
	if err != nil {
		return fmt.Errorf("error deleting movement: %w", err)
	}

	return nil
}

func (u *Movement) handleDeleteOneVirtual(
	ctx context.Context,
	tx *gorm.DB,
	recurrent domain.RecurrentMovement,
	targetDate time.Time,
) error {
	splitResult, err := u.splitRecurrenceForDeleteOne(ctx, recurrent, targetDate)
	if err != nil {
		return err
	}

	updatedRecurrent := u.endRecurrence(recurrent, targetDate)
	_, err = u.recurrentRepo.Update(ctx, tx, recurrent.ID, updatedRecurrent)
	if err != nil {
		return fmt.Errorf("error updating recurrent movement: %w", err)
	}

	_, err = u.recurrentRepo.Add(ctx, tx, *splitResult)
	if err != nil {
		return fmt.Errorf("error creating new recurrent movement: %w", err)
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

	err = u.movementRepo.Delete(ctx, tx, *existingMovement.ID)
	if err != nil {
		return fmt.Errorf("error deleting movement: %w", err)
	}

	return nil
}
