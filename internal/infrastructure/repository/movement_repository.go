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
		return domain.Movement{}, fmt.Errorf("error creating movement: %w: %s", ErrDatabaseError, err.Error())
	}

	if isLocalTx {
		if err := tx.Commit().Error; err != nil {
			return domain.Movement{}, fmt.Errorf("error committing transaction: %w: %s", ErrDatabaseError, err.Error())
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
			return domain.Movement{}, fmt.Errorf("error finding movement: %w: %s", ErrMovementNotFound, err.Error())
		}
		return domain.Movement{}, fmt.Errorf("error finding movement: %w: %s", ErrDatabaseError, err.Error())
	}

	return dbModel.ToDomain(), nil
}

func (r *MovementRepository) FindByInstallmentGroupFromNumber(ctx context.Context, groupID uuid.UUID, fromNumber int) (domain.MovementList, error) {
	var dbModel MovementDB
	tableName := dbModel.TableName()

	query := BuildBaseQuery(ctx, r.db, tableName)
	query = r.appendPreloads(query)

	var dbMovements []MovementDB
	err := query.Where(fmt.Sprintf("%s.installment_group_id = ? AND %s.installment_number >= ?", tableName, tableName), groupID, fromNumber).
		Order(fmt.Sprintf("%s.installment_number ASC", tableName)).
		Find(&dbMovements).Error
	if err != nil {
		return domain.MovementList{}, fmt.Errorf("error finding movements by installment group: %w: %s", ErrDatabaseError, err.Error())
	}

	movements := make(domain.MovementList, len(dbMovements))
	for i, dbMovement := range dbMovements {
		movements[i] = dbMovement.ToDomain()
	}

	return movements, nil
}

func (r *MovementRepository) FindByPeriod(ctx context.Context, period domain.Period) (domain.MovementList, error) {
	var dbModel MovementDB
	tableName := dbModel.TableName()

	query := BuildBaseQuery(ctx, r.db, tableName)
	query = r.appendPreloads(query)

	var dbMovements []MovementDB
	err := query.Where(fmt.Sprintf("%s.date BETWEEN ? AND ?", tableName), period.From, period.To).
		Find(&dbMovements).Error
	if err != nil {
		return domain.MovementList{}, fmt.Errorf("error finding movements by period: %w: %s", ErrDatabaseError, err.Error())
	}

	movements := make(domain.MovementList, len(dbMovements))
	for i, dbMovement := range dbMovements {
		movements[i] = dbMovement.ToDomain()
	}

	return movements, nil
}

func (r *MovementRepository) UpdateIsPaid(ctx context.Context, tx *gorm.DB, id uuid.UUID, movement domain.Movement) (domain.Movement, error) {
	var isLocalTx bool
	if tx == nil {
		isLocalTx = true
		tx = r.db.WithContext(ctx).Begin()
		defer tx.Rollback()
	}

	userID := ctx.Value(authentication.UserID).(string)
	now := time.Now()

	result := tx.Model(&MovementDB{}).
		Where("id = ? AND user_id = ?", id, userID).
		Updates(map[string]interface{}{
			"is_paid":     movement.IsPaid,
			"date_update": now,
		})

	if err := result.Error; err != nil {
		return domain.Movement{}, fmt.Errorf("error updating movement: %w: %s", ErrDatabaseError, err.Error())
	}

	if result.RowsAffected == 0 {
		return domain.Movement{}, fmt.Errorf("error updating movement: %w", ErrMovementNotFound)
	}

	if isLocalTx {
		if err := tx.Commit().Error; err != nil {
			return domain.Movement{}, fmt.Errorf("error committing transaction: %w: %s", ErrDatabaseError, err.Error())
		}
	}

	return movement, nil
}

