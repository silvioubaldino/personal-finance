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

var (
	ErrUserPreferencesNotFound = errors.New("user preferences not found")
)

type UserPreferencesRepository struct {
	db *gorm.DB
}

func NewUserPreferencesRepository(db *gorm.DB) *UserPreferencesRepository {
	return &UserPreferencesRepository{
		db: db,
	}
}

func (r *UserPreferencesRepository) GetOrCreateDefaults(ctx context.Context) (domain.UserPreferences, error) {
	userID := ctx.Value(authentication.UserID).(string)
	now := time.Now()

	defaults := UserPreferencesDB{
		UserID:     userID,
		Language:   domain.DefaultLanguage,
		Currency:   domain.DefaultCurrency,
		DateCreate: now,
		DateUpdate: now,
	}

	// Insert defaults if not exists (ON CONFLICT DO NOTHING)
	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&defaults).Error
	if err != nil {
		return domain.UserPreferences{}, fmt.Errorf("error creating default preferences: %w: %s", ErrDatabaseError, err.Error())
	}

	// Read current preferences
	var dbModel UserPreferencesDB
	err = r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		First(&dbModel).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.UserPreferences{}, fmt.Errorf("error finding preferences: %w: %s", ErrUserPreferencesNotFound, err.Error())
		}
		return domain.UserPreferences{}, fmt.Errorf("error finding preferences: %w: %s", ErrDatabaseError, err.Error())
	}

	return dbModel.ToDomain(), nil
}

func (r *UserPreferencesRepository) Upsert(ctx context.Context, prefs domain.UserPreferences) (domain.UserPreferences, error) {
	userID := ctx.Value(authentication.UserID).(string)
	now := time.Now()

	updateColumns := []string{"date_update"}

	language := prefs.Language
	if language == "" {
		language = domain.DefaultLanguage
	} else {
		updateColumns = append(updateColumns, "language")
	}

	currency := prefs.Currency
	if currency == "" {
		currency = domain.DefaultCurrency
	} else {
		updateColumns = append(updateColumns, "currency")
	}

	dbModel := UserPreferencesDB{
		UserID:     userID,
		Language:   language,
		Currency:   currency,
		DateCreate: now,
		DateUpdate: now,
	}

	// Upsert: insert or update on conflict
	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}},
			DoUpdates: clause.AssignmentColumns(updateColumns),
		}).
		Create(&dbModel).Error
	if err != nil {
		return domain.UserPreferences{}, fmt.Errorf("error upserting preferences: %w: %s", ErrDatabaseError, err.Error())
	}

	// Read back to get actual values (including date_create if it was existing)
	var result UserPreferencesDB
	err = r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		First(&result).Error
	if err != nil {
		return domain.UserPreferences{}, fmt.Errorf("error reading preferences after upsert: %w: %s", ErrDatabaseError, err.Error())
	}

	return result.ToDomain(), nil
}
