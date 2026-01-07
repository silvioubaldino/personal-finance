package repository

import (
	"context"
	"errors"
	"fmt"

	"personal-finance/internal/domain"
	"personal-finance/internal/plataform/authentication"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrUserConsentNotFound = errors.New("user consent not found")
)

type UserConsentRepository struct {
	db *gorm.DB
}

func NewUserConsentRepository(db *gorm.DB) *UserConsentRepository {
	return &UserConsentRepository{
		db: db,
	}
}

func (r *UserConsentRepository) Save(ctx context.Context, consent domain.UserConsent) (domain.UserConsent, error) {
	dbModel := FromUserConsentDomain(consent)

	err := r.db.WithContext(ctx).Create(&dbModel).Error
	if err != nil {
		return domain.UserConsent{}, fmt.Errorf("error saving consent: %w: %s", ErrDatabaseError, err.Error())
	}

	return dbModel.ToDomain(), nil
}

func (r *UserConsentRepository) FindByUserID(ctx context.Context) ([]domain.UserConsent, error) {
	userID := ctx.Value(authentication.UserID).(string)

	var dbModels []UserConsentDB
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("agreed_at DESC").
		Find(&dbModels).Error
	if err != nil {
		return nil, fmt.Errorf("error finding consents: %w: %s", ErrDatabaseError, err.Error())
	}

	consents := make([]domain.UserConsent, len(dbModels))
	for i, dbModel := range dbModels {
		consents[i] = dbModel.ToDomain()
	}

	return consents, nil
}

func (r *UserConsentRepository) FindLatestByTermVersion(ctx context.Context, termVersion string) (domain.UserConsent, error) {
	userID := ctx.Value(authentication.UserID).(string)

	var dbModel UserConsentDB
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND term_version = ?", userID, termVersion).
		Order("agreed_at DESC").
		First(&dbModel).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.UserConsent{}, fmt.Errorf("consent not found: %w: %s", ErrUserConsentNotFound, err.Error())
		}
		return domain.UserConsent{}, fmt.Errorf("error finding consent: %w: %s", ErrDatabaseError, err.Error())
	}

	return dbModel.ToDomain(), nil
}

func (r *UserConsentRepository) HasConsentedToVersion(ctx context.Context, termVersion string) (bool, error) {
	userID := ctx.Value(authentication.UserID).(string)

	var count int64
	err := r.db.WithContext(ctx).
		Model(&UserConsentDB{}).
		Where("user_id = ? AND term_version = ?", userID, termVersion).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("error checking consent: %w: %s", ErrDatabaseError, err.Error())
	}

	return count > 0, nil
}

func (r *UserConsentRepository) DeleteAllByUserID(ctx context.Context, tx *gorm.DB, userID string) error {
	db := r.db
	if tx != nil {
		db = tx
	}

	err := db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&UserConsentDB{}).Error
	if err != nil {
		return fmt.Errorf("error deleting consents: %w: %s", ErrDatabaseError, err.Error())
	}

	return nil
}

func (r *UserConsentRepository) FindByID(ctx context.Context, id uuid.UUID) (domain.UserConsent, error) {
	userID := ctx.Value(authentication.UserID).(string)

	var dbModel UserConsentDB
	err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		First(&dbModel).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.UserConsent{}, fmt.Errorf("consent not found: %w: %s", ErrUserConsentNotFound, err.Error())
		}
		return domain.UserConsent{}, fmt.Errorf("error finding consent: %w: %s", ErrDatabaseError, err.Error())
	}

	return dbModel.ToDomain(), nil
}
