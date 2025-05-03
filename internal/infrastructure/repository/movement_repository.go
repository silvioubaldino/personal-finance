package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/plataform/authentication"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MovementRepository struct {
	db *gorm.DB
}

func NewMovementRepository(db *gorm.DB) *MovementRepository {
	return &MovementRepository{
		db: db,
	}
}

func (r *MovementRepository) Add(ctx context.Context, tx *gorm.DB, movement domain.Movement) (domain.Movement, error) {
	var isLocalTx bool
	if tx == nil {
		isLocalTx = true
		tx = r.db.WithContext(ctx).Begin()
		defer tx.Rollback()
	}

	userID := ctx.Value(authentication.UserID).(string)
	now := time.Now()
	id := uuid.New()

	movement.ID = &id
	movement.DateCreate = now
	movement.DateUpdate = now
	movement.UserID = userID

	dbMovement := FromMovementDomain(movement)

	if err := tx.WithContext(ctx).Create(&dbMovement).Error; err != nil {
		return domain.Movement{}, domain.WrapInternalError(err, "error creating movement")
	}

	if isLocalTx {
		if err := tx.Commit().Error; err != nil {
			return domain.Movement{}, domain.WrapInternalError(err, "error committing transaction")
		}
	}

	return dbMovement.ToDomain(), nil
}

func (r *MovementRepository) FindByID(ctx context.Context, id uuid.UUID) (domain.Movement, error) {
	var dbModel MovementDB
	tableName := dbModel.TableName()

	query := BuildBaseQuery(ctx, r.db, tableName)
	query = r.appendPreloads(query)

	if err := query.First(&dbModel, fmt.Sprintf("%s.id = ?", tableName), id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Movement{}, domain.WrapNotFound(err, "movement not found")
		}
		return domain.Movement{}, domain.WrapInternalError(err, "error finding movement")
	}

	return dbModel.ToDomain(), nil
}

func (r *MovementRepository) FindByPeriod(ctx context.Context, period domain.Period) (domain.MovementList, error) {
	var dbModel MovementDB
	tableName := dbModel.TableName()

	query := BuildBaseQuery(ctx, r.db, tableName)
	query = r.appendPreloads(query)

	var dbMovements []MovementDB
	err := query.Where(fmt.Sprintf("%s.date BETWEEN ? AND ?", tableName), period.From, period.To).Find(&dbMovements).Error
	if err != nil {
		return nil, domain.WrapInternalError(err, "error finding movements by period")
	}

	movements := make(domain.MovementList, len(dbMovements))
	for i, dbMovement := range dbMovements {
		movements[i] = dbMovement.ToDomain()
	}

	return movements, nil
}

func (r *MovementRepository) appendPreloads(query *gorm.DB) *gorm.DB {
	var (
		movementDB    MovementDB
		walletDB      WalletDB
		categoryDB    CategoryDB
		subCategoryDB SubCategoryDB
	)

	movementTable := movementDB.TableName()
	walletTable := walletDB.TableName()
	categoryTable := categoryDB.TableName()
	subCategoryTable := subCategoryDB.TableName()

	return query.
		Joins(fmt.Sprintf("LEFT JOIN %s ON %s.id = %s.wallet_id", walletTable, walletTable, movementTable)).
		Joins(fmt.Sprintf("LEFT JOIN %s ON %s.id = %s.category_id", categoryTable, categoryTable, movementTable)).
		Joins(fmt.Sprintf("LEFT JOIN %s ON %s.id = %s.sub_category_id", subCategoryTable, subCategoryTable, movementTable)).
		Select(fmt.Sprintf("%s.*, %s.*, %s.*, %s.*",
			movementTable, walletTable, categoryTable, subCategoryTable))
}
