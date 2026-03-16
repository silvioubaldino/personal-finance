package repository

import (
	"context"
	"fmt"

	"personal-finance/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AgentAuditRepository struct {
	db *gorm.DB
}

func NewAgentAuditRepository(db *gorm.DB) *AgentAuditRepository {
	return &AgentAuditRepository{db: db}
}

func (r *AgentAuditRepository) Log(ctx context.Context, record domain.AgentAuditRecord) error {
	record.ID = uuid.New()
	dbModel := FromAgentAuditRecordDomain(record)
	err := r.db.WithContext(ctx).Create(&dbModel).Error
	if err != nil {
		return fmt.Errorf("error saving audit log: %w: %s", ErrDatabaseError, err.Error())
	}
	return nil
}

func (r *AgentAuditRepository) DeleteAllByUserID(ctx context.Context, tx *gorm.DB, userID string) error {
	db := r.db
	if tx != nil {
		db = tx
	}
	err := db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&AgentAuditLogDB{}).Error
	if err != nil {
		return fmt.Errorf("error deleting audit logs: %w: %s", ErrDatabaseError, err.Error())
	}
	return nil
}
