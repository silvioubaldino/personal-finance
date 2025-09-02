package usecase

import (
	"context"
	"errors"
	"personal-finance/internal/domain"
	"personal-finance/internal/infrastructure/repository"
	"personal-finance/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (u *Movement) UpdateOne(ctx context.Context, id uuid.UUID, newMovement domain.Movement) (domain.Movement, error) {
	err := u.isSubCategoryValid(ctx, newMovement.SubCategoryID, newMovement.CategoryID)
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
		// pode nao  existir uma movement criada, apenas um recurrent,
		// nesse caso, vamos criar uma movement a partir do recurrent
		if newMovement.IsRecurrent {
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
			err = u.handlePaid(ctx, tx, id, existingMovement, newMovement)
			if err != nil {
				return err
			}
		}

		result, err = u.movementRepo.UpdateOne(ctx, tx, id, newMovement)
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

	endDate := model.SetMonthYear(*recurrent.InitialDate, newMovement.Date.Month()-1, newMovement.Date.Year())
	recurrent.EndDate = &endDate

	_, err = u.recurrentRepo.Update(ctx, tx, &id, recurrent)
	if err != nil {
		return err
	}

	newInitialDate := model.SetMonthYear(*newRecurrent.InitialDate, newMovement.Date.Month()+1, newMovement.Date.Year())
	newRecurrent.InitialDate = &newInitialDate

	_, err = u.recurrentRepo.Add(ctx, tx, newRecurrent)
	if err != nil {
		return err
	}

	return nil
}

func (u *Movement) handlePaid(ctx context.Context, tx *gorm.DB, id uuid.UUID, existingMovement, newMovement domain.Movement) error {
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
	if newMovement.SubCategoryID != nil && *newMovement.SubCategoryID != *movementFound.SubCategoryID {
		movementFound.SubCategoryID = newMovement.SubCategoryID
	}
	return movementFound
}
