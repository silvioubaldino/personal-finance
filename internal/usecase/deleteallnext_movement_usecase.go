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

func (u *Movement) DeleteAllNext(ctx context.Context, id uuid.UUID, date time.Time) error {
	return u.txManager.WithTransaction(ctx, func(tx *gorm.DB) error {
		existingMovement, err := u.movementRepo.FindByID(ctx, id)
		if err != nil {
			if !errors.Is(err, repository.ErrMovementNotFound) {
				return fmt.Errorf("error finding movement: %w", err)
			}

			return u.truncateRecurrentByID(ctx, tx, id, date)
		}

		if existingMovement.IsCreditCardMovement() {
			return u.deleteAllNextCreditCard(ctx, tx, id, &existingMovement)
		}

		if existingMovement.RecurrentID != nil {
			return u.deleteAllNextFromRecurrentChain(ctx, tx, &existingMovement, date)
		}

		return u.deleteRegularMovement(ctx, tx, id, &existingMovement)
	})
}

func (u *Movement) deleteAllNextCreditCard(ctx context.Context, tx *gorm.DB, id uuid.UUID, existingMovement *domain.Movement) error {
	if existingMovement.IsPaid {
		return ErrCreditMovementShouldNotBePaid
	}

	if !existingMovement.IsInstallmentMovement() {
		if err := u.handleCreditCardMovementDelete(ctx, tx, existingMovement); err != nil {
			return err
		}

		return u.movementRepo.Delete(ctx, tx, id)
	}

	return u.handleCreditCardDeleteAllNext(ctx, tx, existingMovement)
}

func (u *Movement) deleteAllNextFromRecurrentChain(ctx context.Context, tx *gorm.DB, movement *domain.Movement, date time.Time) error {
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
	if err := u.movementRepo.Delete(ctx, tx, *movement.ID); err != nil {
		return fmt.Errorf("error deleting movement: %w", err)
	}

	return u.truncateRecurrentChain(ctx, tx, &recurrent, effectiveDate)
}

func (u *Movement) truncateRecurrentByID(ctx context.Context, tx *gorm.DB, id uuid.UUID, date time.Time) error {
	recurrent, err := u.recurrentRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("error finding recurrent movement: %w", err)
	}

	if date.IsZero() {
		return ErrDateRequired
	}

	return u.truncateRecurrentChain(ctx, tx, &recurrent, date)
}

func (u *Movement) truncateRecurrentChain(ctx context.Context, tx *gorm.DB, recurrent *domain.RecurrentMovement, date time.Time) error {
	// endDate = last valid month (one before the truncation point)
	endDate := domain.SetMonthYear(*recurrent.InitialDate, date.Month()-1, date.Year())

	if endDate.Before(*recurrent.InitialDate) {
		// Truncating from the first month — delete all movements and the entire recurrent chain
		if err := u.movementRepo.DeleteAllByRecurrentID(ctx, tx, *recurrent.ID); err != nil {
			return fmt.Errorf("error deleting movements for recurrent: %w", err)
		}
		return u.recurrentRepo.Delete(ctx, tx, recurrent.ID)
	}

	updatedRecurrent := *recurrent
	updatedRecurrent.EndDate = &endDate

	_, err := u.recurrentRepo.Update(ctx, tx, recurrent.ID, updatedRecurrent)
	if err != nil {
		return fmt.Errorf("error updating recurrent end date: %w", err)
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
