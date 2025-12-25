package usecase

import (
	"context"
	"errors"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/infrastructure/repository"

	"github.com/google/uuid"
)

type resolvedTarget struct {
	movement        domain.Movement
	recurrent       *domain.RecurrentMovement
	isVirtual       bool
	isRecurrent     bool
	isCreditCard    bool
	targetMonthYear time.Time
}

func (u *Movement) resolveTarget(ctx context.Context, id uuid.UUID, targetDate *time.Time) (resolvedTarget, error) {
	result := resolvedTarget{}

	movement, err := u.movementRepo.FindByID(ctx, id)
	if err != nil {
		if !errors.Is(err, repository.ErrMovementNotFound) {
			return result, err
		}

		recurrent, err := u.recurrentRepo.FindByID(ctx, id)
		if err != nil {
			return result, err
		}

		if targetDate == nil {
			return result, ErrDateRequired
		}

		virtualMov := domain.FromRecurrentMovement(recurrent, *targetDate)
		virtualMov.ID = recurrent.ID

		result.movement = virtualMov
		result.recurrent = &recurrent
		result.isVirtual = true
		result.isRecurrent = true
		result.isCreditCard = virtualMov.IsCreditCardMovement()
		result.targetMonthYear = *targetDate
		return result, nil
	}

	result.movement = movement
	result.isVirtual = false
	result.isCreditCard = movement.IsCreditCardMovement()

	if movement.RecurrentID != nil {
		recurrent, err := u.recurrentRepo.FindByID(ctx, *movement.RecurrentID)
		if err != nil && !errors.Is(err, repository.ErrRecurrentMovementNotFound) {
			return result, err
		}
		if err == nil {
			result.recurrent = &recurrent
			result.isRecurrent = true
		}
	}

	if movement.Date != nil {
		result.targetMonthYear = *movement.Date
	} else if targetDate != nil {
		result.targetMonthYear = *targetDate
	}

	return result, nil
}

func (u *Movement) splitRecurrenceForUpdateOne(
	ctx context.Context,
	recurrent domain.RecurrentMovement,
	targetDate time.Time,
) (*domain.RecurrentMovement, error) {
	endDate := domain.SetMonthYearClamped(*recurrent.InitialDate, targetDate.Month()-1, targetDate.Year())
	recurrent.EndDate = &endDate

	if recurrent.EndDate.Before(*recurrent.InitialDate) {
		recurrent.EndDate = recurrent.InitialDate
	}

	newRecurrent := recurrent
	newRecurrent.ID = nil
	newRecurrent.EndDate = nil
	newInitialDate := domain.SetMonthYearClamped(*recurrent.InitialDate, targetDate.Month()+1, targetDate.Year())
	newRecurrent.InitialDate = &newInitialDate

	return &newRecurrent, nil
}

func (u *Movement) splitRecurrenceForUpdateAllNext(
	ctx context.Context,
	recurrent domain.RecurrentMovement,
	targetDate time.Time,
	newValues domain.RecurrentMovement,
) (*domain.RecurrentMovement, error) {
	endDate := domain.SetMonthYearClamped(*recurrent.InitialDate, targetDate.Month()-1, targetDate.Year())
	recurrent.EndDate = &endDate

	if recurrent.EndDate.Before(*recurrent.InitialDate) {
		recurrent.EndDate = recurrent.InitialDate
	}

	newRecurrent := newValues
	newRecurrent.ID = nil
	newRecurrent.EndDate = nil
	newInitialDate := domain.SetMonthYearClamped(*recurrent.InitialDate, targetDate.Month()+1, targetDate.Year())
	newRecurrent.InitialDate = &newInitialDate

	return &newRecurrent, nil
}

func (u *Movement) splitRecurrenceForDeleteOne(
	ctx context.Context,
	recurrent domain.RecurrentMovement,
	targetDate time.Time,
) (*domain.RecurrentMovement, error) {
	endDate := domain.SetMonthYearClamped(*recurrent.InitialDate, targetDate.Month()-1, targetDate.Year())
	recurrent.EndDate = &endDate

	if recurrent.EndDate.Before(*recurrent.InitialDate) {
		recurrent.EndDate = recurrent.InitialDate
	}

	newRecurrent := recurrent
	newRecurrent.ID = nil
	newRecurrent.EndDate = nil
	newInitialDate := domain.SetMonthYearClamped(*recurrent.InitialDate, targetDate.Month()+1, targetDate.Year())
	newRecurrent.InitialDate = &newInitialDate

	return &newRecurrent, nil
}

func (u *Movement) endRecurrence(
	recurrent domain.RecurrentMovement,
	targetDate time.Time,
) domain.RecurrentMovement {
	endDate := domain.SetMonthYearClamped(*recurrent.InitialDate, targetDate.Month()-1, targetDate.Year())
	recurrent.EndDate = &endDate

	if recurrent.EndDate.Before(*recurrent.InitialDate) {
		recurrent.EndDate = recurrent.InitialDate
	}

	return recurrent
}
