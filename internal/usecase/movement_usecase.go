package usecase

import (
	"context"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/infrastructure/repository/transaction"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type (
	MovementRepository interface {
		Add(ctx context.Context, tx *gorm.DB, movement domain.Movement) (domain.Movement, error)
		FindByPeriod(ctx context.Context, period domain.Period) (domain.MovementList, error)
	}

	RecurrentRepository interface {
		Add(ctx context.Context, tx *gorm.DB, recurrent domain.RecurrentMovement) (domain.RecurrentMovement, error)
		FindByMonth(ctx context.Context, month time.Time) ([]domain.RecurrentMovement, error)
	}

	Movement struct {
		movementRepo    MovementRepository
		recurrentRepo   RecurrentRepository
		walletRepo      WalletRepository
		subCategoryRepo SubCategoryRepository
		txManager       transaction.Manager
	}
)

func NewMovement(
	movementRepo MovementRepository,
	recurrentRepo RecurrentRepository,
	walletRepo WalletRepository,
	subCategoryRepo SubCategoryRepository,
	txManager transaction.Manager,
) Movement {
	return Movement{
		movementRepo:    movementRepo,
		recurrentRepo:   recurrentRepo,
		walletRepo:      walletRepo,
		subCategoryRepo: subCategoryRepo,
		txManager:       txManager,
	}
}

func (u *Movement) isSubCategoryValid(ctx context.Context, subCategoryID, categoryID *uuid.UUID) error {
	if subCategoryID == nil {
		return nil
	}

	isSubCategoryValid, err := u.subCategoryRepo.IsSubCategoryBelongsToCategory(ctx, *subCategoryID, *categoryID)
	if err != nil {
		return err
	}

	if !isSubCategoryValid {
		return domain.WrapInvalidInput(
			domain.New("subcategory does not belong to the provided category"),
			"validate subcategory",
		)
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
		if movement.ShouldCreateRecurrent() {
			recurrent := domain.ToRecurrentMovement(movement)

			createdRecurrent, err := u.recurrentRepo.Add(ctx, tx, recurrent)
			if err != nil {
				return err
			}

			movement.RecurrentID = createdRecurrent.ID
		}

		createdMovement, err := u.movementRepo.Add(ctx, tx, movement)
		if err != nil {
			return err
		}

		if movement.IsPaid {
			wallet, err := u.walletRepo.FindByID(ctx, movement.WalletID)
			if err != nil {
				return err
			}

			if movement.Amount < 0 && wallet.Balance+movement.Amount < 0 {
				return domain.WrapWalletInsufficient(
					domain.New("wallet has insufficient balance"),
					"update wallet balance",
				)
			}

			wallet.Balance += movement.Amount
			err = u.walletRepo.UpdateAmount(ctx, tx, wallet.ID, wallet.Balance)
			if err != nil {
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

func (u *Movement) FindByPeriod(ctx context.Context, period domain.Period) ([]domain.Movement, error) {
	movements, err := u.movementRepo.FindByPeriod(ctx, period)
	if err != nil {
		return []domain.Movement{}, domain.WrapInternalError(err, "error to find transactions")
	}

	recurrents, err := u.recurrentRepo.FindByMonth(ctx, period.To)
	if err != nil {
		return nil, domain.WrapInternalError(err, "error to find recurrents")
	}

	return u.mergeMovementsWithRecurrents(movements, recurrents, period.To), nil
}

func (u *Movement) mergeMovementsWithRecurrents(
	movements domain.MovementList,
	recurrents []domain.RecurrentMovement,
	date time.Time,
) []domain.Movement {
	recurrentMap := make(map[uuid.UUID]struct{}, len(recurrents))
	for i, mov := range movements {
		if mov.RecurrentID != nil {
			movements[i].IsRecurrent = true
			recurrentMap[*mov.RecurrentID] = struct{}{}
		}
	}

	for _, recurrent := range recurrents {
		if _, ok := recurrentMap[*recurrent.ID]; !ok {
			mov := domain.FromRecurrentMovement(recurrent, date)
			mov.ID = mov.RecurrentID
			movements = append(movements, mov)
		}
	}

	return movements
}
