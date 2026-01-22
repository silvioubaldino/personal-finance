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
	"gorm.io/gorm/clause"
)

var (
	ErrDeviceNotFound = errors.New("device not found")
)

type DeviceRepository struct {
	db *gorm.DB
}

func NewDeviceRepository(db *gorm.DB) *DeviceRepository {
	return &DeviceRepository{
		db: db,
	}
}

func (r *DeviceRepository) Upsert(ctx context.Context, device domain.Device) (domain.Device, error) {
	userID := ctx.Value(authentication.UserID).(string)
	now := time.Now()

	dbModel := UserDeviceDB{
		ID:            uuid.New(),
		UserID:        userID,
		ExpoPushToken: device.ExpoPushToken,
		Platform:      string(device.Platform),
		DateCreate:    now,
		DateUpdate:    now,
		LastSeenAt:    &now,
	}

	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "expo_push_token"}},
			DoUpdates: clause.AssignmentColumns([]string{"user_id", "platform", "date_update", "last_seen_at"}),
		}).
		Create(&dbModel).Error
	if err != nil {
		return domain.Device{}, fmt.Errorf("error upserting device: %w: %s", ErrDatabaseError, err.Error())
	}

	var result UserDeviceDB
	err = r.db.WithContext(ctx).
		Where("expo_push_token = ?", device.ExpoPushToken).
		First(&result).Error
	if err != nil {
		return domain.Device{}, fmt.Errorf("error reading device after upsert: %w: %s", ErrDatabaseError, err.Error())
	}

	return result.ToDomain(), nil
}

func (r *DeviceRepository) FindByUserID(ctx context.Context) ([]domain.Device, error) {
	userID := ctx.Value(authentication.UserID).(string)

	var dbModels []UserDeviceDB
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("date_create DESC").
		Find(&dbModels).Error
	if err != nil {
		return nil, fmt.Errorf("error finding devices: %w: %s", ErrDatabaseError, err.Error())
	}

	devices := make([]domain.Device, len(dbModels))
	for i, m := range dbModels {
		devices[i] = m.ToDomain()
	}

	return devices, nil
}

func (r *DeviceRepository) DeleteByToken(ctx context.Context, token string) error {
	userID := ctx.Value(authentication.UserID).(string)

	result := r.db.WithContext(ctx).
		Where("expo_push_token = ? AND user_id = ?", token, userID).
		Delete(&UserDeviceDB{})
	if result.Error != nil {
		return fmt.Errorf("error deleting device: %w: %s", ErrDatabaseError, result.Error.Error())
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("error deleting device: %w", ErrDeviceNotFound)
	}

	return nil
}

func (r *DeviceRepository) FindByUserIDs(ctx context.Context, userIDs []string) ([]domain.Device, error) {
	if len(userIDs) == 0 {
		return []domain.Device{}, nil
	}

	var dbModels []UserDeviceDB
	err := r.db.WithContext(ctx).
		Where("user_id IN ?", userIDs).
		Find(&dbModels).Error
	if err != nil {
		return nil, fmt.Errorf("error finding devices by user ids: %w: %s", ErrDatabaseError, err.Error())
	}

	devices := make([]domain.Device, len(dbModels))
	for i, m := range dbModels {
		devices[i] = m.ToDomain()
	}

	return devices, nil
}

func (r *DeviceRepository) DeleteByTokens(ctx context.Context, tokens []string) error {
	if len(tokens) == 0 {
		return nil
	}

	err := r.db.WithContext(ctx).
		Where("expo_push_token IN ?", tokens).
		Delete(&UserDeviceDB{}).Error
	if err != nil {
		return fmt.Errorf("error deleting devices by tokens: %w: %s", ErrDatabaseError, err.Error())
	}

	return nil
}

func (r *DeviceRepository) DeleteByUserID(ctx context.Context, tx *gorm.DB, userID string) error {
	db := r.db
	if tx != nil {
		db = tx
	}

	err := db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&UserDeviceDB{}).Error
	if err != nil {
		return fmt.Errorf("error deleting devices: %w: %s", ErrDatabaseError, err.Error())
	}

	return nil
}
