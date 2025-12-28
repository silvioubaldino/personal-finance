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
	if err := u.validateUpdateOneInput(ctx, newMovement); err != nil {
		return domain.Movement{}, err
	}

	var result domain.Movement
	err := u.txManager.WithTransaction(ctx, func(tx *gorm.DB) error {
		var txErr error
		result, txErr = u.executeUpdateOne(ctx, tx, id, newMovement)
		return txErr
	})
	if err != nil {
		return domain.Movement{}, err
	}

	return result, nil
}

func (u *Movement) validateUpdateOneInput(ctx context.Context, newMovement domain.Movement) error {
	if newMovement.Date == nil {
		return fmt.Errorf("movement date is required")
	}

	return u.validateSubCategory(ctx, newMovement.SubCategoryID, newMovement.CategoryID)
}

func (u *Movement) executeUpdateOne(ctx context.Context, tx *gorm.DB, id uuid.UUID, newMovement domain.Movement) (domain.Movement, error) {
	existingMovement, isVirtual, err := u.resolveMovement(ctx, id, newMovement)
	if err != nil {
		return domain.Movement{}, err
	}

	if existingMovement.IsCreditCardMovement() {
		if err := u.handleCreditCardMovementUpdate(ctx, tx, &existingMovement, &newMovement); err != nil {
			return domain.Movement{}, err
		}
	}

	if u.isRecurrentUpdate(existingMovement, newMovement) {
		if err := u.splitRecurrentSeriesForUpdateOne(ctx, tx, newMovement); err != nil {
			return domain.Movement{}, err
		}
	}

	if isVirtual {
		return u.createRealMovementFromVirtual(ctx, tx, existingMovement, newMovement)
	}

	if existingMovement.IsPaid {
		if err := u.handlePaidMovementUpdate(ctx, tx, existingMovement, newMovement); err != nil {
			return domain.Movement{}, err
		}
	}

	return u.movementRepo.Update(ctx, tx, id, newMovement)
}

func (u *Movement) resolveMovement(ctx context.Context, id uuid.UUID, newMovement domain.Movement) (domain.Movement, bool, error) {
	existingMovement, err := u.movementRepo.FindByID(ctx, id)
	if err == nil {
		return existingMovement, false, nil
	}

	if !errors.Is(err, repository.ErrMovementNotFound) {
		return domain.Movement{}, false, err
	}

	recurrent, err := u.recurrentRepo.FindByID(ctx, id)
	if err != nil {
		return domain.Movement{}, false, fmt.Errorf("movement or recurrent not found: %w", err)
	}

	movementFromRecurrent := domain.FromRecurrentMovement(recurrent, *newMovement.Date)
	mergedMovement := mergeMovementFields(newMovement, movementFromRecurrent)

	return mergedMovement, true, nil
}

func (u *Movement) isRecurrentUpdate(existing, new domain.Movement) bool {
	return new.IsRecurrent || existing.IsRecurrent
}

func (u *Movement) splitRecurrentSeriesForUpdateOne(ctx context.Context, tx *gorm.DB, newMovement domain.Movement) error {
	if newMovement.RecurrentID == nil {
		return fmt.Errorf("recurrent_id is required for recurrent movement update")
	}

	recurrent, err := u.recurrentRepo.FindByID(ctx, *newMovement.RecurrentID)
	if err != nil {
		return fmt.Errorf("error finding recurrent: %w", err)
	}

	// Preserva o EndDate original ANTES de modificar a recorrência
	originalEndDate := recurrent.EndDate

	targetMonth := newMovement.Date.Month()
	targetYear := newMovement.Date.Year()

	endDate := domain.SetMonthYear(*recurrent.InitialDate, targetMonth-1, targetYear)

	// Se a endDate calculada for antes ou igual à initialDate, significa que estamos
	// atualizando no primeiro mês da recorrência - devemos DELETAR e recriar a recorrência
	if !endDate.After(*recurrent.InitialDate) {
		// Se não há meses futuros (era o último mês), simplesmente deleta a recorrência
		if originalEndDate != nil {
			newInitialDate := domain.SetMonthYear(*recurrent.InitialDate, targetMonth+1, targetYear)
			if !newInitialDate.Before(*originalEndDate) {
				return u.recurrentRepo.Delete(ctx, tx, *newMovement.RecurrentID)
			}
		}

		// Deleta a recorrência atual e cria nova começando em T+1 (com dados originais)
		if err := u.recurrentRepo.Delete(ctx, tx, *newMovement.RecurrentID); err != nil {
			return fmt.Errorf("error deleting recurrent: %w", err)
		}

		newInitialDate := domain.SetMonthYear(*recurrent.InitialDate, targetMonth+1, targetYear)
		newRecurrent := recurrent
		newRecurrent.ID = nil
		newRecurrent.InitialDate = &newInitialDate
		newRecurrent.EndDate = originalEndDate

		if _, err := u.recurrentRepo.Add(ctx, tx, newRecurrent); err != nil {
			return fmt.Errorf("error creating new recurrent series: %w", err)
		}

		return nil
	}

	recurrent.EndDate = &endDate
	if _, err := u.recurrentRepo.Update(ctx, tx, newMovement.RecurrentID, recurrent); err != nil {
		return fmt.Errorf("error updating recurrent end date: %w", err)
	}

	// Calcula a data inicial da nova recorrência (T+1)
	newInitialDate := domain.SetMonthYear(*recurrent.InitialDate, targetMonth+1, targetYear)

	// Se a recorrência original já terminava no mês do update (ou antes de T+1),
	// NÃO cria nova recorrência - a série simplesmente encerra aqui
	if originalEndDate != nil && !newInitialDate.Before(*originalEndDate) {
		return nil
	}

	newRecurrent := recurrent
	newRecurrent.ID = nil
	newRecurrent.InitialDate = &newInitialDate
	newRecurrent.EndDate = originalEndDate // Preserva o EndDate original

	if _, err := u.recurrentRepo.Add(ctx, tx, newRecurrent); err != nil {
		return fmt.Errorf("error creating new recurrent series: %w", err)
	}

	return nil
}

