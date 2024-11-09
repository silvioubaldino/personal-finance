package repository

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"personal-finance/internal/domain/wallet/repository"
	"personal-finance/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	Add(ctx context.Context, movement model.Movement, userID string) (model.Movement, error)
	AddConsistent(_ context.Context, tx *gorm.DB, movement model.Movement, userID string) (model.Movement, error)
	AddUpdatingWallet(ctx context.Context, tx *gorm.DB, movement model.Movement, userID string) (model.Movement, error)
	FindByID(_ context.Context, id uuid.UUID, userID string) (model.Movement, error)
	FindByPeriod(ctx context.Context, period model.Period, userID string) (model.MovementList, error)
	Update(ctx context.Context, id uuid.UUID, transaction model.Movement, userID string) (model.Movement, error)
	UpdateIsPay(ctx context.Context, id uuid.UUID, newMovement model.Movement, userID string) (model.Movement, error)
	Delete(ctx context.Context, id uuid.UUID, userID string) error
	FindByTransactionID(_ context.Context, parentID uuid.UUID, transactionStatusID int, userID string) (model.MovementList, error)
	FindByStatusByPeriod(_ context.Context, transactionStatusID int, period model.Period, userID string) ([]model.Movement, error)
	EstimateExpensesByPeriod(period model.Period, userID string) (float64, error)
	EstimateIncomesByPeriod(period model.Period, userID string) (float64, error)
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
	defaultJoins = []string{"Wallet", "Category", "TypePayment", "SubCategory"}

	updateStrategyEmpty           updateStrategy = ""
	updateStrategyDifferentAmount updateStrategy = "different_amount"
	updateStrategyDifferentWallet updateStrategy = "different_wallet"
	updateStrategyPay             updateStrategy = "pay"
	updateStrategyRevertPay       updateStrategy = "revert_pay"
)

type PgRepository struct {
	gorm       *gorm.DB
	walletRepo repository.Repository
}

func NewPgRepository(gorm *gorm.DB, walletRepo repository.Repository) Repository {
	return PgRepository{
		gorm:       gorm,
		walletRepo: walletRepo,
	}
}

func (p PgRepository) Add(_ context.Context, movement model.Movement, userID string) (model.Movement, error) {
	now := time.Now()
	id := uuid.New()

	movement.ID = &id
	movement.DateCreate = now
	movement.DateUpdate = now
	movement.UserID = userID

	if movement.TransactionID == &uuid.Nil {
		movement.TransactionID = movement.ID
	}

	result := p.gorm.Create(&movement)
	if err := result.Error; err != nil {
		log.Printf("Error: %v", err)
		return model.Movement{}, handleError("repository error", err)
	}
	return movement, nil
}

func (p PgRepository) FindByID(_ context.Context, id uuid.UUID, userID string) (model.Movement, error) {
	var transaction model.Movement
	result := p.gorm.Where("user_id=?", userID).First(&transaction, id)
	if err := result.Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.Movement{}, model.BuildErrNotfound("resource not found")
		}
		return model.Movement{}, handleError("repository error", err)
	}
	return transaction, nil
}

