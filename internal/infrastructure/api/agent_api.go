package api

import (
	"context"
	"net/http"

	"personal-finance/internal/domain"
	"personal-finance/internal/usecase"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type (
	AgentUseCase interface {
		Chat(ctx context.Context, input usecase.AgentChatInput) (usecase.AgentChatOutput, error)
		SaveMemory(ctx context.Context, input usecase.SaveMemoryInput) (domain.AgentMemory, error)
		DeleteMemory(ctx context.Context, id uuid.UUID) error
		UpdateMemory(ctx context.Context, id uuid.UUID, content string, metadata map[string]any) (domain.AgentMemory, error)
		SearchMemories(ctx context.Context, query string) ([]domain.AgentMemory, error)
		GetConversation(ctx context.Context, id uuid.UUID) (domain.AgentConversation, error)
		ListConversations(ctx context.Context) ([]domain.AgentConversation, error)
		PurgeExpiredMemories(ctx context.Context) (int64, error)
		PurgeExpiredConversations(ctx context.Context) (int64, error)
	}

	AgentHandler struct {
		usecase AgentUseCase
	}

	// --- Request/Response DTOs ---

	AgentChatRequest struct {
		Message        string  `json:"message" binding:"required"`
		ConversationID *string `json:"conversation_id,omitempty"`
	}

	AgentChatResponse struct {
		ConversationID string `json:"conversation_id"`
		Response       string `json:"response"`
	}

	AgentSaveMemoryRequest struct {
		MemoryType string         `json:"memory_type" binding:"required"`
		Content    string         `json:"content" binding:"required"`
		Metadata   map[string]any `json:"metadata,omitempty"`
	}

	AgentUpdateMemoryRequest struct {
		Content  string         `json:"content" binding:"required"`
		Metadata map[string]any `json:"metadata,omitempty"`
	}

	AgentPurgeResponse struct {
		Deleted int64 `json:"deleted"`
	}
)

func NewAgentHandlers(r *gin.Engine, srv AgentUseCase) {
	handler := AgentHandler{usecase: srv}

	agentGroup := r.Group("/agent")

	// Chat
	agentGroup.POST("/chat", handler.Chat())

	// Conversations
	agentGroup.GET("/conversations", handler.ListConversations())
	agentGroup.GET("/conversations/:id", handler.GetConversation())

	// Memories
	agentGroup.POST("/memories", handler.SaveMemory())
	agentGroup.GET("/memories", handler.SearchMemories())
	agentGroup.PUT("/memories/:id", handler.UpdateMemory())
	agentGroup.DELETE("/memories/:id", handler.DeleteMemory())
}

func NewAgentJobHandlers(r *gin.RouterGroup, srv AgentUseCase) {
	handler := AgentHandler{usecase: srv}

	r.POST("/agent/purge-memories", handler.PurgeExpiredMemories())
	r.POST("/agent/purge-conversations", handler.PurgeExpiredConversations())
}

// --- Chat ---

func (h AgentHandler) Chat() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		var req AgentChatRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "invalid json body"))
			return
		}

		input := usecase.AgentChatInput{
			Message: req.Message,
		}

		if req.ConversationID != nil && *req.ConversationID != "" {
			convID, err := uuid.Parse(*req.ConversationID)
			if err != nil {
				HandleErr(c, ctx, domain.WrapInvalidInput(err, "conversation_id must be a valid UUID"))
				return
			}
			input.ConversationID = &convID
		}

		output, err := h.usecase.Chat(ctx, input)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, AgentChatResponse{
			ConversationID: output.ConversationID.String(),
			Response:       output.Response,
		})
	}
}

// --- Conversations ---

func (h AgentHandler) ListConversations() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		convs, err := h.usecase.ListConversations(ctx)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, convs)
	}
}

func (h AgentHandler) GetConversation() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "id must be a valid UUID"))
			return
		}

		conv, err := h.usecase.GetConversation(ctx, id)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, conv)
	}
}

// --- Memories ---

func (h AgentHandler) SaveMemory() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		var req AgentSaveMemoryRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "invalid json body"))
			return
		}

		input := usecase.SaveMemoryInput{
			MemoryType: req.MemoryType,
			Content:    req.Content,
			Metadata:   req.Metadata,
		}

		memory, err := h.usecase.SaveMemory(ctx, input)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusCreated, memory)
	}
}

func (h AgentHandler) SearchMemories() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		query := c.Query("q")
		memories, err := h.usecase.SearchMemories(ctx, query)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, memories)
	}
}

func (h AgentHandler) UpdateMemory() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "id must be a valid UUID"))
			return
		}

		var req AgentUpdateMemoryRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "invalid json body"))
			return
		}

		memory, err := h.usecase.UpdateMemory(ctx, id, req.Content, req.Metadata)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, memory)
	}
}

func (h AgentHandler) DeleteMemory() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "id must be a valid UUID"))
			return
		}

		if err := h.usecase.DeleteMemory(ctx, id); err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.Status(http.StatusNoContent)
	}
}

// --- Jobs ---

func (h AgentHandler) PurgeExpiredMemories() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		count, err := h.usecase.PurgeExpiredMemories(ctx)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, AgentPurgeResponse{Deleted: count})
	}
}

func (h AgentHandler) PurgeExpiredConversations() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		count, err := h.usecase.PurgeExpiredConversations(ctx)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, AgentPurgeResponse{Deleted: count})
	}
}
