package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"personal-finance/internal/domain/movement/repository"
	recurrentRepository "personal-finance/internal/domain/recurrentmovement/repository"
	subCategoryRepository "personal-finance/internal/domain/subcategory/repository"
	"personal-finance/internal/model"
)

type Movement interface {
	AddSimple(ctx context.Context, transaction model.Movement) (model.Movement, error)
	FindByPeriod(ctx context.Context, period model.Period) ([]model.Movement, error)
	Pay(ctx context.Context, id uuid.UUID, date time.Time) (model.Movement, error)
	RevertPay(ctx context.Context, id uuid.UUID) (model.Movement, error)
	Update(ctx context.Context, id uuid.UUID, transaction model.Movement) (model.Movement, error)
	UpdateAllNext(ctx context.Context, id *uuid.UUID, newMovement model.Movement) (model.Movement, error)
	Delete(ctx context.Context, id uuid.UUID, date time.Time) error
	DeleteAllNext(ctx context.Context, id uuid.UUID, date time.Time) error
}

type movement struct {
	repo            repository.Repository
	subCategoryRepo subCategoryRepository.Repository
	recurrentRepo   recurrentRepository.RecurrentRepository
}

func NewMovementService(
	repo repository.Repository,
	subCategoryRepo subCategoryRepository.Repository,
	recurrentRepo recurrentRepository.RecurrentRepository,
) Movement {
	return movement{
		repo:            repo,
		subCategoryRepo: subCategoryRepo,
		recurrentRepo:   recurrentRepo,
	}
}

func (s movement) AddSimple(ctx context.Context, movement model.Movement) (model.Movement, error) {
	if movement.SubCategoryID != nil {
		sub, err := s.subCategoryRepo.FindByID(ctx, *movement.SubCategoryID)
		if err != nil {
			return model.Movement{}, fmt.Errorf("error to find subcategory: %w", err)
		}

		if *sub.CategoryID != *movement.CategoryID {
			return model.Movement{}, errors.New("subcategory does not belong to the category")
		}
	}

	if movement.IsPaid {
		movement, err := s.repo.AddUpdatingWallet(ctx, nil, movement)
		if err != nil {
			return model.Movement{}, fmt.Errorf("error to add transactions: %w", err)
		}
		return movement, nil
	}

	CreatedMovement, err := s.repo.Add(ctx, movement)
	if err != nil {
		return model.Movement{}, fmt.Errorf("error to add transactions: %w", err)
	}
	return CreatedMovement, nil
}

func (s movement) FindByPeriod(ctx context.Context, period model.Period) ([]model.Movement, error) {
	result, err := s.repo.FindByPeriod(ctx, period)
	if err != nil {
		return []model.Movement{}, fmt.Errorf("error to find transactions: %w", err)
	}

	recurrents, err := s.recurrentRepo.FindByMonth(ctx, period.To)
	if err != nil {
		return []model.Movement{}, fmt.Errorf("error to find recurrents: %w", err)
	}

	recurrentMap := make(map[uuid.UUID]struct{}, len(recurrents))
	for i, mov := range result {
		if mov.RecurrentID != nil {
			result[i].IsRecurrent = true
			recurrentMap[*mov.RecurrentID] = struct{}{}
		}
	}

	for _, recurrent := range recurrents {
		if _, ok := recurrentMap[*recurrent.ID]; !ok {
			mov := model.FromRecurrentMovement(recurrent, period.To)

			mov.ID = mov.RecurrentID
			result = append(result, mov)
		}
	}

	return result, nil
}

func (s movement) Pay(ctx context.Context, id uuid.UUID, date time.Time) (model.Movement, error) {
	movement, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if !errors.Is(err, model.ErrNotFound) {
			return model.Movement{}, err
		}

		recurrent, err := s.recurrentRepo.FindByID(ctx, id)
		if err != nil {
			return model.Movement{}, fmt.Errorf("error finding transactions: %w", err)
		}

		if date.IsZero() {
			return model.Movement{}, errors.New("date must be informed")
		}
		mov := model.FromRecurrentMovement(recurrent, date)
		mov.IsPaid = true
		addSimple, err := s.AddSimple(ctx, mov)
		if err != nil {
			return model.Movement{}, err
		}
		return addSimple, nil
	}

	if movement.IsPaid {
		return model.Movement{}, errors.New("transaction already paid")
	}
	movement.IsPaid = true

	result, err := s.repo.UpdateIsPaid(ctx, id, movement)
	if err != nil {
		return model.Movement{}, fmt.Errorf("error updating transactions: %w", err)
	}
	return result, nil
}

func (s movement) RevertPay(ctx context.Context, id uuid.UUID) (model.Movement, error) {
	movement, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return model.Movement{}, fmt.Errorf("error finding transactions: %w", err)
	}

	if !movement.IsPaid {
		return model.Movement{}, errors.New("transaction is not paid")
	}

	movement.IsPaid = false

	result, err := s.repo.UpdateIsPaid(ctx, id, movement)
	if err != nil {
		return model.Movement{}, fmt.Errorf("error updating transactions: %w", err)
	}
	return result, nil
}

func (s movement) Update(ctx context.Context, id uuid.UUID, newMovement model.Movement) (model.Movement, error) {
	movementFound, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if !errors.Is(err, model.ErrNotFound) {
			return model.Movement{}, err
		}

		recurrent, err := s.recurrentRepo.FindByID(ctx, id)
		if err != nil {
			return model.Movement{}, fmt.Errorf("error finding transactions: %w", err)
		}

		mov := model.FromRecurrentMovement(recurrent, *newMovement.Date)
		mov = update(newMovement, mov)
		mov.IsRecurrent = true
		mov.RecurrentID = recurrent.ID
		addSimple, err := s.AddSimple(ctx, mov)
		if err != nil {
			return model.Movement{}, err
		}
		return addSimple, nil
	}

	result, err := s.repo.Update(ctx, newMovement, movementFound)
	if err != nil {
		return model.Movement{}, fmt.Errorf("error updating transactions: %w", err)
	}
	return result, nil
}

