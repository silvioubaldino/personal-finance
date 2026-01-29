package api

import (
	"context"
	"net/http"

	"personal-finance/internal/usecase"

	"github.com/gin-gonic/gin"
)

type (
	LimitsUseCase interface {
		GetLimits(ctx context.Context) (usecase.LimitsResponse, error)
	}

	LimitsHandler struct {
		usecase LimitsUseCase
	}
)

func NewLimitsHandlers(r *gin.Engine, srv LimitsUseCase) {
	handler := LimitsHandler{
		usecase: srv,
	}

	meGroup := r.Group("/me")

	meGroup.GET("/limits", handler.GetLimits())
}

func (h LimitsHandler) GetLimits() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		limits, err := h.usecase.GetLimits(ctx)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, limits)
	}
}
