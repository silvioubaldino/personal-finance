package usecase

import (
	"context"
	"fmt"

	"personal-finance/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (u *Movement) DeleteOne(ctx context.Context, id uuid.UUID) error {
	err := u.txManager.WithTransaction(ctx, func(tx *gorm.DB) error {
		existingMovement, err := u.movementRepo.FindByID(ctx, id)
		if err != nil {
			return fmt.Errorf("error finding movement: %w", err)
		}

		if !existingMovement.IsCreditCardMovement() {
			return ErrUnsupportedMovementTypeV2
		}

		err = u.handleCreditCardMovementDelete(ctx, tx, &existingMovement)
		if err != nil {
			return err
		}

		err = u.movementRepo.Delete(ctx, tx, id)
		if err != nil {
			return fmt.Errorf("error deleting movement: %w", err)
		}

		return nil
	})

	if err != nil {
		return err
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

	return nil
}
