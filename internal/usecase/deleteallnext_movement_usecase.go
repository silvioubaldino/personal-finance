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

func (u *Movement) DeleteAllNext(ctx context.Context, id uuid.UUID, date *time.Time) error {
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

			return u.handleDeleteAllNextVirtual(ctx, tx, recurrent, *date)
		}

		if existingMovement.IsCreditCardMovement() {
			return u.handleCreditCardDeleteAllNext(ctx, tx, &existingMovement)
		}

		if !existingMovement.IsRecurrent && existingMovement.RecurrentID == nil {
			return u.handleDeleteAllNextNonRecurrent(ctx, tx, existingMovement)
		}

		targetDate := *existingMovement.Date
		if date != nil {
			targetDate = *date
		}

		return u.handleDeleteAllNextRecurrent(ctx, tx, existingMovement, targetDate)
	})

	return err
}

func (u *Movement) handleDeleteAllNextNonRecurrent(
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

func (u *Movement) handleDeleteAllNextRecurrent(
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

	updatedRecurrent := u.endRecurrence(recurrent, targetDate)
	_, err = u.recurrentRepo.Update(ctx, tx, recurrent.ID, updatedRecurrent)
	if err != nil {
		return fmt.Errorf("error updating recurrent movement: %w", err)
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

func (u *Movement) handleDeleteAllNextVirtual(
	ctx context.Context,
	tx *gorm.DB,
	recurrent domain.RecurrentMovement,
	targetDate time.Time,
) error {
	updatedRecurrent := u.endRecurrence(recurrent, targetDate)
	_, err := u.recurrentRepo.Update(ctx, tx, recurrent.ID, updatedRecurrent)
	if err != nil {
		return fmt.Errorf("error updating recurrent movement: %w", err)
	}

	return nil
}

func (u *Movement) handleCreditCardDeleteAllNext(
	ctx context.Context,
	tx *gorm.DB,
	existingMovement *domain.Movement,
) error {
	if existingMovement.IsPaid {
		return ErrCreditMovementShouldNotBePaid
	}

	if !existingMovement.IsInstallmentMovement() {
		return u.handleCreditCardMovementDelete(ctx, tx, existingMovement)
	}

	if existingMovement.CreditCardInfo == nil ||
		existingMovement.CreditCardInfo.InstallmentGroupID == nil ||
		existingMovement.CreditCardInfo.InstallmentNumber == nil {
		return ErrUnsupportedMovementTypeV2
	}

	installments, err := u.movementRepo.FindByInstallmentGroupFromNumber(
		ctx,
		*existingMovement.CreditCardInfo.InstallmentGroupID,
		*existingMovement.CreditCardInfo.InstallmentNumber,
	)
	if err != nil {
		return fmt.Errorf("error finding installments: %w", err)
	}

	for _, installment := range installments {
		if installment.CreditCardInfo == nil || installment.CreditCardInfo.InvoiceID == nil {
			return ErrUnsupportedMovementTypeV2
		}

		invoice, err := u.invoiceRepo.FindByID(ctx, *installment.CreditCardInfo.InvoiceID)
		if err != nil {
			return fmt.Errorf("error finding invoice: %w", err)
		}

		if invoice.IsPaid {
			return ErrInvoiceAlreadyPaid
		}
	}

	for _, installment := range installments {
		invoice, err := u.invoiceRepo.FindByID(ctx, *installment.CreditCardInfo.InvoiceID)
		if err != nil {
			return fmt.Errorf("error finding invoice: %w", err)
		}

		newAmount := invoice.Amount - installment.Amount
		_, err = u.invoiceRepo.UpdateAmount(ctx, tx, *installment.CreditCardInfo.InvoiceID, newAmount)
		if err != nil {
			return fmt.Errorf("error updating invoice amount: %w", err)
		}

		_, err = u.creditCardRepo.UpdateLimitDelta(ctx, tx, *installment.CreditCardInfo.CreditCardID, -installment.Amount)
		if err != nil {
			return fmt.Errorf("error updating credit card limit: %w", err)
		}

		err = u.movementRepo.Delete(ctx, tx, *installment.ID)
		if err != nil {
			return fmt.Errorf("error deleting installment: %w", err)
		}
	}

	return nil
}
