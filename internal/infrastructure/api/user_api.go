package api

import (
	"context"
	"net/http"

	"personal-finance/internal/domain"
	"personal-finance/internal/usecase"

	"github.com/gin-gonic/gin"
)

type (
	UserUseCase interface {
		Get(ctx context.Context) (domain.User, error)
		Update(ctx context.Context, input usecase.UserInput) (domain.User, error)
	}

	UserHandler struct {
		usecase UserUseCase
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

func NewUserHandlers(r *gin.Engine, srv UserUseCase) {
	handler := UserHandler{usecase: srv}

	meGroup := r.Group("/me")

	meGroup.GET("/preferences", handler.Get())
	meGroup.PUT("/preferences", handler.Update())
}

func (h UserHandler) Get() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		user, err := h.usecase.Get(ctx)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, toUserPreferencesResponse(user))
	}
}

func (h UserHandler) Update() gin.HandlerFunc {
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

		input := usecase.UserInput{
			Language: req.Language,
			Currency: req.Currency,
		}

		user, err := h.usecase.Update(ctx, input)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, toUserPreferencesResponse(user))
	}
}

func toUserPreferencesResponse(user domain.User) UserPreferencesResponse {
	return UserPreferencesResponse{
		Language: user.Language,
		Currency: user.Currency,
	}
}
