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

func (u *Movement) UpdateOne(ctx context.Context, id uuid.UUID, newMovement domain.Movement) (domain.Movement, error) {
	if newMovement.Date == nil {
		return domain.Movement{}, fmt.Errorf("movement date is required")
	}

	err := u.validateSubCategory(ctx, newMovement.SubCategoryID, newMovement.CategoryID)
	if err != nil {
		return domain.Movement{}, err
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

			newFromRecurrent := domain.FromRecurrentMovement(recurrent, *newMovement.Date)
			newMovement = update(newMovement, newFromRecurrent)
		}

		if existingMovement.TypePayment == domain.TypePaymentInternalTransfer {
			return ErrTransferMustUsePairEndpoint
		}

		if existingMovement.IsCreditCardMovement() {
			err = u.handleCreditCardMovementUpdate(ctx, tx, &existingMovement, &newMovement)
			if err != nil {
				return err
			}
		}

		if existingMovement.RecurrentID != nil || newMovement.RecurrentID != nil {
			if newMovement.RecurrentID == nil {
				newMovement.RecurrentID = existingMovement.RecurrentID
			}
			err = u.handleRecurrent(ctx, tx, *newMovement.RecurrentID, newMovement)
			if err != nil {
				return err
			}
		}

		if existingMovement.ID == nil {
			result, err = u.movementRepo.Add(ctx, tx, newMovement)
			if err != nil {
				return err
			}
			return nil
		}

		if existingMovement.IsPaid {
			err = u.handlePaid(ctx, tx, existingMovement, newMovement)
			if err != nil {
				return err
			}
		}

		result, err = u.movementRepo.Update(ctx, tx, id, newMovement)
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

func (u *Movement) handleRecurrent(ctx context.Context, tx *gorm.DB, id uuid.UUID, newMovement domain.Movement) error {
	recurrent, err := u.recurrentRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	newRecurrent := recurrent

	endDate := domain.SetMonthYear(*recurrent.InitialDate, newMovement.Date.Month()-1, newMovement.Date.Year())
	recurrent.EndDate = &endDate
	if recurrent.EndDate.Before(*recurrent.InitialDate) {
		// TODO delete recurrent
		recurrent.EndDate = recurrent.InitialDate
	}

	_, err = u.recurrentRepo.Update(ctx, tx, &id, recurrent)
	if err != nil {
		return err
	}

	newInitialDate := domain.SetMonthYear(*newRecurrent.InitialDate, newMovement.Date.Month()+1, newMovement.Date.Year())
	newRecurrent.InitialDate = &newInitialDate

	_, err = u.recurrentRepo.Add(ctx, tx, newRecurrent)
	if err != nil {
		return err
	}

	return nil
}

func (u *Movement) handlePaid(ctx context.Context, tx *gorm.DB, existingMovement, newMovement domain.Movement) error {
	idDiffWallet := existingMovement.WalletID != nil && newMovement.WalletID != nil && *existingMovement.WalletID != *newMovement.WalletID
	isDiffAmount := existingMovement.Amount != newMovement.Amount && newMovement.Amount != 0

	if idDiffWallet {
		err := u.updateWalletBalance(ctx, tx, existingMovement.WalletID, existingMovement.ReverseAmount())
		if err != nil {
			return err
		}
		err = u.updateWalletBalance(ctx, tx, newMovement.WalletID, newMovement.Amount)
		if err != nil {
			return err
		}
		return nil
	}

	if isDiffAmount {
		diff := newMovement.Amount - existingMovement.Amount

		err := u.updateWalletBalance(ctx, tx, existingMovement.WalletID, diff)
		if err != nil {
			return err
		}
		return nil
	}
	return nil
}

func (u *Movement) handleCreditCardMovementUpdate(
	ctx context.Context,
	tx *gorm.DB,
	existingMovement *domain.Movement,
	newMovement *domain.Movement,
) error {
	if existingMovement.IsPaid {
		return errFrozenPurchase()
	}

	if newMovement.IsPaid {
		return ErrCreditMovementShouldNotBePaid
	}

	invoice, err := u.invoiceRepo.FindByID(ctx, *existingMovement.CreditCardInfo.InvoiceID)
	if err != nil {
		return fmt.Errorf("error finding invoice: %w", err)
	}

	if invoice.IsPaid {
		return errFrozenPurchase()
	}

	newAmount := existingMovement.Amount
	if newMovement.Amount != 0 {
		newAmount = newMovement.Amount
	}
	delta := newAmount - existingMovement.Amount

	targetInvoice := invoice
	if newMovement.Date != nil && existingMovement.Date != nil && !newMovement.Date.Equal(*existingMovement.Date) {
		found, err := u.invoiceUseCase.FindOrCreateInvoiceForMovement(
			ctx, nil, existingMovement.CreditCardInfo.CreditCardID, *newMovement.Date,
		)
		if err != nil {
			return fmt.Errorf("error finding/creating invoice: %w", err)
		}
		targetInvoice = found
	}

	if delta != 0 {
		if err := u.validateCreditLimit(ctx, existingMovement.CreditCardInfo.CreditCardID, delta); err != nil {
			return err
		}
	}

	if targetInvoice.ID != nil && *targetInvoice.ID != *invoice.ID {
		return u.moveCreditCardMovementToInvoice(ctx, tx, existingMovement, newMovement, invoice, targetInvoice, newAmount, delta)
	}

	_, err = u.invoiceRepo.UpdateAmount(ctx, tx, *existingMovement.CreditCardInfo.InvoiceID, invoice.Amount+delta)
	if err != nil {
		return fmt.Errorf("error updating invoice amount: %w", err)
	}

	if delta != 0 {
		_, err = u.creditCardRepo.UpdateLimitDelta(ctx, tx, *existingMovement.CreditCardInfo.CreditCardID, delta)
		if err != nil {
			return fmt.Errorf("error updating credit card limit: %w", err)
		}
	}

	return nil
}

// moveCreditCardMovementToInvoice transfere a compra para a fatura da nova data:
// tira o valor antigo da fatura de origem, soma o novo na de destino e ajusta o limite pela diferença.
func (u *Movement) moveCreditCardMovementToInvoice(
	ctx context.Context,
	tx *gorm.DB,
	existingMovement, newMovement *domain.Movement,
	from, to domain.Invoice,
	newAmount, delta float64,
) error {
	if to.IsPaid {
		return errFrozenPurchase()
	}

	if _, err := u.invoiceRepo.UpdateAmount(ctx, tx, *from.ID, from.Amount-existingMovement.Amount); err != nil {
		return fmt.Errorf("error updating invoice amount: %w", err)
	}

	if _, err := u.invoiceRepo.UpdateAmount(ctx, tx, *to.ID, to.Amount+newAmount); err != nil {
		return fmt.Errorf("error updating invoice amount: %w", err)
	}

	if delta != 0 {
		if _, err := u.creditCardRepo.UpdateLimitDelta(ctx, tx, *existingMovement.CreditCardInfo.CreditCardID, delta); err != nil {
			return fmt.Errorf("error updating credit card limit: %w", err)
		}
	}

	info := *existingMovement.CreditCardInfo
	info.InvoiceID = to.ID
	newMovement.CreditCardInfo = &info

	return nil
}

func update(newMovement, movementFound domain.Movement) domain.Movement {
	if newMovement.Description != "" && newMovement.Description != movementFound.Description {
		movementFound.Description = newMovement.Description
	}
	if newMovement.Amount != 0 && newMovement.Amount != movementFound.Amount {
		movementFound.Amount = newMovement.Amount
	}
	if newMovement.Date != nil && *newMovement.Date != *movementFound.Date {
		movementFound.Date = newMovement.Date
	}
	if newMovement.WalletID != nil && *newMovement.WalletID != *movementFound.WalletID {
		movementFound.WalletID = newMovement.WalletID
	}
	if newMovement.TypePayment != "" && newMovement.TypePayment != movementFound.TypePayment {
		movementFound.TypePayment = newMovement.TypePayment
	}
	if newMovement.CategoryID != nil && *newMovement.CategoryID != *movementFound.CategoryID {
		movementFound.CategoryID = newMovement.CategoryID
	}
	if newMovement.SubCategoryID != nil && (movementFound.SubCategoryID == nil || *newMovement.SubCategoryID != *movementFound.SubCategoryID) {
		movementFound.SubCategoryID = newMovement.SubCategoryID
	}
	return movementFound
}
