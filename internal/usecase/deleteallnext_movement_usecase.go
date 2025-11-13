package usecase

import (
	"context"
	"fmt"
	"time"

	"personal-finance/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (u *Movement) DeleteAllNext(ctx context.Context, id uuid.UUID, date time.Time) error {
	err := u.txManager.WithTransaction(ctx, func(tx *gorm.DB) error {
		existingMovement, err := u.movementRepo.FindByID(ctx, id)
		if err != nil {
			return fmt.Errorf("error finding movement: %w", err)
		}

		if !existingMovement.IsCreditCardMovement() {
			return ErrUnsupportedMovementTypeV2
		}

		if existingMovement.IsPaid {
			return ErrCreditMovementShouldNotBePaid
		}

		if !existingMovement.IsInstallmentMovement() {
			err = u.handleCreditCardMovementDelete(ctx, tx, &existingMovement)
			if err != nil {
				return err
			}

			err = u.movementRepo.Delete(ctx, tx, id)
			if err != nil {
				return fmt.Errorf("error deleting movement: %w", err)
			}

			return nil
		}

		err = u.handleCreditCardDeleteAllNext(ctx, tx, &existingMovement)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

func (u *Movement) handleCreditCardDeleteAllNext(
	ctx context.Context,
	tx *gorm.DB,
	existingMovement *domain.Movement,
) error {
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

		err = u.movementRepo.Delete(ctx, tx, *installment.ID)
		if err != nil {
			return fmt.Errorf("error deleting installment: %w", err)
		}
	}

	return nil
}
