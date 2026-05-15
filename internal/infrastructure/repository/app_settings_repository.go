package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrSettingNotFound = errors.New("setting not found")

type AppSettingsRepository struct {
	db *gorm.DB
}

func NewAppSettingsRepository(db *gorm.DB) *AppSettingsRepository {
	return &AppSettingsRepository{db: db}
}

func (r *AppSettingsRepository) GetFloat(ctx context.Context, key string) (float64, error) {
	var row AppSettingsDB
	err := r.db.WithContext(ctx).Where("key = ?", key).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, fmt.Errorf("%w: %s", ErrSettingNotFound, key)
		}
		return 0, fmt.Errorf("error reading setting %s: %w: %s", key, ErrDatabaseError, err.Error())
	}
	v, err := strconv.ParseFloat(row.Value, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid float value for setting %s: %w", key, ErrDatabaseError)
	}
	return v, nil
}

func (r *AppSettingsRepository) SetFloat(ctx context.Context, key string, value float64) error {
	row := AppSettingsDB{
		Key:       key,
		Value:     strconv.FormatFloat(value, 'f', -1, 64),
		UpdatedAt: time.Now(),
	}
	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
		}).
		Create(&row).Error
	if err != nil {
		return fmt.Errorf("error setting %s: %w: %s", key, ErrDatabaseError, err.Error())
	}
	return nil
}
