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
	"personal-finance/internal/domain/transaction/service"
	"personal-finance/internal/model"
)

type Movement interface {
	Add(ctx context.Context, transaction model.Movement, userID string) (model.Movement, error)
	AddSimple(ctx context.Context, transaction model.Movement, userID string) (model.Movement, error)
	FindByID(ctx context.Context, id uuid.UUID, userID string) (model.Movement, error)
	FindByPeriod(ctx context.Context, period model.Period, userID string) ([]model.Movement, error)
	Pay(ctx context.Context, id uuid.UUID, date time.Time, userID string) (model.Movement, error)
	RevertPay(ctx context.Context, id uuid.UUID, userID string) (model.Movement, error)
	Update(ctx context.Context, id uuid.UUID, transaction model.Movement, userID string) (model.Movement, error)
	UpdateAllNext(ctx context.Context, id *uuid.UUID, newMovement model.Movement) (model.Movement, error)
	Delete(ctx context.Context, id uuid.UUID, userID string) error
	DeleteAllNext(ctx context.Context, id uuid.UUID, date time.Time) error
}

type movement struct {
	repo            repository.Repository
	subCategoryRepo subCategoryRepository.Repository
	transactionSvc  service.Transaction
	recurrentRepo   recurrentRepository.RecurrentRepository
}

func NewMovementService(
	repo repository.Repository,
	subCategoryRepo subCategoryRepository.Repository,
	transactionSvc service.Transaction,
	recurrentRepo recurrentRepository.RecurrentRepository,
) Movement {
	return movement{
		repo:            repo,
		subCategoryRepo: subCategoryRepo,
		transactionSvc:  transactionSvc,
		recurrentRepo:   recurrentRepo,
	}
}

func (s movement) Add(ctx context.Context, movement model.Movement, userID string) (model.Movement, error) {
	if movement.TransactionID == nil {
		if movement.StatusID == model.TransactionStatusPlannedID {
			movement, err := s.repo.Add(ctx, movement, userID)
			if err != nil {
				return model.Movement{}, fmt.Errorf("error to add transactions: %w", err)
			}
			return movement, nil
		}

		if movement.StatusID == model.TransactionStatusPaidID {
			transaction, err := s.transactionSvc.AddDirectDoneTransaction(ctx, movement, userID)
			if err != nil {
				return model.Movement{}, fmt.Errorf("error to add transactions: %w", err)
			}
			return *transaction.Estimate, nil
		}
	}

	if movement.StatusID == model.TransactionStatusPlannedID {
		return model.Movement{}, errors.New("planned transactions must not have transactionID")
	}

	movement, err := s.repo.AddUpdatingWallet(ctx, nil, movement, userID)
	if err != nil {
		return model.Movement{}, fmt.Errorf("error to add transactions: %w", err)
	}
	return movement, nil
}

func (s movement) AddSimple(ctx context.Context, movement model.Movement, userID string) (model.Movement, error) {
	if movement.SubCategoryID != nil {
		sub, err := s.subCategoryRepo.FindByID(ctx, *movement.SubCategoryID, userID)
		if err != nil {
			return model.Movement{}, fmt.Errorf("error to find subcategory: %w", err)
		}

		if *sub.CategoryID != *movement.CategoryID {
			return model.Movement{}, errors.New("subcategory does not belong to the category")
		}
	}

	if movement.IsPaid {
		movement.StatusID = 1
		movement, err := s.repo.AddUpdatingWallet(ctx, nil, movement, userID)
		if err != nil {
			return model.Movement{}, fmt.Errorf("error to add transactions: %w", err)
		}
		return movement, nil
	}

	movement.StatusID = 2
	CreatedMovement, err := s.repo.Add(ctx, movement, userID)
	if err != nil {
		return model.Movement{}, fmt.Errorf("error to add transactions: %w", err)
	}
	return CreatedMovement, nil
}

func (s movement) FindByID(ctx context.Context, id uuid.UUID, userID string) (model.Movement, error) {
	result, err := s.repo.FindByID(ctx, id, userID)
	if err != nil {
		return model.Movement{}, fmt.Errorf("error to find transactions: %w", err)
	}
	return result, nil
}