func (p PgRepository) FindByPeriod(_ context.Context, period model.Period, userID string) (model.MovementList, error) {
	var transaction []model.Movement
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
			"movements.status_id",
			`w.id as "Wallet__id"`,
			`w.description as "Wallet__description"`,
			`c.id as "Category__id"`,
			`c.description as "Category__description"`,
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

func (p PgRepository) Update(ctx context.Context, id uuid.UUID, newMovement model.Movement, userID string) (model.Movement, error) {
	movementFound, err := p.FindByID(ctx, id, userID)
	if err != nil {
		return model.Movement{}, err
	}
	var updated bool
	strategy := strategy{movementFound, newMovement, updateStrategyEmpty}

	if newMovement.Description != "" {
		movementFound.Description = newMovement.Description
		updated = true
	}
	if newMovement.Date != nil {
		movementFound.Date = newMovement.Date
		updated = true
	}
	if newMovement.TypePaymentID != 0 {
		movementFound.TypePaymentID = newMovement.TypePaymentID
		updated = true
	}
	if newMovement.CategoryID != nil {
		movementFound.CategoryID = newMovement.CategoryID
		updated = true
	}
	if newMovement.Amount != 0 && newMovement.Amount != movementFound.Amount {
		strategy.updateStrategies = updateStrategyDifferentAmount
		movementFound.Amount = newMovement.Amount
		updated = true
	}
	if newMovement.WalletID != nil {
		if *newMovement.WalletID != *movementFound.WalletID {
			strategy.updateStrategies = updateStrategyDifferentWallet
			movementFound.WalletID = newMovement.WalletID
			movementFound.Amount = newMovement.Amount
			updated = true
		}
	}

	if !updated {
		return model.Movement{}, handleError("no changes", errors.New("no changes"))
	}
	movementFound.DateUpdate = time.Now()

	gormTransactionErr := p.gorm.Transaction(func(tx *gorm.DB) error {
		movementFound, err = p.update(ctx, tx, movementFound, &strategy, userID)
		return err
	})
	if gormTransactionErr != nil {
		return model.Movement{}, gormTransactionErr
	}

	return movementFound, nil
}

func (p PgRepository) UpdateIsPay(ctx context.Context, id uuid.UUID, newMovement model.Movement, userID string) (model.Movement, error) {
	movementFound, err := p.FindByID(context.Background(), id, userID)
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
		movementFound.StatusID = newMovement.StatusID
	}

	movementFound.DateUpdate = time.Now()

	gormTransactionErr := p.gorm.Transaction(func(tx *gorm.DB) error {
		movementFound, err = p.update(ctx, tx, movementFound, &strategy, userID)
		return err
	})
	if gormTransactionErr != nil {
		return model.Movement{}, gormTransactionErr
	}

	return movementFound, nil
}

func (p PgRepository) update(ctx context.Context, tx *gorm.DB, movementFound model.Movement, strategy *strategy, userID string) (model.Movement, error) {
	result := tx.Save(&movementFound)
	if err := result.Error; err != nil {
		return model.Movement{}, err
	}
	err := p.updateWallet(ctx, tx, *strategy, userID)
	if err != nil {
		return model.Movement{}, err
	}

	tx = result.Scan(&movementFound)

	return movementFound, nil
}

func (p PgRepository) updateWallet(ctx context.Context, tx *gorm.DB, strategy strategy, userID string) error {
	originalWallet, err := p.walletRepo.FindByID(ctx, strategy.originalMovement.WalletID, userID)
	if err != nil {
		return err
	}

	switch strategy.updateStrategies {
	case updateStrategyPay:
		originalWallet.Balance += strategy.newMovement.Amount
		_, err = p.walletRepo.UpdateConsistent(ctx, tx, originalWallet, userID)
		if err != nil {
			return err
		}
		return nil

	case updateStrategyRevertPay:
		originalWallet.Balance -= strategy.newMovement.Amount
		_, err = p.walletRepo.UpdateConsistent(ctx, tx, originalWallet, userID)
		if err != nil {
			return err
		}
		return nil

	case updateStrategyDifferentAmount:
		originalWallet.Balance += strategy.newMovement.Amount - strategy.originalMovement.Amount
		_, err = p.walletRepo.UpdateConsistent(ctx, tx, originalWallet, userID)
		if err != nil {
			return err
		}
		return nil

	case updateStrategyDifferentWallet:
		originalWallet.Balance -= strategy.originalMovement.Amount
		_, err = p.walletRepo.UpdateConsistent(ctx, tx, originalWallet, userID)
		if err != nil {
			return err
		}

		newWallet, err := p.walletRepo.FindByID(ctx, strategy.newMovement.WalletID, userID)
		if err != nil {
			return err
		}
		if strategy.newMovement.Amount == 0 {
			strategy.newMovement.Amount = strategy.originalMovement.Amount
		}
		newWallet.Balance += strategy.newMovement.Amount

		_, err = p.walletRepo.UpdateConsistent(ctx, tx, newWallet, userID)
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

func (p PgRepository) Delete(_ context.Context, id uuid.UUID, userID string) error {
	if err := p.gorm.Where("user_id=?", userID).Delete(&model.Movement{}, id).Error; err != nil {
		return handleError("repository error", err)
	}
	return nil
}

func (p PgRepository) FindByTransactionID(_ context.Context, parentID uuid.UUID, transactionStatusID int, userID string) (model.MovementList, error) {
	var movementList model.MovementList
	result := p.buildBaseQuery(userID, defaultJoins...).
		Where("status_id = ?", transactionStatusID).
		Where("transaction_id = ?", parentID).
		Find(&movementList)
	if err := result.Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []model.Movement{}, model.BuildErrNotfound("resource not found")
		}
		return []model.Movement{}, handleError("repository error", err)
	}
	return movementList, nil
}