func (r *MovementRepository) Update(ctx context.Context, tx *gorm.DB, id uuid.UUID, movement domain.Movement) (domain.Movement, error) {
	var isLocalTx bool
	if tx == nil {
		isLocalTx = true
		tx = r.db.WithContext(ctx).Begin()
		defer tx.Rollback()
	}

	userID := ctx.Value(authentication.UserID).(string)
	now := time.Now()

	movement.DateUpdate = now
	dbMovement := FromMovementDomain(movement)

	result := tx.Model(&MovementDB{}).
		Where("id = ? AND user_id = ?", id, userID).
		Select("description", "amount", "date", "wallet_id", "category_id", "sub_category_id", "type_payment", "date_update").
		Updates(dbMovement)

	if err := result.Error; err != nil {
		return domain.Movement{}, fmt.Errorf("error updating movement: %w: %s", ErrDatabaseError, err.Error())
	}

	if result.RowsAffected == 0 {
		return domain.Movement{}, fmt.Errorf("error updating movement: %w", ErrMovementNotFound)
	}

	if isLocalTx {
		if err := tx.Commit().Error; err != nil {
			return domain.Movement{}, fmt.Errorf("error committing transaction: %w: %s", ErrDatabaseError, err.Error())
		}
	}

	return dbMovement.ToDomain(), nil
}

func (r *MovementRepository) appendPreloads(query *gorm.DB) *gorm.DB {
	return query.Preload("Category").Preload("SubCategory").Preload("Wallet").Preload("Invoice")
}

func (r *MovementRepository) FindByInvoiceID(ctx context.Context, invoiceID uuid.UUID) (domain.MovementList, error) {
	var dbModel MovementDB
	tableName := dbModel.TableName()

	query := BuildBaseQuery(ctx, r.db, tableName)
	query = r.appendPreloads(query)

	var dbMovements []MovementDB
	err := query.Where(fmt.Sprintf("%s.invoice_id = ? AND %s.type_payment != ?", tableName, tableName), invoiceID, domain.TypePaymentInvoicePayment).
		Find(&dbMovements).Error
	if err != nil {
		return domain.MovementList{}, fmt.Errorf("error finding movements by invoice id: %w: %s", ErrDatabaseError, err.Error())
	}

	movements := make(domain.MovementList, len(dbMovements))
	for i, dbMovement := range dbMovements {
		movements[i] = dbMovement.ToDomain()
	}

	return movements, nil
}

func (r *MovementRepository) Delete(ctx context.Context, tx *gorm.DB, id uuid.UUID) error {
	var isLocalTx bool
	if tx == nil {
		isLocalTx = true
		tx = r.db.WithContext(ctx).Begin()
		defer tx.Rollback()
	}

	userID := ctx.Value(authentication.UserID).(string)

	result := tx.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&MovementDB{})

	if err := result.Error; err != nil {
		return fmt.Errorf("error deleting movement: %w: %s", ErrDatabaseError, err.Error())
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("error deleting movement: %w", ErrMovementNotFound)
	}

	if isLocalTx {
		if err := tx.Commit().Error; err != nil {
			return fmt.Errorf("error committing transaction: %w: %s", ErrDatabaseError, err.Error())
		}
	}

	return nil
}

func (r *MovementRepository) DeleteByInvoiceID(ctx context.Context, tx *gorm.DB, invoiceID uuid.UUID) error {
	var isLocalTx bool
	if tx == nil {
		isLocalTx = true
		tx = r.db.WithContext(ctx).Begin()
		defer tx.Rollback()
	}

	userID := ctx.Value(authentication.UserID).(string)

	result := tx.WithContext(ctx).
		Where("invoice_id = ? AND type_payment = ? AND user_id = ?", invoiceID, domain.TypePaymentInvoicePayment, userID).
		Delete(&MovementDB{})

	if err := result.Error; err != nil {
		return fmt.Errorf("error deleting movements by invoice id: %w: %s", ErrDatabaseError, err.Error())
	}

	if isLocalTx {
		if err := tx.Commit().Error; err != nil {
			return fmt.Errorf("error committing transaction: %w: %s", ErrDatabaseError, err.Error())
		}
	}

	return nil
}
