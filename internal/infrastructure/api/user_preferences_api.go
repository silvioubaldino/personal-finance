package api

import (
	"context"
	"net/http"

	"personal-finance/internal/domain"
	"personal-finance/internal/usecase"

	"github.com/gin-gonic/gin"
)

type (
	UserPreferencesUseCase interface {
		Get(ctx context.Context) (domain.UserPreferences, error)
		Update(ctx context.Context, input usecase.UserPreferencesInput) (domain.UserPreferences, error)
	}

	UserPreferencesHandler struct {
		usecase UserPreferencesUseCase
	}

	UserPreferencesRequest struct {
		Language string `json:"language"`
		Currency string `json:"currency"`
	}

	UserPreferencesResponse struct {
		Language string `json:"language"`
		Currency string `json:"currency"`
	}
)

func NewUserPreferencesHandlers(r *gin.Engine, srv UserPreferencesUseCase) {
	handler := UserPreferencesHandler{
		usecase: srv,
	}

	meGroup := r.Group("/me")

	meGroup.GET("/preferences", handler.Get())
	meGroup.PUT("/preferences", handler.Update())
}

func (h UserPreferencesHandler) Get() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		prefs, err := h.usecase.Get(ctx)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, toUserPreferencesResponse(prefs))
	}
}

func (h UserPreferencesHandler) Update() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		var req UserPreferencesRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "invalid json body"))
			return
		}

		if req.Language == "" && req.Currency == "" {
			HandleErr(c, ctx, domain.WrapInvalidInput(domain.New("empty request"), "at least one field (language or currency) must be provided"))
			return
		}

		input := usecase.UserPreferencesInput{
			Language: req.Language,
			Currency: req.Currency,
		}

		prefs, err := h.usecase.Update(ctx, input)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, toUserPreferencesResponse(prefs))
	}
}

func toUserPreferencesResponse(prefs domain.UserPreferences) UserPreferencesResponse {
	return UserPreferencesResponse{
		Language: prefs.Language,
		Currency: prefs.Currency,
	}
}
