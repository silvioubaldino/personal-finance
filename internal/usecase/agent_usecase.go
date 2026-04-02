package usecase

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/plataform/authentication"

	"github.com/google/uuid"
)

const maxConversationTitleRunes = 80

// --- Interfaces ---

type AgentMemoryRepository interface {
	Save(ctx context.Context, memory domain.AgentMemory) (domain.AgentMemory, error)
	FindByID(ctx context.Context, id uuid.UUID) (domain.AgentMemory, error)
	FindByUserID(ctx context.Context, userID string) ([]domain.AgentMemory, error)
	FindByUserIDAndType(ctx context.Context, userID string, memType domain.AgentMemoryType) ([]domain.AgentMemory, error)
	SearchByContent(ctx context.Context, userID string, query string) ([]domain.AgentMemory, error)
	Update(ctx context.Context, memory domain.AgentMemory) (domain.AgentMemory, error)
	Delete(ctx context.Context, id uuid.UUID) error
	CountByUserID(ctx context.Context, userID string) (int64, error)
	UpsertRiskProfile(ctx context.Context, memory domain.AgentMemory) (domain.AgentMemory, error)
	DeleteExpired(ctx context.Context) (int64, error)
}

type AgentConversationRepository interface {
	Save(ctx context.Context, conv domain.AgentConversation) (domain.AgentConversation, error)
	FindByID(ctx context.Context, id uuid.UUID) (domain.AgentConversation, error)
	FindByUserID(ctx context.Context, userID string) ([]domain.AgentConversation, error)
	SaveMessage(ctx context.Context, msg domain.AgentMessage) (domain.AgentMessage, error)
	UpdateTitle(ctx context.Context, id uuid.UUID, title string) error
	DeleteExpired(ctx context.Context) (int64, error)
}

type AgentAuditRepository interface {
	Log(ctx context.Context, record domain.AgentAuditRecord) error
}

type AgentGateway interface {
	Chat(ctx context.Context, systemPrompt string, userMessage string, history []domain.AgentMessage) (domain.AgentGatewayResponse, error)
}

// --- Input/Output DTOs ---

type AgentChatInput struct {
	Message        string     `json:"message"`
	ConversationID *uuid.UUID `json:"conversation_id,omitempty"`
}

type AgentChatOutput struct {
	ConversationID uuid.UUID `json:"conversation_id"`
	Response       string    `json:"response"`
}

