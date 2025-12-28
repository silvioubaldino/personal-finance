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

// UpdateAllNext atualiza a ocorrência atual e propaga as mudanças para todas as futuras.
// A recorrência original é encerrada no mês anterior (T-1) e uma nova recorrência
// é criada com os NOVOS dados iniciando no mês seguinte (T+1).
func (u *Movement) UpdateAllNext(ctx context.Context, id uuid.UUID, newMovement domain.Movement) (domain.Movement, error) {
	if err := u.validateUpdateAllNextInput(ctx, newMovement); err != nil {
		return domain.Movement{}, err
	}

	var result domain.Movement
	err := u.txManager.WithTransaction(ctx, func(tx *gorm.DB) error {
		var txErr error
		result, txErr = u.executeUpdateAllNext(ctx, tx, id, newMovement)
		return txErr
	})
	if err != nil {
		return domain.Movement{}, err
	}

	return result, nil
}

func (u *Movement) validateUpdateAllNextInput(ctx context.Context, newMovement domain.Movement) error {
	if newMovement.Date == nil {
		return fmt.Errorf("movement date is required")
	}

	if newMovement.RecurrentID == nil {
		return fmt.Errorf("recurrent_id is required for UpdateAllNext")
	}

	return u.validateSubCategory(ctx, newMovement.SubCategoryID, newMovement.CategoryID)
}

func (u *Movement) executeUpdateAllNext(ctx context.Context, tx *gorm.DB, id uuid.UUID, newMovement domain.Movement) (domain.Movement, error) {
	existingMovement, isVirtual, err := u.resolveMovementForAllNext(ctx, id, newMovement)
	if err != nil {
		return domain.Movement{}, err
	}

	if existingMovement.IsCreditCardMovement() {
		return domain.Movement{}, fmt.Errorf("credit card movements cannot be recurrent")
	}

	if err := u.splitRecurrentSeriesWithNewData(ctx, tx, newMovement); err != nil {
		return domain.Movement{}, err
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

func (u *Movement) resolveMovementForAllNext(ctx context.Context, id uuid.UUID, newMovement domain.Movement) (domain.Movement, bool, error) {
	existingMovement, err := u.movementRepo.FindByID(ctx, id)
	if err == nil {
		inputMovement := domain.Movement{ID: &id, RecurrentID: newMovement.RecurrentID}
		return existingMovement, inputMovement.IsVirtualMovement(), nil
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

// splitRecurrentSeriesWithNewData encerra a recorrência original em T-1 e cria
// uma nova recorrência com os NOVOS dados iniciando em T+1.
// Esta é a diferença principal em relação ao UpdateOne: a nova série herda os novos valores.
func (u *Movement) splitRecurrentSeriesWithNewData(ctx context.Context, tx *gorm.DB, newMovement domain.Movement) error {
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

		// Deleta a recorrência atual e cria nova começando em T+1 (com novos dados)
		if err := u.recurrentRepo.Delete(ctx, tx, *newMovement.RecurrentID); err != nil {
			return fmt.Errorf("error deleting recurrent: %w", err)
		}

		newRecurrent := u.buildNewRecurrentFromMovement(recurrent, newMovement, targetMonth, targetYear)

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

	// Restaura o EndDate original para passar para a nova recorrência
	recurrent.EndDate = originalEndDate

	newRecurrent := u.buildNewRecurrentFromMovement(recurrent, newMovement, targetMonth, targetYear)

	if _, err := u.recurrentRepo.Add(ctx, tx, newRecurrent); err != nil {
		return fmt.Errorf("error creating new recurrent series: %w", err)
	}

	return nil
}

// buildNewRecurrentFromMovement cria uma nova RecurrentMovement com os dados do newMovement.
// Isso garante que a série futura refletirá as alterações solicitadas.
// IMPORTANTE: Preserva o EndDate original para manter o limite da série.
func (u *Movement) buildNewRecurrentFromMovement(
	originalRecurrent domain.RecurrentMovement,
	newMovement domain.Movement,
	targetMonth time.Month,
	targetYear int,
) domain.RecurrentMovement {
	newInitialDate := domain.SetMonthYear(*originalRecurrent.InitialDate, targetMonth+1, targetYear)

	newRecurrent := domain.RecurrentMovement{
		ID:            nil,
		InitialDate:   &newInitialDate,
		EndDate:       originalRecurrent.EndDate, // Preserva o EndDate original
		WalletID:      originalRecurrent.WalletID,
		CategoryID:    originalRecurrent.CategoryID,
		SubCategoryID: originalRecurrent.SubCategoryID,
		TypePayment:   originalRecurrent.TypePayment,
		UserID:        originalRecurrent.UserID,
	}

	if newMovement.Description != "" {
		newRecurrent.Description = newMovement.Description
	} else {
		newRecurrent.Description = originalRecurrent.Description
	}

	if newMovement.Amount != 0 {
		newRecurrent.Amount = newMovement.Amount
	} else {
		newRecurrent.Amount = originalRecurrent.Amount
	}

	if newMovement.WalletID != nil {
		newRecurrent.WalletID = newMovement.WalletID
	}

	if newMovement.CategoryID != nil {
		newRecurrent.CategoryID = newMovement.CategoryID
	}

	if newMovement.SubCategoryID != nil {
		newRecurrent.SubCategoryID = newMovement.SubCategoryID
	}

	if newMovement.TypePayment != "" {
		newRecurrent.TypePayment = newMovement.TypePayment
	}

	return newRecurrent
}
