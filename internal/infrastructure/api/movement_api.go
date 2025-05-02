package api

import (
	"net/http"

	"personal-finance/internal/domain"
	"personal-finance/internal/domain/output"
	"personal-finance/internal/usecase"

	"github.com/gin-gonic/gin"
)

type MovementHandler struct {
	usecase usecase.Movement
}

func NewMovementV2Handlers(r *gin.Engine, srv usecase.Movement) {
	handler := MovementHandler{
		usecase: srv,
	}

	movementGroup := r.Group("/v2/movements")
	movementGroup.POST("/", handler.AddSimple())
}

func (h MovementHandler) AddSimple() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		var movement domain.Movement

		err := c.ShouldBindJSON(&movement)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		savedMovement, err := h.usecase.Add(ctx, movement)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusCreated, output.ToMovementOutput(savedMovement))
	}
}
