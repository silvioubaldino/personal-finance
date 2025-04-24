package api

import (
	"log"
	"net/http"

	"personal-finance/internal/domain"
	"personal-finance/internal/domain/output"
	"personal-finance/internal/usecase"

	"github.com/gin-gonic/gin"
)

type MovementHandler struct {
	service usecase.Movement
}

func NewMovementV2Handlers(r *gin.Engine, srv usecase.Movement) {
	handler := MovementHandler{
		service: srv,
	}

	movementGroup := r.Group("/v2/movements")
	movementGroup.POST("/simple", handler.AddSimple())
}

func (h MovementHandler) AddSimple() gin.HandlerFunc {
	return func(c *gin.Context) {
		var movement domain.Movement
		err := c.ShouldBindJSON(&movement)
		if err != nil {
			log.Printf("Error: %v", err)
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}

		savedMovement, err := h.service.Add(c.Request.Context(), movement)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		c.JSON(http.StatusCreated, output.ToMovementOutput(savedMovement))
	}
}
