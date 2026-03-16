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

type AgentMemoryRepository struct {
	db *gorm.DB
}

func NewAgentMemoryRepository(db *gorm.DB) *AgentMemoryRepository {
	return &AgentMemoryRepository{db: db}
}

func (r *AgentMemoryRepository) Save(ctx context.Context, memory domain.AgentMemory) (domain.AgentMemory, error) {
	dbModel := FromAgentMemoryDomain(memory)
	err := r.db.WithContext(ctx).Create(&dbModel).Error
	if err != nil {
		return domain.AgentMemory{}, fmt.Errorf("error saving agent memory: %w: %s", ErrDatabaseError, err.Error())
	}
	return dbModel.ToDomain(), nil
}

func (r *AgentMemoryRepository) FindByID(ctx context.Context, id uuid.UUID) (domain.AgentMemory, error) {
	userID := authentication.UserIDFromContext(ctx)

	var dbModel AgentMemoryDB
	err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		First(&dbModel).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.AgentMemory{}, fmt.Errorf("agent memory not found: %w", domain.ErrAgentMemoryNotFound)
		}
		return domain.AgentMemory{}, fmt.Errorf("error finding agent memory: %w: %s", ErrDatabaseError, err.Error())
	}
	return dbModel.ToDomain(), nil
}

func (r *AgentMemoryRepository) FindByUserID(ctx context.Context, userID string) ([]domain.AgentMemory, error) {
	var dbModels []AgentMemoryDB
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND (expires_at IS NULL OR expires_at > ?)", userID, time.Now()).
		Order("created_at DESC").
		Find(&dbModels).Error
	if err != nil {
		return nil, fmt.Errorf("error finding agent memories: %w: %s", ErrDatabaseError, err.Error())
	}

	memories := make([]domain.AgentMemory, len(dbModels))
	for i, dbModel := range dbModels {
		memories[i] = dbModel.ToDomain()
	}
	return memories, nil
}

func (r *AgentMemoryRepository) FindByUserIDAndType(ctx context.Context, userID string, memType domain.AgentMemoryType) ([]domain.AgentMemory, error) {
	var dbModels []AgentMemoryDB
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND memory_type = ? AND (expires_at IS NULL OR expires_at > ?)", userID, string(memType), time.Now()).
		Order("created_at DESC").
		Find(&dbModels).Error
	if err != nil {
		return nil, fmt.Errorf("error finding agent memories by type: %w: %s", ErrDatabaseError, err.Error())
	}

	memories := make([]domain.AgentMemory, len(dbModels))
	for i, dbModel := range dbModels {
		memories[i] = dbModel.ToDomain()
	}
	return memories, nil
}

func (r *AgentMemoryRepository) SearchByContent(ctx context.Context, userID string, query string) ([]domain.AgentMemory, error) {
	var dbModels []AgentMemoryDB
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND content ILIKE ? AND (expires_at IS NULL OR expires_at > ?)", userID, "%"+query+"%", time.Now()).
		Order("updated_at DESC").
		Limit(10).
		Find(&dbModels).Error
	if err != nil {
		return nil, fmt.Errorf("error searching agent memories: %w: %s", ErrDatabaseError, err.Error())
	}

	memories := make([]domain.AgentMemory, len(dbModels))
	for i, dbModel := range dbModels {
		memories[i] = dbModel.ToDomain()
	}
	return memories, nil
}

func (r *AgentMemoryRepository) Update(ctx context.Context, memory domain.AgentMemory) (domain.AgentMemory, error) {
	dbModel := FromAgentMemoryDomain(memory)
	err := r.db.WithContext(ctx).
		Model(&AgentMemoryDB{}).
		Where("id = ? AND user_id = ?", memory.ID, memory.UserID).
		Updates(map[string]interface{}{
			"content":        dbModel.Content,
			"metadata":       dbModel.Metadata,
			"confidence":     dbModel.Confidence,
			"updated_at":     time.Now(),
			"last_validated": dbModel.LastValidated,
			"expires_at":     dbModel.ExpiresAt,
		}).Error
	if err != nil {
		return domain.AgentMemory{}, fmt.Errorf("error updating agent memory: %w: %s", ErrDatabaseError, err.Error())
	}
	return r.FindByID(ctx, memory.ID)
}

func (r *AgentMemoryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	userID := authentication.UserIDFromContext(ctx)

	result := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&AgentMemoryDB{})
	if result.Error != nil {
		return fmt.Errorf("error deleting agent memory: %w: %s", ErrDatabaseError, result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("agent memory not found: %w", domain.ErrAgentMemoryNotFound)
	}
	return nil
}

func (r *AgentMemoryRepository) CountByUserID(ctx context.Context, userID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&AgentMemoryDB{}).
		Where("user_id = ? AND (expires_at IS NULL OR expires_at > ?)", userID, time.Now()).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("error counting agent memories: %w: %s", ErrDatabaseError, err.Error())
	}
	return count, nil
}

func (r *AgentMemoryRepository) DeleteExpired(ctx context.Context) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("expires_at IS NOT NULL AND expires_at < ?", time.Now()).
		Delete(&AgentMemoryDB{})
	if result.Error != nil {
		return 0, fmt.Errorf("error deleting expired memories: %w: %s", ErrDatabaseError, result.Error.Error())
	}
	return result.RowsAffected, nil
}

// UpsertRiskProfile ensures only one risk_profile exists per user.
func (r *AgentMemoryRepository) UpsertRiskProfile(ctx context.Context, memory domain.AgentMemory) (domain.AgentMemory, error) {
	userID := memory.UserID

	var existing AgentMemoryDB
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND memory_type = ?", userID, string(domain.MemoryTypeRiskProfile)).
		First(&existing).Error

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.AgentMemory{}, fmt.Errorf("error finding risk profile: %w: %s", ErrDatabaseError, err.Error())
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r.Save(ctx, memory)
	}

	memory.ID = *existing.ID
	memory.CreatedAt = existing.CreatedAt
	return r.Update(ctx, memory)
}

func (r *AgentMemoryRepository) DeleteAllByUserID(ctx context.Context, tx *gorm.DB, userID string) error {
	db := r.db
	if tx != nil {
		db = tx
	}
	err := db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&AgentMemoryDB{}).Error
	if err != nil {
		return fmt.Errorf("error deleting all agent memories: %w: %s", ErrDatabaseError, err.Error())
	}
	return nil
}