func (u *Movement) createRealMovementFromVirtual(ctx context.Context, tx *gorm.DB, base, new domain.Movement) (domain.Movement, error) {
	movement := mergeMovementFields(new, base)

	movement.ID = nil

	if movement.IsPaid {
		if err := u.updateWalletBalance(ctx, tx, movement.WalletID, movement.Amount); err != nil {
			return domain.Movement{}, err
		}
	}

	return u.movementRepo.Add(ctx, tx, movement)
}

func (u *Movement) handlePaidMovementUpdate(ctx context.Context, tx *gorm.DB, existing, new domain.Movement) error {
	isDiffWallet := existing.WalletID != nil && new.WalletID != nil && *existing.WalletID != *new.WalletID
	isDiffAmount := existing.Amount != new.Amount && new.Amount != 0

	if isDiffWallet {
		if err := u.updateWalletBalance(ctx, tx, existing.WalletID, existing.ReverseAmount()); err != nil {
			return err
		}
		return u.updateWalletBalance(ctx, tx, new.WalletID, new.Amount)
	}

	if isDiffAmount {
		diff := new.Amount - existing.Amount
		return u.updateWalletBalance(ctx, tx, existing.WalletID, diff)
	}

	return nil
}

func (u *Movement) handleCreditCardMovementUpdate(
	ctx context.Context,
	tx *gorm.DB,
	existingMovement *domain.Movement,
	newMovement *domain.Movement,
) error {
	if newMovement.IsPaid || existingMovement.IsPaid {
		return ErrCreditMovementShouldNotBePaid
	}

	if existingMovement.CreditCardInfo == nil || existingMovement.CreditCardInfo.InvoiceID == nil {
		return fmt.Errorf("credit card info is required for credit card movement")
	}

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
	}

	if _, err := u.invoiceRepo.UpdateAmount(ctx, tx, *existingMovement.CreditCardInfo.InvoiceID, invoice.Amount+delta); err != nil {
		return fmt.Errorf("error updating invoice amount: %w", err)
	}

	if delta != 0 {
		if _, err := u.creditCardRepo.UpdateLimitDelta(ctx, tx, *existingMovement.CreditCardInfo.CreditCardID, delta); err != nil {
			return fmt.Errorf("error updating credit card limit: %w", err)
		}
	}

	return nil
}

func mergeMovementFields(new, base domain.Movement) domain.Movement {
	result := base

	if new.Description != "" && new.Description != base.Description {
		result.Description = new.Description
	}
	if new.Amount != 0 && new.Amount != base.Amount {
		result.Amount = new.Amount
	}
	if new.Date != nil && (base.Date == nil || *new.Date != *base.Date) {
		result.Date = new.Date
	}
	if new.WalletID != nil && (base.WalletID == nil || *new.WalletID != *base.WalletID) {
		result.WalletID = new.WalletID
	}
	if new.TypePayment != "" && new.TypePayment != base.TypePayment {
		result.TypePayment = new.TypePayment
	}
	if new.CategoryID != nil && (base.CategoryID == nil || *new.CategoryID != *base.CategoryID) {
		result.CategoryID = new.CategoryID
	}
	if new.SubCategoryID != nil && (base.SubCategoryID == nil || *new.SubCategoryID != *base.SubCategoryID) {
		result.SubCategoryID = new.SubCategoryID
	}
	if new.IsPaid != base.IsPaid {
		result.IsPaid = new.IsPaid
	}

	return result
}

func update(newMovement, movementFound domain.Movement) domain.Movement {
	return mergeMovementFields(newMovement, movementFound)
}

func (u *Movement) handleRecurrent(ctx context.Context, tx *gorm.DB, id uuid.UUID, newMovement domain.Movement) error {
	newMovement.RecurrentID = &id
	return u.splitRecurrentSeriesForUpdateOne(ctx, tx, newMovement)
}

func (u *Movement) handlePaid(ctx context.Context, tx *gorm.DB, existingMovement, newMovement domain.Movement) error {
	return u.handlePaidMovementUpdate(ctx, tx, existingMovement, newMovement)
}
