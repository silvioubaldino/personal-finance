package repository

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	recurrentRepository "personal-finance/internal/domain/recurrentmovement/repository"
	"personal-finance/internal/domain/wallet/repository"
	"personal-finance/internal/model"
	"personal-finance/internal/plataform/authentication"
)

type Repository interface {
	Add(ctx context.Context, movement model.Movement) (model.Movement, error)
	AddConsistent(_ context.Context, tx *gorm.DB, movement model.Movement) (model.Movement, error)
	AddUpdatingWallet(ctx context.Context, tx *gorm.DB, movement model.Movement) (model.Movement, error)
	FindByID(_ context.Context, id uuid.UUID) (model.Movement, error)
	FindByPeriod(ctx context.Context, period model.Period) (model.MovementList, error)
	Update(ctx context.Context, newMovement, movementFound model.Movement) (model.Movement, error)
	UpdateAllNextRecurrent(ctx context.Context, newMovement, movementFound model.Movement, recurrent model.RecurrentMovement) (model.Movement, error)
	UpdateIsPaid(ctx context.Context, id uuid.UUID, newMovement model.Movement) (model.Movement, error)
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteOneRecurrent(ctx context.Context, id *uuid.UUID, movement model.Movement, recurrent, newRecurrent model.RecurrentMovement) error
	DeleteAllNextRecurrent(ctx context.Context, id *uuid.UUID, movement model.Movement, recurrent model.RecurrentMovement) error
}

type updateStrategy string

type strategy struct {
	originalMovement model.Movement
	newMovement      model.Movement
	updateStrategies updateStrategy
}

const (
	tableName = "movements"
)

var (
	updateStrategyEmpty           updateStrategy = ""
	updateStrategyDifferentAmount updateStrategy = "different_amount"
	updateStrategyDifferentWallet updateStrategy = "different_wallet"
	updateStrategyPay             updateStrategy = "pay"
	updateStrategyRevertPay       updateStrategy = "revert_pay"
)

type PgRepository struct {
	gorm          *gorm.DB
	walletRepo    repository.Repository
	recurrentRepo recurrentRepository.RecurrentRepository
}

func NewPgRepository(gorm *gorm.DB, walletRepo repository.Repository, recurrentRepo recurrentRepository.RecurrentRepository) Repository {
	return PgRepository{
		gorm:          gorm,
		walletRepo:    walletRepo,
		recurrentRepo: recurrentRepo,
	}
}

func (p PgRepository) Add(ctx context.Context, movement model.Movement) (model.Movement, error) {
	now := time.Now()
	id := uuid.New()
	userID := ctx.Value(authentication.UserID).(string)

	movement.ID = &id
	movement.DateCreate = now
	movement.DateUpdate = now
	movement.UserID = userID

	gormTransactionErr := p.gorm.Transaction(func(tx *gorm.DB) error {
		var recurrent model.RecurrentMovement
		var err error
		if movement.IsRecurrent && movement.RecurrentID == nil {
			recurrent, err = p.recurrentRepo.AddConsistent(ctx, tx, model.ToRecurrentMovement(movement))
			if err != nil {
				return err
			}
			movement.RecurrentID = recurrent.ID
		}

		result := tx.
			Select([]string{
				"id",
				"description",
				"amount",
				"date",
				"user_id",
				"type_payment",
				"date_create",
				"date_update",
				"is_paid",
				"sub_category_id",
				"category_id",
				"wallet_id",
				"recurrent_id",
			}).
			Create(&movement)
		if err := result.Error; err != nil {
			log.Printf("Error: %v", err)
			return err
		}
		return nil
	})
	if gormTransactionErr != nil {
		return model.Movement{}, handleError("repository error", gormTransactionErr)
	}
	return movement, nil
}

func (p PgRepository) FindByID(ctx context.Context, id uuid.UUID) (model.Movement, error) {
	var transaction model.Movement
	userID := ctx.Value(authentication.UserID).(string)
	result := p.gorm.Where("user_id=?", userID).First(&transaction, id)
	if err := result.Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.Movement{}, model.BuildErrNotfound("resource not found")
		}
		return model.Movement{}, handleError("repository error", err)
	}
	return transaction, nil
}

