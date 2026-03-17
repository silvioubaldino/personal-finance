package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"personal-finance/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AgentConversationRepository struct {
	db *gorm.DB
}

func NewAgentConversationRepository(db *gorm.DB) *AgentConversationRepository {
	return &AgentConversationRepository{db: db}
}

func (r *AgentConversationRepository) Save(ctx context.Context, conv domain.AgentConversation) (domain.AgentConversation, error) {
	dbModel := FromAgentConversationDomain(conv)
	err := r.db.WithContext(ctx).Create(&dbModel).Error
	if err != nil {
		return domain.AgentConversation{}, fmt.Errorf("error saving agent conversation: %w: %s", ErrDatabaseError, err.Error())
	}
	return dbModel.ToDomain(), nil
}

func (r *AgentConversationRepository) FindByID(ctx context.Context, id uuid.UUID) (domain.AgentConversation, error) {
	var dbModel AgentConversationDB
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&dbModel).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.AgentConversation{}, fmt.Errorf("agent conversation not found: %w", domain.ErrNotFound)
		}
		return domain.AgentConversation{}, fmt.Errorf("error finding agent conversation: %w: %s", ErrDatabaseError, err.Error())
	}

	conv := dbModel.ToDomain()

	// Load messages
	var msgModels []AgentMessageDB
	err = r.db.WithContext(ctx).
		Where("conversation_id = ?", id).
		Order("created_at ASC").
		Find(&msgModels).Error
	if err != nil {
		return domain.AgentConversation{}, fmt.Errorf("error loading messages: %w: %s", ErrDatabaseError, err.Error())
	}

	messages := make([]domain.AgentMessage, len(msgModels))
	for i, msg := range msgModels {
		messages[i] = msg.ToDomain()
	}
	conv.Messages = messages

	return conv, nil
}

func (r *AgentConversationRepository) FindByUserID(ctx context.Context, userID string) ([]domain.AgentConversation, error) {
	var dbModels []AgentConversationDB
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("updated_at DESC").
		Find(&dbModels).Error
	if err != nil {
		return nil, fmt.Errorf("error finding conversations: %w: %s", ErrDatabaseError, err.Error())
	}

	convs := make([]domain.AgentConversation, len(dbModels))
	for i, dbModel := range dbModels {
		convs[i] = dbModel.ToDomain()
	}
	return convs, nil
}

func (r *AgentConversationRepository) SaveMessage(ctx context.Context, msg domain.AgentMessage) (domain.AgentMessage, error) {
	dbModel := FromAgentMessageDomain(msg)
	err := r.db.WithContext(ctx).Create(&dbModel).Error
	if err != nil {
		return domain.AgentMessage{}, fmt.Errorf("error saving agent message: %w: %s", ErrDatabaseError, err.Error())
	}

	// Update conversation updated_at
	r.db.WithContext(ctx).
		Model(&AgentConversationDB{}).
		Where("id = ?", msg.ConversationID).
		Update("updated_at", time.Now())

	return dbModel.ToDomain(), nil
}

func (r *AgentConversationRepository) UpdateTitle(ctx context.Context, id uuid.UUID, title string) error {
	err := r.db.WithContext(ctx).
		Model(&AgentConversationDB{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"title":      title,
			"updated_at": time.Now(),
		}).Error
	if err != nil {
		return fmt.Errorf("error updating conversation title: %w: %s", ErrDatabaseError, err.Error())
	}
	return nil
}

func (r *AgentConversationRepository) DeleteExpired(ctx context.Context) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("expires_at < ?", time.Now()).
		Delete(&AgentConversationDB{})
	if result.Error != nil {
		return 0, fmt.Errorf("error deleting expired conversations: %w: %s", ErrDatabaseError, result.Error.Error())
	}
	return result.RowsAffected, nil
}

func (r *AgentConversationRepository) DeleteAllByUserID(ctx context.Context, tx *gorm.DB, userID string) error {
	db := r.db
	if tx != nil {
		db = tx
	}

	// Messages cascade via FK, but delete explicitly for safety
	err := db.WithContext(ctx).
		Where("conversation_id IN (SELECT id FROM agent_conversations WHERE user_id = ?)", userID).
		Delete(&AgentMessageDB{}).Error
	if err != nil {
		return fmt.Errorf("error deleting agent messages: %w: %s", ErrDatabaseError, err.Error())
	}

	err = db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&AgentConversationDB{}).Error
	if err != nil {
		return fmt.Errorf("error deleting agent conversations: %w: %s", ErrDatabaseError, err.Error())
	}
	return nil
}