func (p PgRepository) FindByStatusByPeriod(_ context.Context, transactionStatusID int, period model.Period, userID string) ([]model.Movement, error) {
	var movements []model.Movement
	result := p.buildBaseQuery(userID, defaultJoins...).
		Where("status_id = ?", transactionStatusID).
		Where("date BETWEEN ? AND ?", period.From, period.To).
		Find(&movements)
	if err := result.Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []model.Movement{}, model.BuildErrNotfound("resource not found")
		}
		return []model.Movement{}, handleError("repository error", err)
	}
	return movements, nil
}

func handleError(msg string, err error) error {
	businessErr := model.BusinessError{}
	if ok := errors.As(err, &businessErr); ok {
		return businessErr
	}
	return model.BuildBusinessError(msg, http.StatusInternalServerError, err)
}

func (p PgRepository) AddConsistent(_ context.Context, tx *gorm.DB, movement model.Movement, userID string) (model.Movement, error) {
	now := time.Now()
	id := uuid.New()

	movement.ID = &id
	movement.DateCreate = now
	movement.DateUpdate = now
	movement.UserID = userID

	if movement.TransactionID == &uuid.Nil {
		movement.TransactionID = movement.ID
	}

	result := tx.Create(&movement)
	if err := result.Error; err != nil {
		return model.Movement{}, handleError("repository error", err)
	}
	return movement, nil // TODO recuperar o objeto salvo de result
}

func (p PgRepository) AddUpdatingWallet(ctx context.Context, tx *gorm.DB, movement model.Movement, userID string) (model.Movement, error) {
	if tx != nil {
		mov, err := p.addUpdatingWalletConsistent(ctx, tx, movement, userID)
		if err != nil {
			return model.Movement{}, errors.New("repository error")
		}
		return mov, nil
	}

	gormTransactionErr := p.gorm.Transaction(func(tx *gorm.DB) error {
		_, err := p.addUpdatingWalletConsistent(ctx, tx, movement, userID)
		if err != nil {
			return err
		}
		return nil
	})
	if gormTransactionErr != nil {
		return model.Movement{}, errors.New("repository error")
	}
	return movement, nil
}

func (p PgRepository) addUpdatingWalletConsistent(ctx context.Context, tx *gorm.DB, movement model.Movement, userID string) (model.Movement, error) {
	if movement.StatusID == model.TransactionStatusPlannedID {
		return model.Movement{}, errors.New("estimate can`t update wallet")
	}

	movement, err := p.AddConsistent(ctx, tx, movement, userID)
	if err != nil {
		return model.Movement{}, handleError("repository error", err)
	}

	wallet, err := p.walletRepo.FindByID(ctx, movement.WalletID, userID)
	if err != nil {
		return model.Movement{}, err
	}
	wallet.Balance += movement.Amount
	_, err = p.walletRepo.UpdateConsistent(ctx, tx, wallet, userID)
	if err != nil {
		return model.Movement{}, err
	}

	return movement, nil // TODO recuperar o objeto salvo de result
}

func (p PgRepository) EstimateExpensesByPeriod(period model.Period, userID string) (float64, error) {
	var expense float64
	result := p.buildBaseQuery(userID).
		Select("COALESCE(sum(amount), 0) as expense").
		Where("date BETWEEN ? AND ?", period.From, period.To).
		Where("status_id = ?", model.TransactionStatusPlannedID).
		Where("amount < 0").
		Scan(&expense)
	if err := result.Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, model.BuildErrNotfound("resource not found")
		}
		return 0, handleError("repository error", err)
	}
	return expense, nil
}

func (p PgRepository) EstimateIncomesByPeriod(period model.Period, userID string) (float64, error) {
	var income float64
	result := p.buildBaseQuery(userID).
		Select("COALESCE(sum(amount), 0) as income").
		Where("date BETWEEN ? AND ?", period.From, period.To).
		Where("status_id = ?", model.TransactionStatusPlannedID).
		Where("amount > 0").
		Scan(&income)
	if err := result.Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, model.BuildErrNotfound("resource not found")
		}
		return 0, handleError("repository error", err)
	}
	return income, nil
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