func (p PgRepository) FindByPeriod(ctx context.Context, period model.Period) (model.MovementList, error) {
	var transaction []model.Movement
	userID := ctx.Value(authentication.UserID).(string)

	result := p.buildBaseQuery(userID,
		"left join wallets w on movements.wallet_id = w.id",
		"left join categories c on movements.category_id = c.id",
		"left join sub_categories sc on movements.sub_category_id = sc.id",
	).
		Where("date BETWEEN ? AND ?", period.From, period.To).
		Select([]string{
			"movements.id",
			"movements.description",
			"movements.date",
			"movements.amount",
			"movements.is_paid",
			"movements.recurrent_id",
			"movements.type_payment",
			`w.id as "Wallet__id"`,
			`w.description as "Wallet__description"`,
			`c.id as "Category__id"`,
			`c.description as "Category__description"`,
			`c.is_income as "Category__is_income"`,
			`sc.id as "SubCategory__id"`,
			`sc.description as "SubCategory__description"`,
		}).
		Order("movements.date desc").
		Find(&transaction)
	if err := result.Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []model.Movement{}, model.BuildErrNotfound("resource not found")
		}
		return []model.Movement{}, handleError("repository error", err)
	}
	return transaction, nil
}

func (p PgRepository) Update(ctx context.Context, newMovement, movementFound model.Movement) (model.Movement, error) {
	var err error
	strategy := strategy{movementFound, newMovement, updateStrategyEmpty}

	strategy, err = setNewFields(strategy)
	if err != nil {
		return model.Movement{}, err
	}

	movementFound = strategy.originalMovement

	movementFound.DateUpdate = time.Now()

	gormTransactionErr := p.gorm.Transaction(func(tx *gorm.DB) error {
		movementFound, err = p.update(ctx, tx, movementFound, &strategy)
		return err
	})
	if gormTransactionErr != nil {
		return model.Movement{}, handleError("repository error", gormTransactionErr)
	}

	return movementFound, nil
}

func setNewFields(strategy strategy) (strategy, error) {
	var updated bool
	if strategy.newMovement.Description != "" {
		strategy.originalMovement.Description = strategy.newMovement.Description
		updated = true
	}
	if strategy.newMovement.Date != nil {
		strategy.originalMovement.Date = strategy.newMovement.Date
		updated = true
	}
	if strategy.newMovement.TypePayment != "" {
		strategy.originalMovement.TypePayment = strategy.newMovement.TypePayment
		updated = true
	}
	if strategy.newMovement.CategoryID != nil {
		strategy.originalMovement.CategoryID = strategy.newMovement.CategoryID
		updated = true
	}
	if strategy.newMovement.SubCategoryID != nil {
		strategy.originalMovement.SubCategoryID = strategy.newMovement.SubCategoryID
		updated = true
	}
	if strategy.newMovement.Amount != 0 && strategy.newMovement.Amount != strategy.originalMovement.Amount {
		strategy.updateStrategies = updateStrategyDifferentAmount
		strategy.originalMovement.Amount = strategy.newMovement.Amount
		updated = true
	}
	if strategy.newMovement.WalletID != nil {
		if *strategy.newMovement.WalletID != *strategy.originalMovement.WalletID {
			strategy.updateStrategies = updateStrategyDifferentWallet
			strategy.originalMovement.WalletID = strategy.newMovement.WalletID
			strategy.originalMovement.Amount = strategy.newMovement.Amount
			updated = true
		}
	}
	if strategy.newMovement.RecurrentID != nil {
		strategy.originalMovement.RecurrentID = strategy.newMovement.RecurrentID
		updated = true
	}

	if !updated {
		return strategy, handleError("no changes", errors.New("no changes"))
	}

	return strategy, nil
}

