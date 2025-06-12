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

type RecurrentMovementRepository struct {
	db *gorm.DB
}

func NewRecurrentMovementRepository(db *gorm.DB) *RecurrentMovementRepository {
	return &RecurrentMovementRepository{
		db: db,
	}
}

func (r *RecurrentMovementRepository) Add(ctx context.Context, tx *gorm.DB, recurrentMovement domain.RecurrentMovement) (domain.RecurrentMovement, error) {
	var isLocalTx bool
	if tx == nil {
		isLocalTx = true
		tx = r.db.Begin()
		defer tx.Rollback()
	}

	userID := ctx.Value(authentication.UserID).(string)
	id := uuid.New()

	recurrentMovement.ID = &id
	recurrentMovement.UserID = userID

	dbRecurrentMovement := FromRecurrentMovementDomain(recurrentMovement)

	if err := tx.Create(&dbRecurrentMovement).Error; err != nil {
		return domain.RecurrentMovement{}, domain.WrapInternalError(err, "error creating recurrent movement")
	}

	if isLocalTx {
		if err := tx.Commit().Error; err != nil {
			return domain.RecurrentMovement{}, domain.WrapInternalError(err, "error committing transaction")
		}
	}

	return dbRecurrentMovement.ToDomain(), nil
}

func (r *RecurrentMovementRepository) FindByID(ctx context.Context, id uuid.UUID) (domain.RecurrentMovement, error) {
	var dbModel RecurrentMovementDB
	tableName := dbModel.TableName()

	query := BuildBaseQuery(ctx, r.db, tableName)
	query = r.appendPreloads(query)

	err := query.
		Where(fmt.Sprintf("%s.id = ?", tableName), id).
		First(&dbModel).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.RecurrentMovement{}, fmt.Errorf("error finding recurrent movement: %w: %s", ErrRecurrentMovementNotFound, err.Error())
		}
		return domain.RecurrentMovement{}, domain.WrapInternalError(err, "error finding recurrent movement")
	}

	return dbModel.ToDomain(), nil
}

func (r *RecurrentMovementRepository) FindByMonth(ctx context.Context, date time.Time) ([]domain.RecurrentMovement, error) {
	var dbRecurrentMovements []RecurrentMovementDB
	var dbModel RecurrentMovementDB
	tableName := dbModel.TableName()

	query := BuildBaseQuery(ctx, r.db, tableName)
	query = r.appendPreloads(query)

	err := query.
		Order(fmt.Sprintf("%s.initial_date desc", tableName)).
		Where(fmt.Sprintf("%s.initial_date <= ?", tableName), date).
		Where(fmt.Sprintf("(%s.end_date >= ? OR %s.end_date IS NULL)", tableName, tableName), date).
		Find(&dbRecurrentMovements).Error
	if err != nil {
		return nil, domain.WrapInternalError(err, "error finding recurrent movements")
	}

	result := make([]domain.RecurrentMovement, len(dbRecurrentMovements))
	for i, rm := range dbRecurrentMovements {
		result[i] = rm.ToDomain()
	}

	return result, nil
}

func (r *RecurrentMovementRepository) appendPreloads(query *gorm.DB) *gorm.DB {
	var (
		recurrentMovementDB RecurrentMovementDB
		walletDB            WalletDB
		categoryDB          CategoryDB
		subCategoryDB       SubCategoryDB
	)

	recurrentMovementTable := recurrentMovementDB.TableName()
	walletTable := walletDB.TableName()
	categoryTable := categoryDB.TableName()
	subCategoryTable := subCategoryDB.TableName()

	return query.
		Joins(fmt.Sprintf("LEFT JOIN %s w ON w.id = %s.wallet_id", walletTable, recurrentMovementTable)).
		Joins(fmt.Sprintf("LEFT JOIN %s c ON c.id = %s.category_id", categoryTable, recurrentMovementTable)).
		Joins(fmt.Sprintf("LEFT JOIN %s sc ON sc.id = %s.sub_category_id", subCategoryTable, recurrentMovementTable)).
		Select([]string{
			fmt.Sprintf("%s.*", recurrentMovementTable),
			`w.id as "Wallet__id"`,
			`w.description as "Wallet__description"`,
			`w.balance as "Wallet__balance"`,
			`c.id as "Category__id"`,
			`c.description as "Category__description"`,
			`sc.id as "SubCategory__id"`,
			`sc.description as "SubCategory__description"`,
		})
}
