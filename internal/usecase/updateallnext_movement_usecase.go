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

// UpdateAllNext atualiza a recorrência (fechando a recorrência atual até o mês anterior à nova data)
// e cria uma nova recorrência iniciando no mês da nova data, aplicando os campos atualizados.
// Em seguida, atualiza a movement existente (ajustando wallets se estiver paga) ou cria a movement se ela não existir.
func (u *Movement) UpdateAllNext(ctx context.Context, id uuid.UUID, newMovement domain.Movement) (domain.Movement, error) {
	if newMovement.Date == nil {
		return domain.Movement{}, fmt.Errorf("movement date is required")
	}

	if err := u.validateSubCategory(ctx, newMovement.SubCategoryID, newMovement.CategoryID); err != nil {
		return domain.Movement{}, err
	}

	var result domain.Movement

	err := u.txManager.WithTransaction(ctx, func(tx *gorm.DB) error {
		existingMovement, findErr := u.movementRepo.FindByID(ctx, id)
		movementExists := findErr == nil
		if findErr != nil && !errors.Is(findErr, repository.ErrMovementNotFound) {
			return findErr
		}

		var recurrent domain.RecurrentMovement
		var recErr error
		if movementExists {
			if existingMovement.RecurrentID == nil {
				recurrent, recErr = u.recurrentRepo.FindByID(ctx, id)
			} else {
				recurrent, recErr = u.recurrentRepo.FindByID(ctx, *existingMovement.RecurrentID)
			}
		} else {
			recurrent, recErr = u.recurrentRepo.FindByID(ctx, id)
		}
		if recErr != nil {
			return recErr
		}

		createdRecurrent, err := u.handleRecurrentAllNext(ctx, tx, recurrent, newMovement)
		if err != nil {
			return err
		}

		if movementExists {
			updatedMovement := update(newMovement, existingMovement)

			if existingMovement.IsCreditCardMovement() {
				if err := u.handleCreditCardMovementUpdate(ctx, tx, &existingMovement, &updatedMovement); err != nil {
					return err
				}
			}

			if existingMovement.IsPaid {
				if err := u.handlePaid(ctx, tx, existingMovement, updatedMovement); err != nil {
					return err
				}
			}

			updatedMovement.IsRecurrent = true
			updatedMovement.RecurrentID = createdRecurrent.ID

			res, err := u.movementRepo.Update(ctx, tx, id, updatedMovement)
			if err != nil {
				return err
			}
			result = res
			return nil
		}

		base := domain.FromRecurrentMovement(createdRecurrent, *newMovement.Date)
		movementToCreate := update(newMovement, base)

		if movementToCreate.IsCreditCardMovement() {
			if err := u.getInvoice(ctx, tx, &movementToCreate); err != nil {
				return err
			}
		}

		createdMovement, err := u.movementRepo.Add(ctx, tx, movementToCreate)
		if err != nil {
			return err
		}

		if createdMovement.IsPaid && !createdMovement.IsCreditCardMovement() {
			if err := u.updateWalletBalance(ctx, tx, createdMovement.WalletID, createdMovement.Amount); err != nil {
				return err
			}
		}

		result = createdMovement
		return nil
	})
	if err != nil {
		return domain.Movement{}, err
	}

	return result, nil
}

// handleRecurrentAllNext fecha a recorrência atual e cria uma nova recorrência a partir da data editada,
// sobrepondo os campos do movimento informado. Retorna a nova recorrência criada.
func (u *Movement) handleRecurrentAllNext(
	ctx context.Context,
	tx *gorm.DB,
	recurrent domain.RecurrentMovement,
	newMovement domain.Movement,
) (domain.RecurrentMovement, error) {
	editedMonth := newMovement.Date.Month()
	editedYear := newMovement.Date.Year()

	prevMonth := editedMonth - 1
	prevYear := editedYear
	if prevMonth < time.January {
		prevMonth = time.December
		prevYear = editedYear - 1
	}

	endDate := domain.SetMonthYear(*recurrent.InitialDate, prevMonth, prevYear)

	if _, err := u.recurrentRepo.Update(ctx, tx, recurrent.ID, domain.RecurrentMovement{EndDate: &endDate}); err != nil {
		return domain.RecurrentMovement{}, err
	}

	newInitialDate := domain.SetMonthYear(*recurrent.InitialDate, editedMonth, editedYear)
	newRecurrent := recurrent
	newRecurrent.ID = nil
	newRecurrent.InitialDate = &newInitialDate
	newRecurrent.EndDate = nil

	if newMovement.Description != "" && newMovement.Description != newRecurrent.Description {
		newRecurrent.Description = newMovement.Description
	}
	if newMovement.Amount != 0 && newMovement.Amount != newRecurrent.Amount {
		newRecurrent.Amount = newMovement.Amount
	}
	if newMovement.TypePayment != "" && newMovement.TypePayment != newRecurrent.TypePayment {
		newRecurrent.TypePayment = newMovement.TypePayment
	}
	if newMovement.CategoryID != nil && (newRecurrent.CategoryID == nil || *newMovement.CategoryID != *newRecurrent.CategoryID) {
		newRecurrent.CategoryID = newMovement.CategoryID
	}
	if newMovement.SubCategoryID != nil && (newRecurrent.SubCategoryID == nil || *newMovement.SubCategoryID != *newRecurrent.SubCategoryID) {
		newRecurrent.SubCategoryID = newMovement.SubCategoryID
	}
	if newMovement.WalletID != nil && (newRecurrent.WalletID == nil || *newMovement.WalletID != *newRecurrent.WalletID) {
		newRecurrent.WalletID = newMovement.WalletID
	}

	createdRecurrent, err := u.recurrentRepo.Add(ctx, tx, newRecurrent)
	if err != nil {
		return domain.RecurrentMovement{}, err
	}

	return createdRecurrent, nil
}