func (p PgRepository) UpdateAllNextRecurrent(
	ctx context.Context,
	newMovement, movementFound model.Movement,
	recurrentFound model.RecurrentMovement,
) (model.Movement, error) {
	gormTransactionErr := p.gorm.Transaction(func(tx *gorm.DB) error {
		endDate := model.SetMonthYear(*recurrentFound.InitialDate, newMovement.Date.Month(), newMovement.Date.Year())
		_, err := p.recurrentRepo.Update(ctx, tx, recurrentFound.ID, model.RecurrentMovement{EndDate: &endDate})
		if err != nil {
			return err
		}

		newRecurrent := model.ToRecurrentMovement(newMovement)
		newInitialDate := model.SetMonthYear(*recurrentFound.InitialDate, newMovement.Date.Month(), newMovement.Date.Year())
		newRecurrent.InitialDate = &newInitialDate
		newRecurrent.EndDate = recurrentFound.EndDate
		newRecurrent = recurrentRepository.SetNewFields(newRecurrent, recurrentFound)
		newRecurrent, err = p.recurrentRepo.AddConsistent(ctx, tx, newRecurrent)
		if err != nil {
			return err
		}

		if movementFound.ID != nil {
			newMovement.RecurrentID = newRecurrent.ID
			strategy := strategy{originalMovement: movementFound, newMovement: newMovement, updateStrategies: updateStrategyEmpty}
			strategy, err = setNewFields(strategy)
			if err != nil {
				return err
			}
			movementFound, err = p.update(ctx, tx, strategy.originalMovement, &strategy)
			if err != nil {
				return err
			}
		}
		return nil
	})

	if gormTransactionErr != nil {
		return model.Movement{}, handleError("repository error", gormTransactionErr)
	}
	return movementFound, nil
}

func (p PgRepository) UpdateIsPaid(ctx context.Context, id uuid.UUID, newMovement model.Movement) (model.Movement, error) {
	movementFound, err := p.FindByID(ctx, id)
	if err != nil {
		return model.Movement{}, err
	}

	strategy := strategy{movementFound, newMovement, updateStrategyEmpty}
	if newMovement.IsPaid != movementFound.IsPaid {
		strategy.updateStrategies = updateStrategyPay
		if movementFound.IsPaid {
			strategy.updateStrategies = updateStrategyRevertPay
		}

		movementFound.IsPaid = newMovement.IsPaid
	}

	movementFound.DateUpdate = time.Now()

	gormTransactionErr := p.gorm.Transaction(func(tx *gorm.DB) error {
		movementFound, err = p.update(ctx, tx, movementFound, &strategy)
		return err
	})
	if gormTransactionErr != nil {
		return model.Movement{}, handleError("repository error", gormTransactionErr)
	}

	return movementFound, nil
}

func (p PgRepository) update(ctx context.Context, tx *gorm.DB, movementFound model.Movement, strategy *strategy) (model.Movement, error) {
	result := tx.
		Select([]string{
			"id",
			"description",
			"amount",
			"date",
			"user_id",
			"type_payment",
			"date_create",
			"date_update",
			"is_paid",
			"sub_category_id",
			"category_id",
			"wallet_id",
			"recurrent_id",
		}).
		Save(&movementFound)
	if err := result.Error; err != nil {
		return model.Movement{}, err
	}
	err := p.updateWallet(ctx, tx, *strategy)
	if err != nil {
		return model.Movement{}, err
	}

	tx = result.Scan(&movementFound)

	return movementFound, nil
}

