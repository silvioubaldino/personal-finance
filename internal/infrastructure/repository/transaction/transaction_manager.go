package transaction

import (
	"context"

	"gorm.io/gorm"
)

type Manager interface {
	WithTransaction(ctx context.Context, fn func(tx *gorm.DB) error) error
}

type GormManager struct {
	db *gorm.DB
}

func NewGormManager(db *gorm.DB) Manager {
	return &GormManager{
		db: db,
	}
}

func (m *GormManager) WithTransaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	tx := m.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}
