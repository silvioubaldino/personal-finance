package repository

import (
	"encoding/json"
	"time"

	"personal-finance/internal/domain"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// --- Agent Memory DB Model ---

type AgentMemoryDB struct {
	ID            *uuid.UUID `gorm:"primaryKey"`
	UserID        string     `gorm:"user_id"`
	MemoryType    string     `gorm:"memory_type"`
	Content       string     `gorm:"content"`
	Metadata      []byte     `gorm:"type:jsonb;default:'{}'"`
	Source        string     `gorm:"source"`
	Confidence    string     `gorm:"confidence"`
	CreatedAt     time.Time  `gorm:"created_at"`
	UpdatedAt     time.Time  `gorm:"updated_at"`
	LastValidated time.Time  `gorm:"last_validated"`
	ExpiresAt     *time.Time `gorm:"expires_at"`
}

func (AgentMemoryDB) TableName() string {
	return "agent_memories"
}

func (m AgentMemoryDB) ToDomain() domain.AgentMemory {
	id := uuid.Nil
	if m.ID != nil {
		id = *m.ID
	}

	var metadata map[string]any
	if len(m.Metadata) > 0 {
		_ = json.Unmarshal(m.Metadata, &metadata)
	}
	if metadata == nil {
		metadata = make(map[string]any)
	}

	return domain.AgentMemory{
		ID:            id,
		UserID:        m.UserID,
		Type:          domain.AgentMemoryType(m.MemoryType),
		Content:       m.Content,
		Metadata:      metadata,
		Source:        domain.AgentMemorySource(m.Source),
		Confidence:    m.Confidence,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
		LastValidated: m.LastValidated,
		ExpiresAt:     m.ExpiresAt,
	}
}

func FromAgentMemoryDomain(d domain.AgentMemory) AgentMemoryDB {
	metadataBytes, _ := json.Marshal(d.Metadata)

	return AgentMemoryDB{
		ID:            &d.ID,
		UserID:        d.UserID,
		MemoryType:    string(d.Type),
		Content:       d.Content,
		Metadata:      metadataBytes,
		Source:        string(d.Source),
		Confidence:    d.Confidence,
		CreatedAt:     d.CreatedAt,
		UpdatedAt:     d.UpdatedAt,
		LastValidated: d.LastValidated,
		ExpiresAt:     d.ExpiresAt,
	}
}

// --- Agent Conversation DB Model ---

type AgentConversationDB struct {
	ID        *uuid.UUID `gorm:"primaryKey"`
	UserID    string     `gorm:"user_id"`
	Title     string     `gorm:"title"`
	CreatedAt time.Time  `gorm:"created_at"`
	UpdatedAt time.Time  `gorm:"updated_at"`
	ExpiresAt time.Time  `gorm:"expires_at"`
}

func (AgentConversationDB) TableName() string {
	return "agent_conversations"
}

func (c AgentConversationDB) ToDomain() domain.AgentConversation {
	id := uuid.Nil
	if c.ID != nil {
		id = *c.ID
	}
	return domain.AgentConversation{
		ID:        id,
		UserID:    c.UserID,
		Title:     c.Title,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
		ExpiresAt: c.ExpiresAt,
	}
}

func FromAgentConversationDomain(d domain.AgentConversation) AgentConversationDB {
	return AgentConversationDB{
		ID:        &d.ID,
		UserID:    d.UserID,
		Title:     d.Title,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
		ExpiresAt: d.ExpiresAt,
	}
}

// --- Agent Message DB Model ---

type AgentMessageDB struct {
	ID             *uuid.UUID `gorm:"primaryKey"`
	ConversationID *uuid.UUID `gorm:"conversation_id"`
	Role           string     `gorm:"role"`
	Content        string     `gorm:"content"`
	CreatedAt      time.Time  `gorm:"created_at"`
}

func (AgentMessageDB) TableName() string {
	return "agent_messages"
}

func (m AgentMessageDB) ToDomain() domain.AgentMessage {
	id := uuid.Nil
	if m.ID != nil {
		id = *m.ID
	}
	convID := uuid.Nil
	if m.ConversationID != nil {
		convID = *m.ConversationID
	}
	return domain.AgentMessage{
		ID:             id,
		ConversationID: convID,
		Role:           m.Role,
		Content:        m.Content,
		CreatedAt:      m.CreatedAt,
	}
}

func FromAgentMessageDomain(d domain.AgentMessage) AgentMessageDB {
	return AgentMessageDB{
		ID:             &d.ID,
		ConversationID: &d.ConversationID,
		Role:           d.Role,
		Content:        d.Content,
		CreatedAt:      d.CreatedAt,
	}
}

// --- Agent Audit Log DB Model ---

type AgentAuditLogDB struct {
	ID             *uuid.UUID     `gorm:"primaryKey"`
	UserID         string         `gorm:"user_id"`
	ConversationID *uuid.UUID     `gorm:"conversation_id"`
	ToolsCalled    pq.StringArray `gorm:"type:text[];tools_called"`
	InputTokens    int            `gorm:"input_tokens"`
	OutputTokens   int            `gorm:"output_tokens"`
	Provider       string         `gorm:"provider"`
	Region         string         `gorm:"region"`
	CreatedAt      time.Time      `gorm:"created_at"`
}

func (AgentAuditLogDB) TableName() string {
	return "agent_audit_log"
}

func FromAgentAuditRecordDomain(d domain.AgentAuditRecord) AgentAuditLogDB {
	return AgentAuditLogDB{
		ID:             &d.ID,
		UserID:         d.UserID,
		ConversationID: d.ConversationID,
		ToolsCalled:    d.ToolsCalled,
		InputTokens:    d.InputTokens,
		OutputTokens:   d.OutputTokens,
		Provider:       d.Provider,
		Region:         d.Region,
		CreatedAt:      d.CreatedAt,
	}
}
