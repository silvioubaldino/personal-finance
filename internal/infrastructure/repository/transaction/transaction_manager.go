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
	return &GormManager{db: db}
}

func (m *GormManager) WithTransaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(tx)
	})
}