func (s movement) UpdateAllNext(ctx context.Context, id *uuid.UUID, newMovement model.Movement) (model.Movement, error) {
	movementFound, err := s.repo.FindByID(ctx, *id)
	if err != nil {
		if !errors.Is(err, model.ErrNotFound) {
			return model.Movement{}, err
		}
	}
	var recurrent model.RecurrentMovement
	if movementFound.ID != nil {
		recurrent, err = s.recurrentRepo.FindByID(ctx, *movementFound.RecurrentID)
		if err != nil {
			if !errors.Is(err, model.ErrNotFound) {
				return model.Movement{}, fmt.Errorf("error finding transactions: %w", err)
			}
		}
	} else {
		recurrent, err = s.recurrentRepo.FindByID(ctx, *id)
		if err != nil {
			if !errors.Is(err, model.ErrNotFound) {
				return model.Movement{}, fmt.Errorf("error finding transactions: %w", err)
			}
		}
	}

	if recurrent.ID != nil {
		_, err = s.repo.UpdateAllNextRecurrent(ctx, newMovement, movementFound, recurrent)
		if err != nil {
			return model.Movement{}, err
		}
		return newMovement, nil
	}

	if movementFound.ID == nil {
		return model.Movement{}, fmt.Errorf("movement not found")
	}

	_, err = s.repo.Update(ctx, newMovement, movementFound)
	if err != nil {
		return model.Movement{}, fmt.Errorf("error updating transactions: %w", err)
	}
	return newMovement, nil
}

func (s movement) Delete(ctx context.Context, id uuid.UUID, date time.Time) error {
	movementFound, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if !errors.Is(err, model.ErrNotFound) {
			return err
		}
	}

	var recurrent model.RecurrentMovement
	if movementFound.ID != nil && movementFound.RecurrentID != nil {
		recurrent, err = s.recurrentRepo.FindByID(ctx, *movementFound.RecurrentID)
		if err != nil {
			if !errors.Is(err, model.ErrNotFound) {
				return fmt.Errorf("error finding transactions: %w", err)
			}
		}
	} else {
		recurrent, err = s.recurrentRepo.FindByID(ctx, id)
		if err != nil {
			if !errors.Is(err, model.ErrNotFound) {
				return fmt.Errorf("error finding transactions: %w", err)
			}
		}
	}

	if recurrent.ID != nil {
		if date.IsZero() {
			if movementFound.ID == nil {
				return fmt.Errorf("date must be informed")
			}
			date = *movementFound.Date
		}

		newRecurrent := recurrent

		endDate := model.SetMonthYear(*recurrent.InitialDate, date.Month(), date.Year())
		recurrent.EndDate = &endDate

		initialDate := model.SetMonthYear(*recurrent.InitialDate, recurrent.EndDate.Month()+1, recurrent.EndDate.Year())
		newRecurrent.InitialDate = &initialDate

		err = s.repo.DeleteOneRecurrent(ctx, recurrent.ID, movementFound, recurrent, newRecurrent)
		if err != nil {
			return err
		}
		return nil
	}

	err = s.repo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("error deleting transactions: %w", err)
	}
	return nil
}

func (s movement) DeleteAllNext(ctx context.Context, id uuid.UUID, date time.Time) error {
	movementFound, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if !errors.Is(err, model.ErrNotFound) {
			return err
		}
	}
	var recurrent model.RecurrentMovement
	if movementFound.ID != nil {
		recurrent, err = s.recurrentRepo.FindByID(ctx, *movementFound.RecurrentID)
		if err != nil {
			if !errors.Is(err, model.ErrNotFound) {
				return fmt.Errorf("error finding transactions: %w", err)
			}
		}
	} else {
		recurrent, err = s.recurrentRepo.FindByID(ctx, id)
		if err != nil {
			if !errors.Is(err, model.ErrNotFound) {
				return fmt.Errorf("error finding transactions: %w", err)
			}
		}
	}

	if recurrent.ID != nil {
		if date.IsZero() {
			if movementFound.ID == nil {
				return fmt.Errorf("date must be informed")
			}
			date = *movementFound.Date
		}

		endDate := model.SetMonthYear(*recurrent.InitialDate, date.Month(), date.Year())
		recurrent.EndDate = &endDate

		err = s.repo.DeleteAllNextRecurrent(ctx, recurrent.ID, movementFound, recurrent)
		if err != nil {
			return err
		}
		return nil
	}

	if movementFound.ID == nil {
		return fmt.Errorf("movement not found")
	}

	err = s.repo.Delete(ctx, *movementFound.ID)
	if err != nil {
		return fmt.Errorf("error deleting transactions: %w", err)
	}
	return nil
}

func update(newMovement, movementFound model.Movement) model.Movement {
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
	if newMovement.TypePaymentID != 0 && newMovement.TypePaymentID != movementFound.TypePaymentID {
		movementFound.TypePaymentID = newMovement.TypePaymentID
	}
	if newMovement.CategoryID != nil && *newMovement.CategoryID != *movementFound.CategoryID {
		movementFound.CategoryID = newMovement.CategoryID
	}
	if newMovement.SubCategoryID != nil && *newMovement.SubCategoryID != *movementFound.SubCategoryID {
		movementFound.SubCategoryID = newMovement.SubCategoryID
	}
	return movementFound
}
