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
		return domain.Movement{}, ErrDateRequired
	}

	err := u.validateSubCategory(ctx, newMovement.SubCategoryID, newMovement.CategoryID)
	if err != nil {
		return domain.Movement{}, err
	}

	if newMovement.IsRecurrent && newMovement.TypePayment == domain.TypePaymentCreditCard {
		return domain.Movement{}, ErrRecurrentCreditCardNotSupported
	}

	result := domain.Movement{}
	err = u.txManager.WithTransaction(ctx, func(tx *gorm.DB) error {
		existingMovement, err := u.movementRepo.FindByID(ctx, id)
		if err != nil {
			if !errors.Is(err, repository.ErrMovementNotFound) {
				return err
			}

			recurrent, err := u.recurrentRepo.FindByID(ctx, id)
			if err != nil {
				return err
			}

			return u.handleUpdateAllNextFromVirtual(ctx, tx, recurrent, newMovement, &result)
		}

		if existingMovement.IsCreditCardMovement() {
			return u.handleCreditCardUpdateAllNext(ctx, tx, existingMovement, newMovement, &result)
		}

		if !existingMovement.IsRecurrent && existingMovement.RecurrentID == nil {
			return u.handleUpdateAllNextNonRecurrent(ctx, tx, existingMovement, newMovement, &result)
		}

		return u.handleUpdateAllNextRecurrent(ctx, tx, existingMovement, newMovement, &result)
	})
	if err != nil {
		return domain.Movement{}, err
	}

	return result, nil
}

func (u *Movement) handleUpdateAllNextFromVirtual(
	ctx context.Context,
	tx *gorm.DB,
	recurrent domain.RecurrentMovement,
	newMovement domain.Movement,
	result *domain.Movement,
) error {
	targetDate := *newMovement.Date

	newRecurrentValues := domain.ToRecurrentMovement(newMovement)
	splitResult, err := u.splitRecurrenceForUpdateAllNext(ctx, recurrent, targetDate, newRecurrentValues)
	if err != nil {
		return err
	}

	updatedRecurrent := u.endRecurrence(recurrent, targetDate)
	_, err = u.recurrentRepo.Update(ctx, tx, recurrent.ID, updatedRecurrent)
	if err != nil {
		return err
	}

	createdRecurrent, err := u.recurrentRepo.Add(ctx, tx, *splitResult)
	if err != nil {
		return err
	}

	movementToCreate := newMovement
	movementToCreate.RecurrentID = createdRecurrent.ID
	movementToCreate.IsRecurrent = true

	created, err := u.movementRepo.Add(ctx, tx, movementToCreate)
	if err != nil {
		return err
	}

	*result = created
	return nil
}

func (u *Movement) handleUpdateAllNextNonRecurrent(
	ctx context.Context,
	tx *gorm.DB,
	existingMovement domain.Movement,
	newMovement domain.Movement,
	result *domain.Movement,
) error {
	if existingMovement.IsPaid {
		err := u.handlePaid(ctx, tx, existingMovement, newMovement)
		if err != nil {
			return err
		}
	}

	updated, err := u.movementRepo.Update(ctx, tx, *existingMovement.ID, newMovement)
	if err != nil {
		return err
	}

	*result = updated
	return nil
}

func (u *Movement) handleUpdateAllNextRecurrent(
	ctx context.Context,
	tx *gorm.DB,
	existingMovement domain.Movement,
	newMovement domain.Movement,
	result *domain.Movement,
) error {
	if existingMovement.RecurrentID == nil {
		return fmt.Errorf("recurrent_id is required for recurrent movement")
	}

	recurrent, err := u.recurrentRepo.FindByID(ctx, *existingMovement.RecurrentID)
	if err != nil {
		return err
	}

	targetDate := *newMovement.Date

	newRecurrentValues := domain.ToRecurrentMovement(newMovement)
	splitResult, err := u.splitRecurrenceForUpdateAllNext(ctx, recurrent, targetDate, newRecurrentValues)
	if err != nil {
		return err
	}

	updatedRecurrent := u.endRecurrence(recurrent, targetDate)
	_, err = u.recurrentRepo.Update(ctx, tx, recurrent.ID, updatedRecurrent)
	if err != nil {
		return err
	}

	createdRecurrent, err := u.recurrentRepo.Add(ctx, tx, *splitResult)
	if err != nil {
		return err
	}

	if existingMovement.IsPaid {
		err = u.handlePaid(ctx, tx, existingMovement, newMovement)
		if err != nil {
			return err
		}
	}

	newMovement.RecurrentID = createdRecurrent.ID
	newMovement.IsRecurrent = true

	updated, err := u.movementRepo.Update(ctx, tx, *existingMovement.ID, newMovement)
	if err != nil {
		return err
	}

	*result = updated
	return nil
}