func (p PgRepository) updateWallet(ctx context.Context, tx *gorm.DB, strategy strategy) error {
	originalWallet, err := p.walletRepo.FindByID(ctx, strategy.originalMovement.WalletID)
	if err != nil {
		return err
	}

	switch strategy.updateStrategies {
	case updateStrategyPay:
		originalWallet.Balance += strategy.newMovement.Amount
		_, err = p.walletRepo.UpdateConsistent(ctx, tx, originalWallet)
		if err != nil {
			return err
		}
		return nil

	case updateStrategyRevertPay:
		originalWallet.Balance -= strategy.newMovement.Amount
		_, err = p.walletRepo.UpdateConsistent(ctx, tx, originalWallet)
		if err != nil {
			return err
		}
		return nil

	case updateStrategyDifferentAmount:
		originalWallet.Balance += strategy.newMovement.Amount - strategy.originalMovement.Amount
		_, err = p.walletRepo.UpdateConsistent(ctx, tx, originalWallet)
		if err != nil {
			return err
		}
		return nil

	case updateStrategyDifferentWallet:
		originalWallet.Balance -= strategy.originalMovement.Amount
		_, err = p.walletRepo.UpdateConsistent(ctx, tx, originalWallet)
		if err != nil {
			return err
		}

		newWallet, err := p.walletRepo.FindByID(ctx, strategy.newMovement.WalletID)
		if err != nil {
			return err
		}
		if strategy.newMovement.Amount == 0 {
			strategy.newMovement.Amount = strategy.originalMovement.Amount
		}
		newWallet.Balance += strategy.newMovement.Amount

		_, err = p.walletRepo.UpdateConsistent(ctx, tx, newWallet)
		if err != nil {
			return err
		}
		return nil
	case "":
		return nil
	default:
		return fmt.Errorf("invalid strategy")
	}
}

func (p PgRepository) Delete(ctx context.Context, id uuid.UUID) error {
	userID := ctx.Value(authentication.UserID).(string)
	movement, err := p.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if !movement.IsPaid {
		if err := p.gorm.Where("user_id=?", userID).Delete(&model.Movement{}, id).Error; err != nil {
			return handleError("repository error", err)
		}
		return nil
	}

	gormTransactionErr := p.gorm.Transaction(func(tx *gorm.DB) error {
		if err := p.gorm.Where("user_id=?", userID).Delete(&model.Movement{}, id).Error; err != nil {
			return handleError("repository error", err)
		}
		err = p.updateWallet(
			ctx,
			tx,
			strategy{
				originalMovement: movement,
				newMovement:      movement,
				updateStrategies: updateStrategyRevertPay,
			})
		if err != nil {
			return err
		}
		return nil
	})
	if gormTransactionErr != nil {
		return handleError("repository error", handleError("repository error", gormTransactionErr))
	}
	return nil
}

func (p PgRepository) DeleteOneRecurrent(ctx context.Context, id *uuid.UUID, movement model.Movement, recurrent, newRecurrent model.RecurrentMovement) error {
	userID := ctx.Value(authentication.UserID).(string)
	gormTransactionErr := p.gorm.Transaction(func(tx *gorm.DB) error {
		if movement.ID != nil {
			if err := p.gorm.Where("user_id=?", userID).Delete(&model.Movement{}, movement.ID).Error; err != nil {
				return handleError("repository error", err)
			}
			if movement.IsPaid {
				err := p.updateWallet(
					ctx,
					tx,
					strategy{
						originalMovement: movement,
						newMovement:      movement,
						updateStrategies: updateStrategyRevertPay,
					})
				if err != nil {
					return err
				}
			}
		}

		_, err := p.recurrentRepo.Update(ctx, tx, id, recurrent)
		if err != nil {
			return err
		}

		_, err = p.recurrentRepo.AddConsistent(ctx, tx, newRecurrent)
		if err != nil {
			return err
		}

		return nil
	})
	if gormTransactionErr != nil {
		return handleError("repository error", gormTransactionErr)
	}
	return nil
}

