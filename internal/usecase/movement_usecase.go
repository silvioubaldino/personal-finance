package usecase

import (
	"context"
	"fmt"

	"personal-finance/internal/domain"
	"personal-finance/internal/infrastructure/repository/transaction"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type (
	MovementRepository interface {
		Add(ctx context.Context, tx *gorm.DB, movement domain.Movement) (domain.Movement, error)
	}

	RecurrentRepository interface {
		Add(ctx context.Context, tx *gorm.DB, recurrent domain.RecurrentMovement) (domain.RecurrentMovement, error)
	}

	Movement struct {
		movementRepo  MovementRepository
		recurrentRepo RecurrentRepository
		walletRepo    WalletRepository
		subCategory   SubCategory
		txManager     transaction.Manager
	}
)

func NewMovement(
	movementRepo MovementRepository,
	recurrentRepo RecurrentRepository,
	walletRepo WalletRepository,
	subCategory SubCategory,
	txManager transaction.Manager,
) Movement {
	return Movement{
		movementRepo:  movementRepo,
		recurrentRepo: recurrentRepo,
		walletRepo:    walletRepo,
		subCategory:   subCategory,
		txManager:     txManager,
	}
}

func (u *Movement) isSubCategoryValid(ctx context.Context, subCategoryID, categoryID *uuid.UUID) error {
	if subCategoryID == nil {
		return nil
	}

	isSubCategoryValid, err := u.subCategory.IsSubCategoryBelongsToCategory(ctx, *subCategoryID, *categoryID)
	if err != nil {
		return fmt.Errorf("error when searching subcategory: %w", err)
	}

	if !isSubCategoryValid {
		return fmt.Errorf("subcategory does not belong to the provided category")
	}

	return nil
}

func (u *Movement) Add(ctx context.Context, movement domain.Movement) (domain.Movement, error) {
	err := u.isSubCategoryValid(ctx, movement.SubCategoryID, movement.CategoryID)
	if err != nil {
		return domain.Movement{}, err
	}

	var result domain.Movement

	err = u.txManager.WithTransaction(ctx, func(tx *gorm.DB) error {
		if movement.IsRecurrent && movement.RecurrentID == nil {
			recurrent := domain.ToRecurrentMovement(movement)

			createdRecurrent, err := u.recurrentRepo.Add(ctx, tx, recurrent)
			if err != nil {
				return fmt.Errorf("error when creating recurrence: %w", err)
			}

			movement.RecurrentID = createdRecurrent.ID
		}

		createdMovement, err := u.movementRepo.Add(ctx, tx, movement)
		if err != nil {
			return fmt.Errorf("error when creating movement: %w", err)
		}

		if movement.IsPaid {
			wallet, err := u.walletRepo.FindByID(ctx, movement.WalletID)
			if err != nil {
				return fmt.Errorf("error when searching wallet: %w", err)
			}

			wallet.Balance += movement.Amount
			_, err = u.walletRepo.AddConsistent(ctx, tx, wallet)
			if err != nil {
				return fmt.Errorf("error when updating wallet: %w", err)
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