func (u *Movement) handleCreditCardUpdateAllNext(
	ctx context.Context,
	tx *gorm.DB,
	existingMovement domain.Movement,
	newMovement domain.Movement,
	result *domain.Movement,
) error {
	if newMovement.IsPaid || existingMovement.IsPaid {
		return ErrCreditMovementShouldNotBePaid
	}

	if !existingMovement.IsInstallmentMovement() {
		return u.handleSingleCreditCardUpdateAllNext(ctx, tx, existingMovement, newMovement, result)
	}

	return u.handleInstallmentCreditCardUpdateAllNext(ctx, tx, existingMovement, newMovement, result)
}

func (u *Movement) handleSingleCreditCardUpdateAllNext(
	ctx context.Context,
	tx *gorm.DB,
	existingMovement domain.Movement,
	newMovement domain.Movement,
	result *domain.Movement,
) error {
	invoice, err := u.invoiceRepo.FindByID(ctx, *existingMovement.CreditCardInfo.InvoiceID)
	if err != nil {
		return fmt.Errorf("error finding invoice: %w", err)
	}

	if invoice.IsPaid {
		return ErrInvoiceAlreadyPaid
	}

	var delta float64
	if existingMovement.Amount != newMovement.Amount && newMovement.Amount != 0 {
		delta = newMovement.Amount - existingMovement.Amount
	}

	if delta != 0 {
		if err := u.validateCreditLimit(ctx, existingMovement.CreditCardInfo.CreditCardID, delta); err != nil {
			return err
		}

		_, err = u.invoiceRepo.UpdateAmount(ctx, tx, *existingMovement.CreditCardInfo.InvoiceID, invoice.Amount+delta)
		if err != nil {
			return fmt.Errorf("error updating invoice amount: %w", err)
		}

		_, err = u.creditCardRepo.UpdateLimitDelta(ctx, tx, *existingMovement.CreditCardInfo.CreditCardID, delta)
		if err != nil {
			return fmt.Errorf("error updating credit card limit: %w", err)
		}
	}

	updated, err := u.movementRepo.Update(ctx, tx, *existingMovement.ID, newMovement)
	if err != nil {
		return err
	}

	*result = updated
	return nil
}

func (u *Movement) handleInstallmentCreditCardUpdateAllNext(
	ctx context.Context,
	tx *gorm.DB,
	existingMovement domain.Movement,
	newMovement domain.Movement,
	result *domain.Movement,
) error {
	if existingMovement.CreditCardInfo.InstallmentGroupID == nil || existingMovement.CreditCardInfo.InstallmentNumber == nil {
		return fmt.Errorf("installment group ID and number are required")
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
		invoice, err := u.invoiceRepo.FindByID(ctx, *installment.CreditCardInfo.InvoiceID)
		if err != nil {
			return fmt.Errorf("error finding invoice: %w", err)
		}

		if invoice.IsPaid {
			return ErrInvoiceAlreadyPaid
		}
	}

	totalDelta := float64(0)
	if existingMovement.Amount != newMovement.Amount && newMovement.Amount != 0 {
		deltaPerInstallment := newMovement.Amount - existingMovement.Amount
		totalDelta = deltaPerInstallment * float64(len(installments))
	}

	if totalDelta != 0 {
		if err := u.validateCreditLimit(ctx, existingMovement.CreditCardInfo.CreditCardID, totalDelta); err != nil {
			return err
		}
	}

	for i, installment := range installments {
		var delta float64
		if existingMovement.Amount != newMovement.Amount && newMovement.Amount != 0 {
			delta = newMovement.Amount - existingMovement.Amount
		}

		if delta != 0 {
			invoice, _ := u.invoiceRepo.FindByID(ctx, *installment.CreditCardInfo.InvoiceID)

			_, err = u.invoiceRepo.UpdateAmount(ctx, tx, *installment.CreditCardInfo.InvoiceID, invoice.Amount+delta)
			if err != nil {
				return fmt.Errorf("error updating invoice amount: %w", err)
			}

			_, err = u.creditCardRepo.UpdateLimitDelta(ctx, tx, *installment.CreditCardInfo.CreditCardID, delta)
			if err != nil {
				return fmt.Errorf("error updating credit card limit: %w", err)
			}
		}

		updatedInstallment := installment
		updatedInstallment.Description = newMovement.Description
		updatedInstallment.Amount = newMovement.Amount

		updated, err := u.movementRepo.Update(ctx, tx, *installment.ID, updatedInstallment)
		if err != nil {
			return err
		}

		if i == 0 {
			*result = updated
		}
	}

	return nil
}
