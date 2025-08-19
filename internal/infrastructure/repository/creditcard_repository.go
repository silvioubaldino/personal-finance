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

var (
	ErrCreditCardNotFound = errors.New("credit card not found")
)

type CreditCardRepository struct {
	db *gorm.DB
}

func NewCreditCardRepository(db *gorm.DB) *CreditCardRepository {
	return &CreditCardRepository{
		db: db,
	}
}

func (r *CreditCardRepository) Add(ctx context.Context, tx *gorm.DB, creditCard domain.CreditCard) (domain.CreditCard, error) {
	var isLocalTx bool
	if tx == nil {
		isLocalTx = true
		tx = r.db.WithContext(ctx).Begin()
		defer tx.Rollback()
	}

	userID := ctx.Value(authentication.UserID).(string)
	now := time.Now()
	id := uuid.New()

	creditCard.ID = &id
	creditCard.DateCreate = now
	creditCard.DateUpdate = now
	creditCard.UserID = userID

	dbCreditCard := FromCreditCardDomain(creditCard)

	if err := tx.WithContext(ctx).Create(&dbCreditCard).Error; err != nil {
		return domain.CreditCard{}, fmt.Errorf("error creating credit card: %w: %s", ErrDatabaseError, err.Error())
	}

	if isLocalTx {
		if err := tx.Commit().Error; err != nil {
			return domain.CreditCard{}, fmt.Errorf("error committing transaction: %w: %s", ErrDatabaseError, err.Error())
		}
	}

	return dbCreditCard.ToDomain(), nil
}

func (r *CreditCardRepository) FindByID(ctx context.Context, id uuid.UUID) (domain.CreditCard, error) {
	var dbModel CreditCardDB
	tableName := dbModel.TableName()

	query := BuildBaseQuery(ctx, r.db, tableName)
	query = r.appendPreloads(query)

	if err := query.First(&dbModel, fmt.Sprintf("%s.id = ?", tableName), id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.CreditCard{}, fmt.Errorf("error finding credit card: %w: %s", ErrCreditCardNotFound, err.Error())
		}
		return domain.CreditCard{}, fmt.Errorf("error finding credit card: %w: %s", ErrDatabaseError, err.Error())
	}

	return dbModel.ToDomain(), nil
}

func (r *CreditCardRepository) FindNameByID(ctx context.Context, id uuid.UUID) (string, error) {
	var name string
	tableName := "credit_cards"

	query := BuildBaseQuery(ctx, r.db, tableName)

	if err := query.Select("name").Where(fmt.Sprintf("%s.id = ?", tableName), id).Scan(&name).Error; err != nil {
		return "", fmt.Errorf("error finding credit card: %w: %s", ErrDatabaseError, err.Error())
	}

	if name == "" {
		return "", fmt.Errorf("error finding credit card: %w", ErrCreditCardNotFound)
	}

	return name, nil
}

func (r *CreditCardRepository) FindAll(ctx context.Context) ([]domain.CreditCard, error) {
	var dbModel CreditCardDB
	tableName := dbModel.TableName()

	query := BuildBaseQuery(ctx, r.db, tableName)
	query = r.appendPreloads(query)

	var dbCreditCards []CreditCardDB
	err := query.Find(&dbCreditCards).Error
	if err != nil {
		return nil, fmt.Errorf("error finding credit cards: %w: %s", ErrDatabaseError, err.Error())
	}

	creditCards := make([]domain.CreditCard, len(dbCreditCards))
	for i, dbCreditCard := range dbCreditCards {
		creditCards[i] = dbCreditCard.ToDomain()
	}

	return creditCards, nil
}

func (r *CreditCardRepository) Update(ctx context.Context, tx *gorm.DB, id uuid.UUID, creditCard domain.CreditCard) (domain.CreditCard, error) {
	var isLocalTx bool
	if tx == nil {
		isLocalTx = true
		tx = r.db.WithContext(ctx).Begin()
		defer tx.Rollback()
	}

	var dbModel CreditCardDB
	tableName := dbModel.TableName()

	query := BuildBaseQuery(ctx, tx, tableName)
	if err := query.First(&dbModel, fmt.Sprintf("%s.id = ?", tableName), id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.CreditCard{}, fmt.Errorf("error finding credit card: %w: %s", ErrCreditCardNotFound, err.Error())
		}
		return domain.CreditCard{}, fmt.Errorf("error finding credit card: %w: %s", ErrDatabaseError, err.Error())
	}

	now := time.Now()
	creditCard.DateUpdate = now
	creditCard.DateCreate = dbModel.DateCreate
	creditCard.UserID = dbModel.UserID
	creditCard.ID = dbModel.ID

	dbCreditCard := FromCreditCardDomain(creditCard)

	if err := tx.WithContext(ctx).Save(&dbCreditCard).Error; err != nil {
		return domain.CreditCard{}, fmt.Errorf("error updating credit card: %w: %s", ErrDatabaseError, err.Error())
	}

	if isLocalTx {
		if err := tx.Commit().Error; err != nil {
			return domain.CreditCard{}, fmt.Errorf("error committing transaction: %w: %s", ErrDatabaseError, err.Error())
		}
	}

	return dbCreditCard.ToDomain(), nil
}

func (r *CreditCardRepository) Delete(ctx context.Context, tx *gorm.DB, id uuid.UUID) error {
	var isLocalTx bool
	if tx == nil {
		isLocalTx = true
		tx = r.db.WithContext(ctx).Begin()
		defer tx.Rollback()
	}

	var dbModel CreditCardDB
	tableName := dbModel.TableName()

	query := BuildBaseQuery(ctx, tx, tableName)
	if err := query.First(&dbModel, fmt.Sprintf("%s.id = ?", tableName), id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("error finding credit card: %w: %s", ErrCreditCardNotFound, err.Error())
		}
		return fmt.Errorf("error finding credit card: %w: %s", ErrDatabaseError, err.Error())
	}

	if err := tx.WithContext(ctx).Delete(&dbModel).Error; err != nil {
		return fmt.Errorf("error deleting credit card: %w: %s", ErrDatabaseError, err.Error())
	}

	if isLocalTx {
		if err := tx.Commit().Error; err != nil {
			return fmt.Errorf("error committing transaction: %w: %s", ErrDatabaseError, err.Error())
		}
	}

	return nil
}

func (r *CreditCardRepository) appendPreloads(query *gorm.DB) *gorm.DB {
	return query.Preload("DefaultWallet")
}
