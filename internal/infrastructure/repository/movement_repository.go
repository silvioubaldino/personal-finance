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
		Where(fmt.Sprintf("%s.type_payment NOT IN ?", tableName), []domain.TypePayment{
			domain.TypePaymentCreditCard,
			domain.TypePaymentInvoiceRemainder,
		}).
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

func (r *MovementRepository) FindByRecurrentIDAndMonth(ctx context.Context, recurrentID uuid.UUID, month time.Time) (*domain.Movement, error) {
	var dbModel MovementDB
	tableName := dbModel.TableName()

	firstDay := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	lastDay := firstDay.AddDate(0, 1, -1)

	query := BuildBaseQuery(ctx, r.db, tableName)
	err := query.
		Where(fmt.Sprintf("%s.recurrent_id = ? AND %s.date BETWEEN ? AND ?", tableName, tableName), recurrentID, firstDay, lastDay).
		First(&dbModel).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("error finding movement by recurrent id and month: %w: %s", ErrDatabaseError, err.Error())
	}

	m := dbModel.ToDomain()
	return &m, nil
}

func (r *MovementRepository) UpdateStatementLink(ctx context.Context, tx *gorm.DB, id uuid.UUID, movement domain.Movement) (domain.Movement, error) {
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
			"description": movement.Description,
			"amount":      movement.Amount,
			"date":        movement.Date,
			"wallet_id":   movement.WalletID,
			"is_paid":     true,
			"date_update": now,
		})

	if err := result.Error; err != nil {
		return domain.Movement{}, fmt.Errorf("error updating movement statement link: %w: %s", ErrDatabaseError, err.Error())
	}
	if result.RowsAffected == 0 {
		return domain.Movement{}, fmt.Errorf("error updating movement: %w", ErrMovementNotFound)
	}

	if isLocalTx {
		if err := tx.Commit().Error; err != nil {
			return domain.Movement{}, fmt.Errorf("error committing transaction: %w: %s", ErrDatabaseError, err.Error())
		}
	}

	movement.DateUpdate = now
	return movement, nil
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

func (r *MovementRepository) PayByInvoiceID(ctx context.Context, tx *gorm.DB, invoiceID uuid.UUID) error {
	var isLocalTx bool
	if tx == nil {
		isLocalTx = true
		tx = r.db.WithContext(ctx).Begin()
		defer tx.Rollback()
	}

	userID := ctx.Value(authentication.UserID).(string)
	now := time.Now()

	result := tx.Model(&MovementDB{}).
		Where("invoice_id = ? AND user_id = ? AND type_payment NOT IN ?", invoiceID, userID, []domain.TypePayment{
			domain.TypePaymentInvoicePayment,
			domain.TypePaymentInvoiceRemainder,
		}).
		Updates(map[string]interface{}{
			"is_paid":     true,
			"date_update": now,
		})

	if err := result.Error; err != nil {
		return fmt.Errorf("error paying movements by invoice id: %w: %s", ErrDatabaseError, err.Error())
	}

	if isLocalTx {
		if err := tx.Commit().Error; err != nil {
			return fmt.Errorf("error committing transaction: %w: %s", ErrDatabaseError, err.Error())
		}
	}

	return nil
}

func (r *MovementRepository) FindInvoicePaymentByInvoiceID(ctx context.Context, invoiceID uuid.UUID) (domain.Movement, error) {
	var dbModel MovementDB
	tableName := dbModel.TableName()

	query := BuildBaseQuery(ctx, r.db, tableName)
	query = r.appendPreloads(query)

	if err := query.Where(fmt.Sprintf("%s.invoice_id = ? AND %s.type_payment = ?", tableName, tableName), invoiceID, domain.TypePaymentInvoicePayment).
		First(&dbModel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Movement{}, fmt.Errorf("error finding invoice payment movement: %w: %s", ErrMovementNotFound, err.Error())
		}
		return domain.Movement{}, fmt.Errorf("error finding invoice payment movement: %w: %s", ErrDatabaseError, err.Error())
	}

	return dbModel.ToDomain(), nil
}

func (r *MovementRepository) RevertPayByInvoiceID(ctx context.Context, tx *gorm.DB, invoiceID uuid.UUID) error {
	var isLocalTx bool
	if tx == nil {
		isLocalTx = true
		tx = r.db.WithContext(ctx).Begin()
		defer tx.Rollback()
	}

	userID := ctx.Value(authentication.UserID).(string)
	now := time.Now()

	result := tx.Model(&MovementDB{}).
		Where("invoice_id = ? AND user_id = ? AND type_payment NOT IN ?", invoiceID, userID, []domain.TypePayment{
			domain.TypePaymentInvoicePayment,
			domain.TypePaymentInvoiceRemainder,
		}).
		Updates(map[string]interface{}{
			"is_paid":     false,
			"date_update": now,
		})

	if err := result.Error; err != nil {
		return fmt.Errorf("error reverting pay for movements by invoice id: %w: %s", ErrDatabaseError, err.Error())
	}

	if isLocalTx {
		if err := tx.Commit().Error; err != nil {
			return fmt.Errorf("error committing transaction: %w: %s", ErrDatabaseError, err.Error())
		}
	}

	return nil
}

