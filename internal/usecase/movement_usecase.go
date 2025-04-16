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
		return fmt.Errorf("erro ao buscar subcategoria: %w", err)
	}

	if !isSubCategoryValid {
		return fmt.Errorf("subcategoria não pertence à categoria informada")
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
				return fmt.Errorf("erro ao criar recorrência: %w", err)
			}

			movement.RecurrentID = createdRecurrent.ID
		}

		createdMovement, err := u.movementRepo.Add(ctx, tx, movement)
		if err != nil {
			return fmt.Errorf("erro ao criar movimento: %w", err)
		}

		if movement.IsPaid {
			wallet, err := u.walletRepo.FindByID(ctx, movement.WalletID)
			if err != nil {
				return fmt.Errorf("erro ao buscar carteira: %w", err)
			}

			wallet.Balance += movement.Amount
			_, err = u.walletRepo.AddConsistent(ctx, tx, wallet)
			if err != nil {
				return fmt.Errorf("erro ao atualizar carteira: %w", err)
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
