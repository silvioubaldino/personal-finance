package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/plataform/authentication"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrUserNotFound = errors.New("user not found")

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) EnsureExists(ctx context.Context, userID string) (bool, error) {
	now := time.Now()
	user := UserDB{
		ID:        userID,
		Language:  domain.DefaultLanguage,
		Currency:  domain.DefaultCurrency,
		CreatedAt: now,
		UpdatedAt: now,
	}

	result := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&user)
	if result.Error != nil {
		return false, fmt.Errorf("error ensuring user exists: %w: %s", ErrDatabaseError, result.Error.Error())
	}
	return result.RowsAffected > 0, nil
}

func (r *UserRepository) Get(ctx context.Context) (domain.User, error) {
	userID := ctx.Value(authentication.UserID).(string)

	if _, err := r.EnsureExists(ctx, userID); err != nil {
		return domain.User{}, err
	}

	var dbModel UserDB
	err := r.db.WithContext(ctx).
		Where("id = ?", userID).
		First(&dbModel).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.User{}, fmt.Errorf("error finding user: %w: %s", ErrUserNotFound, err.Error())
		}
		return domain.User{}, fmt.Errorf("error finding user: %w: %s", ErrDatabaseError, err.Error())
	}

	return dbModel.ToDomain(), nil
}

func (r *UserRepository) Update(ctx context.Context, user domain.User) (domain.User, error) {
	userID := ctx.Value(authentication.UserID).(string)
	now := time.Now()

	updateColumns := []string{"updated_at"}

	language := user.Language
	if language == "" {
		language = domain.DefaultLanguage
	} else {
		updateColumns = append(updateColumns, "language")
	}

	currency := user.Currency
	if currency == "" {
		currency = domain.DefaultCurrency
	} else {
		updateColumns = append(updateColumns, "currency")
	}

	dbModel := UserDB{
		ID:        userID,
		Language:  language,
		Currency:  currency,
		CreatedAt: now,
		UpdatedAt: now,
	}

	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns(updateColumns),
		}).
		Create(&dbModel).Error
	if err != nil {
		return domain.User{}, fmt.Errorf("error updating user: %w: %s", ErrDatabaseError, err.Error())
	}

	var result UserDB
	err = r.db.WithContext(ctx).
		Where("id = ?", userID).
		First(&result).Error
	if err != nil {
		return domain.User{}, fmt.Errorf("error reading user after update: %w: %s", ErrDatabaseError, err.Error())
	}

	return result.ToDomain(), nil
}

func (r *UserRepository) Delete(ctx context.Context, tx *gorm.DB, userID string) error {
	db := r.db
	if tx != nil {
		db = tx
	}

	err := db.WithContext(ctx).
		Where("id = ?", userID).
		Delete(&UserDB{}).Error
	if err != nil {
		return fmt.Errorf("error deleting user: %w: %s", ErrDatabaseError, err.Error())
	}

	return nil
}