func (p PgRepository) DeleteAllNextRecurrent(ctx context.Context, id *uuid.UUID, movement model.Movement, recurrent model.RecurrentMovement) error {
	userID := ctx.Value(authentication.UserID).(string)

	gormTransactionErr := p.gorm.Transaction(func(tx *gorm.DB) error {
		if movement.ID != nil {
			if err := p.gorm.Where("user_id=?", userID).Delete(&model.Movement{}, movement.ID).Error; err != nil {
				return handleError("repository error", err)
			}
			if movement.IsPaid {
				err := p.updateWallet(
					ctx,
					tx,
					strategy{
						originalMovement: movement,
						newMovement:      movement,
						updateStrategies: updateStrategyRevertPay,
					})
				if err != nil {
					return err
				}
			}
		}

		if recurrent.InitialDate.Month() == recurrent.EndDate.Month() && recurrent.InitialDate.Year() == recurrent.EndDate.Year() {
			err := p.recurrentRepo.Delete(ctx, id)
			if err != nil {
				return err
			}
			return nil
		}
		_, err := p.recurrentRepo.Update(ctx, tx, id, recurrent)
		if err != nil {
			return err
		}
		return nil
	})
	if gormTransactionErr != nil {
		return handleError("repository error", gormTransactionErr)
	}
	return nil
}

func handleError(msg string, err error) error {
	businessErr := model.BusinessError{}
	if ok := errors.As(err, &businessErr); ok {
		return businessErr
	}
	return model.BuildBusinessError(msg, http.StatusInternalServerError, err)
}

func (p PgRepository) AddConsistent(ctx context.Context, tx *gorm.DB, movement model.Movement) (model.Movement, error) {
	userID := ctx.Value(authentication.UserID).(string)
	now := time.Now()
	id := uuid.New()

	movement.ID = &id
	movement.DateCreate = now
	movement.DateUpdate = now
	movement.UserID = userID

	result := tx.
		Select([]string{
			"id",
			"description",
			"amount",
			"date",
			"user_id",
			"type_payment",
			"date_create",
			"date_update",
			"is_paid",
			"sub_category_id",
			"category_id",
			"wallet_id",
			"recurrent_id",
		}).
		Create(&movement)
	if err := result.Error; err != nil {
		return model.Movement{}, handleError("repository error", err)
	}
	return movement, nil // TODO recuperar o objeto salvo de result
}

func (p PgRepository) AddUpdatingWallet(ctx context.Context, tx *gorm.DB, movement model.Movement) (model.Movement, error) {
	if tx != nil {
		mov, err := p.addUpdatingWalletConsistent(ctx, tx, movement)
		if err != nil {
			return model.Movement{}, errors.New("repository error")
		}
		return mov, nil
	}

	gormTransactionErr := p.gorm.Transaction(func(tx *gorm.DB) error {
		_, err := p.addUpdatingWalletConsistent(ctx, tx, movement)
		if err != nil {
			return err
		}
		return nil
	})
	if gormTransactionErr != nil {
		return model.Movement{}, handleError("repository error", gormTransactionErr)
	}
	return movement, nil
}

func (p PgRepository) addUpdatingWalletConsistent(ctx context.Context, tx *gorm.DB, movement model.Movement) (model.Movement, error) {
	if !movement.IsPaid {
		return model.Movement{}, errors.New("estimate can`t update wallet")
	}
	var recurrent model.RecurrentMovement
	var err error
	if movement.IsRecurrent && movement.RecurrentID == nil {
		recurrent, err = p.recurrentRepo.AddConsistent(ctx, tx, model.ToRecurrentMovement(movement))
		if err != nil {
			return model.Movement{}, err
		}
		movement.RecurrentID = recurrent.ID
	}

	movement, err = p.AddConsistent(ctx, tx, movement)
	if err != nil {
		return model.Movement{}, handleError("repository error", err)
	}

	wallet, err := p.walletRepo.FindByID(ctx, movement.WalletID)
	if err != nil {
		return model.Movement{}, err
	}
	wallet.Balance += movement.Amount
	_, err = p.walletRepo.UpdateConsistent(ctx, tx, wallet)
	if err != nil {
		return model.Movement{}, err
	}

	return movement, nil // TODO recuperar o objeto salvo de result
}

func (p PgRepository) buildBaseQuery(userID string, joins ...string) *gorm.DB {
	query := p.gorm.
		Table(tableName).
		Where(fmt.Sprintf("%s.user_id=?", tableName), userID)

	for _, join := range joins {
		query = query.Joins(join)
	}
	return query
}
