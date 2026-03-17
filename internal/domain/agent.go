package domain

import (
	"time"

	"github.com/google/uuid"
)

// --- Agent Memory Types ---

type AgentMemoryType string

const (
	MemoryTypeGoal        AgentMemoryType = "goal"
	MemoryTypeFact        AgentMemoryType = "fact"
	MemoryTypeConstraint  AgentMemoryType = "constraint"
	MemoryTypeInsight     AgentMemoryType = "insight"
	MemoryTypeCommitment  AgentMemoryType = "commitment"
	MemoryTypeRiskProfile AgentMemoryType = "risk_profile"
	MemoryTypeLifeEvent   AgentMemoryType = "life_event"
)

func (t AgentMemoryType) IsValid() bool {
	switch t {
	case MemoryTypeGoal, MemoryTypeFact, MemoryTypeConstraint,
		MemoryTypeInsight, MemoryTypeCommitment, MemoryTypeRiskProfile,
		MemoryTypeLifeEvent:
		return true
	}
	return false
}

type AgentMemorySource string

const (
	MemorySourceExplicit AgentMemorySource = "explicit"
	MemorySourceElicited AgentMemorySource = "elicited"
	MemorySourceDerived  AgentMemorySource = "derived"
)

// --- Agent Memory ---

type AgentMemory struct {
	ID            uuid.UUID         `json:"id"`
	UserID        string            `json:"user_id"`
	Type          AgentMemoryType   `json:"memory_type"`
	Content       string            `json:"content"`
	Metadata      map[string]any    `json:"metadata,omitempty"`
	Source        AgentMemorySource `json:"source"`
	Confidence    string            `json:"confidence"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
	LastValidated time.Time         `json:"last_validated"`
	ExpiresAt     *time.Time        `json:"expires_at,omitempty"`
}

func NewAgentMemory(userID string, memType AgentMemoryType, content string, source AgentMemorySource) AgentMemory {
	now := time.Now()
	m := AgentMemory{
		ID:            uuid.New(),
		UserID:        userID,
		Type:          memType,
		Content:       content,
		Metadata:      make(map[string]any),
		Source:        source,
		Confidence:    "high",
		CreatedAt:     now,
		UpdatedAt:     now,
		LastValidated: now,
	}
	m.ExpiresAt = m.DefaultExpiry()
	return m
}

// DefaultExpiry returns the recommended expiry based on memory type.
func (m *AgentMemory) DefaultExpiry() *time.Time {
	now := time.Now()
	switch m.Type {
	case MemoryTypeGoal:
		t := now.AddDate(1, 0, 0)
		return &t
	case MemoryTypeInsight:
		t := now.AddDate(0, 3, 0) // 90 days, force revalidation
		return &t
	case MemoryTypeCommitment:
		t := now.AddDate(0, 3, 0)
		return &t
	case MemoryTypeLifeEvent:
		t := now.AddDate(1, 0, 0)
		return &t
	case MemoryTypeFact, MemoryTypeConstraint, MemoryTypeRiskProfile:
		return nil // permanent until contradicted
	}
	return nil
}

// --- Agent Conversation ---

type AgentConversation struct {
	ID        uuid.UUID      `json:"id"`
	UserID    string         `json:"user_id"`
	Title     string         `json:"title,omitempty"`
	Messages  []AgentMessage `json:"messages,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	ExpiresAt time.Time      `json:"expires_at"`
}

func NewAgentConversation(userID string) AgentConversation {
	now := time.Now()
	return AgentConversation{
		ID:        uuid.New(),
		UserID:    userID,
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: now.AddDate(0, 0, 30),
	}
}

// --- Agent Message ---

type AgentMessage struct {
	ID             uuid.UUID `json:"id"`
	ConversationID uuid.UUID `json:"conversation_id"`
	Role           string    `json:"role"` // "user" | "assistant"
	Content        string    `json:"content"`
	CreatedAt      time.Time `json:"created_at"`
}

func NewAgentMessage(conversationID uuid.UUID, role, content string) AgentMessage {
	return AgentMessage{
		ID:             uuid.New(),
		ConversationID: conversationID,
		Role:           role,
		Content:        content,
		CreatedAt:      time.Now(),
	}
}

// --- Agent Audit Record ---

type AgentAuditRecord struct {
	ID             uuid.UUID  `json:"id"`
	UserID         string     `json:"user_id"`
	ConversationID *uuid.UUID `json:"conversation_id,omitempty"`
	ToolsCalled    []string   `json:"tools_called,omitempty"`
	InputTokens    int        `json:"input_tokens"`
	OutputTokens   int        `json:"output_tokens"`
	Provider       string     `json:"provider"`
	Region         string     `json:"region"`
	CreatedAt      time.Time  `json:"created_at"`
}

// --- Agent Gateway Response ---

type AgentGatewayResponse struct {
	Content     string   `json:"content"`
	ToolsCalled []string `json:"tools_called,omitempty"`
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
}

// --- Interfaces (Ports) ---

const (
	MaxMemoriesPerUser    = 50
	MaxMemoriesPerPrompt  = 15
)
