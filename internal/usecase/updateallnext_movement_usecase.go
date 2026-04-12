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
		return domain.Movement{}, fmt.Errorf("movement date is required")
	}

	err := u.validateSubCategory(ctx, newMovement.SubCategoryID, newMovement.CategoryID)
	if err != nil {
		return domain.Movement{}, err
	}

	var result domain.Movement
	err = u.txManager.WithTransaction(ctx, func(tx *gorm.DB) error {
		existingMovement, findErr := u.movementRepo.FindByID(ctx, id)
		if findErr != nil && !errors.Is(findErr, repository.ErrMovementNotFound) {
			return findErr
		}

		recurrent, recErr := u.resolveRecurrent(ctx, &existingMovement, id)
		if recErr != nil {
			return recErr
		}

		if recurrent.ID != nil {
			return u.updateAllNextRecurrent(ctx, tx, &existingMovement, &recurrent, newMovement, &result)
		}

		if existingMovement.ID == nil {
			return fmt.Errorf("movement not found")
		}

		return u.updateSingleMovement(ctx, tx, id, &existingMovement, newMovement, &result)
	})
	if err != nil {
		return domain.Movement{}, err
	}

	return result, nil
}

func (u *Movement) resolveRecurrent(ctx context.Context, existingMovement *domain.Movement, id uuid.UUID) (domain.RecurrentMovement, error) {
	if existingMovement.ID != nil && existingMovement.RecurrentID != nil {
		recurrent, err := u.recurrentRepo.FindByID(ctx, *existingMovement.RecurrentID)
		if err != nil && !errors.Is(err, repository.ErrRecurrentMovementNotFound) {
			return domain.RecurrentMovement{}, fmt.Errorf("error finding recurrent: %w", err)
		}
		return recurrent, nil
	}

	if existingMovement.ID == nil {
		recurrent, err := u.recurrentRepo.FindByID(ctx, id)
		if err != nil {
			return domain.RecurrentMovement{}, fmt.Errorf("error finding recurrent: %w", err)
		}
		return recurrent, nil
	}

	return domain.RecurrentMovement{}, nil
}

func (u *Movement) updateAllNextRecurrent(
	ctx context.Context,
	tx *gorm.DB,
	existingMovement *domain.Movement,
	recurrent *domain.RecurrentMovement,
	newMovement domain.Movement,
	result *domain.Movement,
) error {
	endDate := domain.SetMonthYear(*recurrent.InitialDate, newMovement.Date.Month(), newMovement.Date.Year())

	updatedRecurrent := *recurrent
	updatedRecurrent.EndDate = &endDate
	if updatedRecurrent.EndDate.Before(*updatedRecurrent.InitialDate) {
		updatedRecurrent.EndDate = updatedRecurrent.InitialDate
	}

	_, err := u.recurrentRepo.Update(ctx, tx, recurrent.ID, updatedRecurrent)
	if err != nil {
		return fmt.Errorf("error updating recurrent end date: %w", err)
	}

	newRecurrent := domain.ToRecurrentMovement(newMovement)
	newInitialDate := domain.SetMonthYear(*recurrent.InitialDate, newMovement.Date.Month()+1, newMovement.Date.Year())
	newRecurrent.InitialDate = &newInitialDate
	newRecurrent.EndDate = recurrent.EndDate

	createdRecurrent, err := u.recurrentRepo.Add(ctx, tx, newRecurrent)
	if err != nil {
		return fmt.Errorf("error creating new recurrent: %w", err)
	}

	if existingMovement.ID != nil {
		if existingMovement.IsPaid {
			if err := u.handlePaid(ctx, tx, *existingMovement, newMovement); err != nil {
				return err
			}
		}

		newMovement.RecurrentID = createdRecurrent.ID
		updated, err := u.movementRepo.Update(ctx, tx, *existingMovement.ID, newMovement)
		if err != nil {
			return fmt.Errorf("error updating movement: %w", err)
		}
		*result = updated
	} else {
		*result = newMovement
	}

	return nil
}

func (u *Movement) updateSingleMovement(
	ctx context.Context,
	tx *gorm.DB,
	id uuid.UUID,
	existingMovement *domain.Movement,
	newMovement domain.Movement,
	result *domain.Movement,
) error {
	if existingMovement.IsPaid {
		if err := u.handlePaid(ctx, tx, *existingMovement, newMovement); err != nil {
			return err
		}
	}

	updated, err := u.movementRepo.Update(ctx, tx, id, newMovement)
	if err != nil {
		return fmt.Errorf("error updating movement: %w", err)
	}
	*result = updated
	return nil
}
