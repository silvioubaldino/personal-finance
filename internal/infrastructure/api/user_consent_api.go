package api

import (
	"context"
	"net/http"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/usecase"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type (
	UserConsentUseCase interface {
		RecordConsent(ctx context.Context, input usecase.UserConsentInput) (domain.UserConsent, error)
		GetAllConsents(ctx context.Context) ([]domain.UserConsent, error)
		HasConsentedToVersion(ctx context.Context, termVersion string) (bool, error)
		GetConsentByID(ctx context.Context, id uuid.UUID) (domain.UserConsent, error)
	}

	UserConsentHandler struct {
		usecase UserConsentUseCase
	}

	UserConsentRequest struct {
		TermVersion string `json:"term_version" binding:"required"`
	}

	UserConsentResponse struct {
		ID          string    `json:"id"`
		TermVersion string    `json:"term_version"`
		AgreedAt    time.Time `json:"agreed_at"`
		IPAddress   string    `json:"ip_address,omitempty"`
		UserAgent   string    `json:"user_agent,omitempty"`
	}

	ConsentCheckResponse struct {
		HasConsented bool   `json:"has_consented"`
		TermVersion  string `json:"term_version"`
	}
)

func NewUserConsentHandlers(r *gin.Engine, srv UserConsentUseCase) {
	handler := UserConsentHandler{
		usecase: srv,
	}

	meGroup := r.Group("/me")

	meGroup.POST("/consents", handler.RecordConsent())
	meGroup.GET("/consents", handler.GetAllConsents())
	meGroup.GET("/consents/check", handler.CheckConsent())
	meGroup.GET("/consents/:id", handler.GetConsentByID())
}

func (h UserConsentHandler) RecordConsent() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		var req UserConsentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "invalid json body"))
			return
		}

		input := usecase.UserConsentInput{
			TermVersion: req.TermVersion,
			IPAddress:   c.ClientIP(),
			UserAgent:   c.Request.UserAgent(),
		}

		consent, err := h.usecase.RecordConsent(ctx, input)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusCreated, toUserConsentResponse(consent))
	}
}

func (h UserConsentHandler) GetAllConsents() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		consents, err := h.usecase.GetAllConsents(ctx)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		responses := make([]UserConsentResponse, len(consents))
		for i, consent := range consents {
			responses[i] = toUserConsentResponse(consent)
		}

		c.JSON(http.StatusOK, responses)
	}
}

func (h UserConsentHandler) CheckConsent() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		termVersion := c.Query("term_version")
		if termVersion == "" {
			HandleErr(c, ctx, domain.WrapInvalidInput(domain.New("term_version query param is required"), "missing term_version"))
			return
		}

		hasConsented, err := h.usecase.HasConsentedToVersion(ctx, termVersion)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, ConsentCheckResponse{
			HasConsented: hasConsented,
			TermVersion:  termVersion,
		})
	}
}

func (h UserConsentHandler) GetConsentByID() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		idParam := c.Param("id")

		id, err := uuid.Parse(idParam)
		if err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "id must be a valid UUID"))
			return
		}

		consent, err := h.usecase.GetConsentByID(ctx, id)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, toUserConsentResponse(consent))
	}
}

func toUserConsentResponse(consent domain.UserConsent) UserConsentResponse {
	return UserConsentResponse{
		ID:          consent.ID.String(),
		TermVersion: consent.TermVersion,
		AgreedAt:    consent.AgreedAt,
		IPAddress:   consent.IPAddress,
		UserAgent:   consent.UserAgent,
	}
}
