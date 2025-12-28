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

func (u *Movement) DeleteAllNext(ctx context.Context, id uuid.UUID, targetDate time.Time) error {
	return u.txManager.WithTransaction(ctx, func(tx *gorm.DB) error {
		return u.executeDeleteAllNext(ctx, tx, id, targetDate)
	})
}

func (u *Movement) executeDeleteAllNext(ctx context.Context, tx *gorm.DB, id uuid.UUID, targetDate time.Time) error {
	existingMovement, isVirtual, recurrentID, err := u.resolveMovementForDelete(ctx, id, targetDate)
	if err != nil {
		return err
	}

	if recurrentID == nil {
		return u.deleteSingleMovement(ctx, tx, id, existingMovement)
	}

	effectiveTargetDate := targetDate
	if !isVirtual && existingMovement.Date != nil {
		effectiveTargetDate = *existingMovement.Date
	}

	if err := u.endRecurrentSeriesAtPreviousMonth(ctx, tx, *recurrentID, effectiveTargetDate); err != nil {
		return err
	}

	if isVirtual {
		return nil
	}

	if existingMovement.IsPaid {
		if err := u.revertWalletBalance(ctx, tx, existingMovement); err != nil {
			return err
		}
	}

	return u.movementRepo.Delete(ctx, tx, id)
}

func (u *Movement) deleteSingleMovement(ctx context.Context, tx *gorm.DB, id uuid.UUID, movement domain.Movement) error {
	if movement.IsPaid {
		if err := u.revertWalletBalance(ctx, tx, movement); err != nil {
			return err
		}
	}

	return u.movementRepo.Delete(ctx, tx, id)
}

func (u *Movement) endRecurrentSeriesAtPreviousMonth(ctx context.Context, tx *gorm.DB, recurrentID uuid.UUID, targetDate time.Time) error {
	recurrent, err := u.recurrentRepo.FindByID(ctx, recurrentID)
	if err != nil {
		if errors.Is(err, repository.ErrRecurrentMovementNotFound) {
			return nil
		}
		return fmt.Errorf("error finding recurrent: %w", err)
	}

	targetMonth := targetDate.Month()
	targetYear := targetDate.Year()

	endDate := domain.SetMonthYear(*recurrent.InitialDate, targetMonth-1, targetYear)

	if !endDate.After(*recurrent.InitialDate) {
		return u.recurrentRepo.Delete(ctx, tx, recurrentID)
	}

	recurrent.EndDate = &endDate
	if _, err := u.recurrentRepo.Update(ctx, tx, &recurrentID, recurrent); err != nil {
		return fmt.Errorf("error updating recurrent end date: %w", err)
	}

	return nil
}

func (u *Movement) DeleteCreditCardAllNext(ctx context.Context, id uuid.UUID) error {
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

		if existingMovement.IsInstallmentMovement() {
			return u.handleCreditCardDeleteAllNext(ctx, tx, &existingMovement)
		}

		// Movement de cartão sem parcelas - delete simples com ajuste de limite
		return u.deleteCreditCardMovement(ctx, tx, id, existingMovement)
	})

	return err
}

func (u *Movement) deleteCreditCardMovement(ctx context.Context, tx *gorm.DB, id uuid.UUID, movement domain.Movement) error {
	if movement.CreditCardInfo == nil || movement.CreditCardInfo.CreditCardID == nil {
		return fmt.Errorf("credit card info is required")
	}

	// Atualiza o valor da fatura
	if movement.CreditCardInfo.InvoiceID != nil {
		invoice, err := u.invoiceRepo.FindByID(ctx, *movement.CreditCardInfo.InvoiceID)
		if err != nil {
			return fmt.Errorf("error finding invoice: %w", err)
		}

		newAmount := invoice.Amount - movement.Amount
		_, err = u.invoiceRepo.UpdateAmount(ctx, tx, *movement.CreditCardInfo.InvoiceID, newAmount)
		if err != nil {
			return fmt.Errorf("error updating invoice amount: %w", err)
		}
	}

	// Atualiza o limite do cartão
	_, err := u.creditCardRepo.UpdateLimitDelta(ctx, tx, *movement.CreditCardInfo.CreditCardID, -movement.Amount)
	if err != nil {
		return fmt.Errorf("error updating credit card limit: %w", err)
	}

	// Deleta a movement
	return u.movementRepo.Delete(ctx, tx, id)
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

	// Primeiro valida que nenhuma fatura está paga
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

	// Processa cada parcela
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