func (r *MovementRepository) FindAllByUserID(ctx context.Context) ([]domain.Movement, error) {
	var dbModel MovementDB
	tableName := dbModel.TableName()

	query := BuildBaseQuery(ctx, r.db, tableName)
	query = r.appendPreloads(query)

	var dbModels []MovementDB
	if err := query.Order(fmt.Sprintf("%s.date DESC", tableName)).Find(&dbModels).Error; err != nil {
		return nil, fmt.Errorf("error finding all movements: %w: %s", ErrDatabaseError, err.Error())
	}

	movements := make([]domain.Movement, len(dbModels))
	for i, m := range dbModels {
		movements[i] = m.ToDomain()
	}

	return movements, nil
}

func (r *MovementRepository) DeleteAllByUserID(ctx context.Context, tx *gorm.DB, userID string) error {
	db := r.db
	if tx != nil {
		db = tx
	}

	err := db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&MovementDB{}).Error
	if err != nil {
		return fmt.Errorf("error deleting movements: %w: %s", ErrDatabaseError, err.Error())
	}

	return nil
}

type UnpaidMovement struct {
	ID          string `gorm:"column:id"`
	Description string `gorm:"column:description"`
	UserID      string `gorm:"column:user_id"`
}

func (r *MovementRepository) FindUnpaidByDate(ctx context.Context, date time.Time) ([]UnpaidMovement, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 59, 999999999, time.UTC)

	var results []UnpaidMovement
	err := r.db.WithContext(ctx).
		Model(&MovementDB{}).
		Select("id, description, user_id").
		Where("date BETWEEN ? AND ?", startOfDay, endOfDay).
		Where("is_paid = ?", false).
		Find(&results).Error
	if err != nil {
		return nil, fmt.Errorf("error finding unpaid movements by date: %w: %s", ErrDatabaseError, err.Error())
	}

	return results, nil
}

func (r *MovementRepository) CountByUserIDAndMonth(ctx context.Context, year int, month time.Month) (int64, error) {
	userID := authentication.UserIDFromContext(ctx)
	if userID == "" {
		return 0, fmt.Errorf("error counting movements: %w: user_id not found in context", ErrDatabaseError)
	}

	firstDayOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	lastDayOfMonth := firstDayOfMonth.AddDate(0, 1, -1).Add(23*time.Hour + 59*time.Minute + 59*time.Second)

	var count int64
	err := r.db.WithContext(ctx).
		Model(&MovementDB{}).
		Where("user_id = ?", userID).
		Where("date BETWEEN ? AND ?", firstDayOfMonth, lastDayOfMonth).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("error counting movements: %w: %s", ErrDatabaseError, err.Error())
	}

	return count, nil
}

func (r *MovementRepository) FindExistingHashes(ctx context.Context, userID string, hashes []string) (map[string]bool, error) {
	if len(hashes) == 0 {
		return map[string]bool{}, nil
	}

	var results []struct {
		IdempotencyHash string `gorm:"column:idempotency_hash"`
	}

	err := r.db.WithContext(ctx).
		Model(&MovementDB{}).
		Select("idempotency_hash").
		Where("user_id = ? AND idempotency_hash IN ?", userID, hashes).
		Find(&results).Error
	if err != nil {
		return nil, fmt.Errorf("error finding existing hashes: %w: %s", ErrDatabaseError, err.Error())
	}

	existing := make(map[string]bool, len(results))
	for _, r := range results {
		existing[r.IdempotencyHash] = true
	}

	return existing, nil
}

func (r *MovementRepository) FindRecentCategorizedByNormalizedDescription(
	ctx context.Context,
	normalizedDesc string,
) (*uuid.UUID, *uuid.UUID, error) {
	userID := ctx.Value(authentication.UserID).(string)

	var result struct {
		CategoryID    *uuid.UUID `gorm:"column:category_id"`
		SubCategoryID *uuid.UUID `gorm:"column:sub_category_id"`
	}

	db := r.db.WithContext(ctx).
		Raw(`
			SELECT category_id, sub_category_id
			FROM movements
			WHERE user_id = ?
			  AND category_id IS NOT NULL
			  AND category_id::text != ?
			  AND lower(regexp_replace(description, '[^a-zA-Z0-9 ]', '', 'g')) LIKE '%' || ? || '%'
			GROUP BY category_id, sub_category_id
			ORDER BY COUNT(*) DESC
			LIMIT 1
		`, userID, domain.UncategorizedCategoryID, normalizedDesc).
		Scan(&result)

	if db.Error != nil {
		return nil, nil, fmt.Errorf("error finding categorized movement by description: %w: %s", ErrDatabaseError, db.Error.Error())
	}

	return result.CategoryID, result.SubCategoryID, nil
}