type SaveMemoryInput struct {
	MemoryType string         `json:"memory_type"`
	Content    string         `json:"content"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

// --- Use Case ---

type AgentUseCase struct {
	memoryRepo AgentMemoryRepository
	convRepo   AgentConversationRepository
	auditRepo  AgentAuditRepository
	gateway    AgentGateway
}

func NewAgentUseCase(
	memoryRepo AgentMemoryRepository,
	convRepo AgentConversationRepository,
	auditRepo AgentAuditRepository,
	gateway AgentGateway,
) *AgentUseCase {
	return &AgentUseCase{
		memoryRepo: memoryRepo,
		convRepo:   convRepo,
		auditRepo:  auditRepo,
		gateway:    gateway,
	}
}

// --- Chat Flow ---

func (u *AgentUseCase) Chat(ctx context.Context, input AgentChatInput) (AgentChatOutput, error) {
	userID := authentication.UserIDFromContext(ctx)
	if userID == "" {
		return AgentChatOutput{}, domain.ErrUnauthorized
	}

	// 1. Resolve or create conversation
	var conv domain.AgentConversation
	if input.ConversationID != nil {
		found, err := u.convRepo.FindByID(ctx, *input.ConversationID)
		if err != nil {
			return AgentChatOutput{}, err
		}
		if found.UserID != userID {
			return AgentChatOutput{}, domain.ErrUnauthorized
		}
		conv = found
	} else {
		newConv := domain.NewAgentConversation(userID)
		saved, err := u.convRepo.Save(ctx, newConv)
		if err != nil {
			return AgentChatOutput{}, err
		}
		conv = saved
	}

	// 2. Load memories for system prompt
	memories, err := u.memoryRepo.FindByUserID(ctx, userID)
	if err != nil {
		return AgentChatOutput{}, err
	}

	selectedMemories := selectMemoriesForPrompt(memories)
	systemPrompt := buildSystemPrompt(selectedMemories)

	// 3. Call the LLM gateway
	gatewayResp, err := u.gateway.Chat(ctx, systemPrompt, input.Message, conv.Messages)
	if err != nil {
		return AgentChatOutput{}, fmt.Errorf("agent gateway error: %w", err)
	}

	if conv.Title == "" {
		if t := deriveConversationTitle(input.Message); t != "" {
			if err := u.convRepo.UpdateTitle(ctx, conv.ID, t); err != nil {
				return AgentChatOutput{}, err
			}
			conv.Title = t
		}
	}

	// 4. Persist messages
	userMsg := domain.NewAgentMessage(conv.ID, "user", input.Message)
	_, _ = u.convRepo.SaveMessage(ctx, userMsg)

	assistantMsg := domain.NewAgentMessage(conv.ID, "assistant", gatewayResp.Content)
	_, _ = u.convRepo.SaveMessage(ctx, assistantMsg)

	// 5. Audit log (LGPD Art. 37)
	auditRecord := domain.AgentAuditRecord{
		UserID:         userID,
		ConversationID: &conv.ID,
		ToolsCalled:    gatewayResp.ToolsCalled,
		InputTokens:    gatewayResp.InputTokens,
		OutputTokens:   gatewayResp.OutputTokens,
		Provider:       "vertex_ai",
		Region:         "southamerica-east1",
		CreatedAt:      time.Now(),
	}
	_ = u.auditRepo.Log(ctx, auditRecord)

	return AgentChatOutput{
		ConversationID: conv.ID,
		Response:       gatewayResp.Content,
	}, nil
}

// --- Memory Management (Agent Tools) ---

func (u *AgentUseCase) SaveMemory(ctx context.Context, input SaveMemoryInput) (domain.AgentMemory, error) {
	userID := authentication.UserIDFromContext(ctx)
	if userID == "" {
		return domain.AgentMemory{}, domain.ErrUnauthorized
	}

	memType := domain.AgentMemoryType(input.MemoryType)
	if !memType.IsValid() {
		return domain.AgentMemory{}, domain.ErrAgentInvalidMemoryType
	}

	// PII check
	if containsPII(input.Content) {
		return domain.AgentMemory{}, domain.ErrAgentPIIDetected
	}

	// Singleton rule for risk_profile
	if memType == domain.MemoryTypeRiskProfile {
		memory := domain.NewAgentMemory(userID, memType, input.Content, domain.MemorySourceExplicit)
		if input.Metadata != nil {
			memory.Metadata = input.Metadata
		}
		return u.memoryRepo.UpsertRiskProfile(ctx, memory)
	}

	// Cap check
	count, err := u.memoryRepo.CountByUserID(ctx, userID)
	if err != nil {
		return domain.AgentMemory{}, err
	}
	if count >= int64(domain.MaxMemoriesPerUser) {
		return domain.AgentMemory{}, domain.ErrAgentMemoryCapExceeded
	}

	memory := domain.NewAgentMemory(userID, memType, input.Content, domain.MemorySourceExplicit)
	if input.Metadata != nil {
		memory.Metadata = input.Metadata
	}

	return u.memoryRepo.Save(ctx, memory)
}

func (u *AgentUseCase) DeleteMemory(ctx context.Context, id uuid.UUID) error {
	return u.memoryRepo.Delete(ctx, id)
}

func (u *AgentUseCase) UpdateMemory(ctx context.Context, id uuid.UUID, content string, metadata map[string]any) (domain.AgentMemory, error) {
	userID := authentication.UserIDFromContext(ctx)

	existing, err := u.memoryRepo.FindByID(ctx, id)
	if err != nil {
		return domain.AgentMemory{}, err
	}
	if existing.UserID != userID {
		return domain.AgentMemory{}, domain.ErrUnauthorized
	}

	if containsPII(content) {
		return domain.AgentMemory{}, domain.ErrAgentPIIDetected
	}

	existing.Content = content
	existing.UpdatedAt = time.Now()
	existing.LastValidated = time.Now()
	if metadata != nil {
		existing.Metadata = metadata
	}

	return u.memoryRepo.Update(ctx, existing)
}

func (u *AgentUseCase) SearchMemories(ctx context.Context, query string) ([]domain.AgentMemory, error) {
	userID := authentication.UserIDFromContext(ctx)
	if query == "" {
		return u.memoryRepo.FindByUserID(ctx, userID)
	}
	return u.memoryRepo.SearchByContent(ctx, userID, query)
}

func (u *AgentUseCase) GetMemoriesByType(ctx context.Context, memType domain.AgentMemoryType) ([]domain.AgentMemory, error) {
	userID := authentication.UserIDFromContext(ctx)
	return u.memoryRepo.FindByUserIDAndType(ctx, userID, memType)
}

// --- Conversation Management ---

func (u *AgentUseCase) GetConversation(ctx context.Context, id uuid.UUID) (domain.AgentConversation, error) {
	userID := authentication.UserIDFromContext(ctx)

	conv, err := u.convRepo.FindByID(ctx, id)
	if err != nil {
		return domain.AgentConversation{}, err
	}
	if conv.UserID != userID {
		return domain.AgentConversation{}, domain.ErrUnauthorized
	}
	return conv, nil
}

func (u *AgentUseCase) ListConversations(ctx context.Context) ([]domain.AgentConversation, error) {
	userID := authentication.UserIDFromContext(ctx)
	return u.convRepo.FindByUserID(ctx, userID)
}

// --- Cleanup Jobs ---

func (u *AgentUseCase) PurgeExpiredMemories(ctx context.Context) (int64, error) {
	return u.memoryRepo.DeleteExpired(ctx)
}

func (u *AgentUseCase) PurgeExpiredConversations(ctx context.Context) (int64, error) {
	return u.convRepo.DeleteExpired(ctx)
}

// --- Internal Helpers ---

var (
	cpfRegex   = regexp.MustCompile(`\d{3}\.?\d{3}\.?\d{3}-?\d{2}`)
	emailRegex = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
)

func containsPII(content string) bool {
	if cpfRegex.MatchString(content) {
		return true
	}
	if emailRegex.MatchString(content) {
		return true
	}
	return false
}

func deriveConversationTitle(message string) string {
	s := strings.TrimSpace(message)
	s = strings.Join(strings.Fields(s), " ")
	if s == "" {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxConversationTitleRunes {
		return s
	}
	return string(runes[:maxConversationTitleRunes-1]) + "…"
}

func selectMemoriesForPrompt(all []domain.AgentMemory) []domain.AgentMemory {
	if len(all) <= domain.MaxMemoriesPerPrompt {
		return all
	}

	var selected []domain.AgentMemory

	// Always include: risk_profile (1)
	for _, m := range all {
		if m.Type == domain.MemoryTypeRiskProfile {
			selected = append(selected, m)
			break
		}
	}

	// Active goals, sorted by priority (max 3)
	goalCount := 0
	for _, m := range all {
		if m.Type == domain.MemoryTypeGoal && goalCount < 3 {
			selected = append(selected, m)
			goalCount++
		}
	}

	// Life events within 6 months (max 2)
	eventCount := 0
	sixMonths := time.Now().AddDate(0, 6, 0)
	for _, m := range all {
		if m.Type == domain.MemoryTypeLifeEvent && eventCount < 2 {
			if m.ExpiresAt == nil || m.ExpiresAt.Before(sixMonths) {
				selected = append(selected, m)
				eventCount++
			}
		}
	}

	// Facts and Constraints (max 5)
	factCount := 0
	for _, m := range all {
		if (m.Type == domain.MemoryTypeFact || m.Type == domain.MemoryTypeConstraint) && factCount < 5 {
			selected = append(selected, m)
			factCount++
		}
	}

	// Insights, most recently validated (max 3)
	insightCount := 0
	for _, m := range all {
		if m.Type == domain.MemoryTypeInsight && insightCount < 3 {
			selected = append(selected, m)
			insightCount++
		}
	}

	// Commitments with check_date this month (max 2)
	commitCount := 0
	now := time.Now()
	for _, m := range all {
		if m.Type == domain.MemoryTypeCommitment && commitCount < 2 {
			_ = now
			selected = append(selected, m)
			commitCount++
		}
	}

	if len(selected) > domain.MaxMemoriesPerPrompt {
		selected = selected[:domain.MaxMemoriesPerPrompt]
	}

	return selected
}

func buildSystemPrompt(memories []domain.AgentMemory) string {
	prompt := `Você é um assistente financeiro pessoal inteligente e empático.
Você ajuda o usuário a entender sua vida financeira, identificar gargalos, planejar o futuro e tomar decisões informadas.

REGRAS:
1. Sempre responda em português brasileiro.
2. Seja conciso mas completo nas análises.
3. Quando o usuário revelar informações sobre sua vida financeira (metas, fatos, restrições, eventos), salve usando a ferramenta save_memory.
4. Nunca inclua CPF, email, nomes de bancos reais ou dados pessoais identificáveis nas memórias salvas.
5. Ao identificar padrões comportamentais em múltiplos meses, salve como insight. Nunca salve snapshots de um único mês como insight.
6. Respeite as restrições (constraints) do usuário ao fazer sugestões.
7. Considere o perfil de risco do usuário ao formular conselhos.
`

	if len(memories) > 0 {
		prompt += "\nMEMÓRIAS DO USUÁRIO (contexto persistente):\n"
		for _, m := range memories {
			prompt += fmt.Sprintf("- [%s] %s\n", m.Type, m.Content)
		}
	}

	return prompt
}