func (s movement) FindByPeriod(ctx context.Context, period model.Period, userID string) ([]model.Movement, error) {
	result, err := s.repo.FindByPeriod(ctx, period, userID)
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

func (s movement) Pay(ctx context.Context, id uuid.UUID, date time.Time, userID string) (model.Movement, error) {
	movement, err := s.repo.FindByID(ctx, id, userID)
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
		addSimple, err := s.AddSimple(ctx, mov, userID)
		if err != nil {
			return model.Movement{}, err
		}
		return addSimple, nil
	}

	if movement.StatusID == model.TransactionStatusPaidID || movement.IsPaid {
		return model.Movement{}, errors.New("transaction already paid")
	}
	movement.StatusID = model.TransactionStatusPaidID
	movement.IsPaid = true

	result, err := s.repo.UpdateIsPaid(ctx, id, movement, userID)
	if err != nil {
		return model.Movement{}, fmt.Errorf("error updating transactions: %w", err)
	}
	return result, nil
}

func (s movement) RevertPay(ctx context.Context, id uuid.UUID, userID string) (model.Movement, error) {
	movement, err := s.repo.FindByID(ctx, id, userID)
	if err != nil {
		return model.Movement{}, fmt.Errorf("error finding transactions: %w", err)
	}

	if movement.StatusID == model.TransactionStatusPlannedID || !movement.IsPaid {
		return model.Movement{}, errors.New("transaction is not paid")
	}

	movement.StatusID = model.TransactionStatusPlannedID
	movement.IsPaid = false

	result, err := s.repo.UpdateIsPaid(ctx, id, movement, userID)
	if err != nil {
		return model.Movement{}, fmt.Errorf("error updating transactions: %w", err)
	}
	return result, nil
}

func (s movement) Update(ctx context.Context, id uuid.UUID, newMovement model.Movement, userID string) (model.Movement, error) {
	movementFound, err := s.repo.FindByID(ctx, id, userID)
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
		addSimple, err := s.AddSimple(ctx, mov, userID)
		if err != nil {
			return model.Movement{}, err
		}
		return addSimple, nil
	}

	result, err := s.repo.Update(ctx, newMovement, movementFound, userID)
	if err != nil {
		return model.Movement{}, fmt.Errorf("error updating transactions: %w", err)
	}
	return result, nil
}

func (s movement) UpdateAllNext(ctx context.Context, id *uuid.UUID, newMovement model.Movement) (model.Movement, error) {
	recurrentMovement, err := s.recurrentRepo.Update(ctx, id, model.ToRecurrentMovement(newMovement))
	if err != nil {
		return model.Movement{}, err
	}
	return model.FromRecurrentMovement(recurrentMovement, *newMovement.Date), nil
}

func (s movement) Delete(ctx context.Context, id uuid.UUID, userID string) error {
	err := s.repo.Delete(ctx, id, userID)
	if err != nil {
		return fmt.Errorf("error deleting transactions: %w", err)
	}
	return nil
}

func (s movement) DeleteAllNext(ctx context.Context, id uuid.UUID, date time.Time) error {
	userID := ctx.Value("user_id").(string)
	movementFound, err := s.repo.FindByID(ctx, id, userID)
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

		endDate := time.Date(
			date.Year(),
			date.Month(),
			recurrent.InitialDate.Day(),
			recurrent.InitialDate.Hour(),
			recurrent.InitialDate.Minute(),
			recurrent.InitialDate.Second(),
			recurrent.InitialDate.Nanosecond(),
			recurrent.InitialDate.Location(),
		)
		recurrent.EndDate = &endDate

		err = s.repo.DeleteAllNext(ctx, recurrent.ID, movementFound, recurrent)
		if err != nil {
			return err
		}
		return nil
	}

	if movementFound.ID == nil {
		return fmt.Errorf("movement not found")
	}

	err = s.repo.Delete(ctx, *movementFound.ID, userID)
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
