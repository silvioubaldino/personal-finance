package api

import (
	"context"
	"net/http"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/domain/output"

	"github.com/gin-gonic/gin"
)

type (
	MovementUsecase interface {
		Add(ctx context.Context, movement domain.Movement) (domain.Movement, error)
		FindByPeriod(ctx context.Context, period domain.Period) (domain.MovementList, error)
	}
	MovementHandler struct {
		usecase MovementUsecase
	}
)

func NewMovementV2Handlers(r *gin.Engine, srv MovementUsecase) {
	handler := MovementHandler{
		usecase: srv,
	}

	movementGroup := r.Group("/v2/movements")
	movementGroup.POST("/", handler.AddSimple())
	movementGroup.GET("/", handler.FindByPeriod())
}

func (h MovementHandler) AddSimple() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		var movement domain.Movement

		err := c.ShouldBindJSON(&movement)
		if err != nil {
			err = domain.WrapInvalidInput(err, "error unmarshalling input")
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

func (h MovementHandler) FindByPeriod() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		period, err := h.parsePeriod(c)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		movements, err := h.usecase.FindByPeriod(ctx, period)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		outputMovements := make([]output.MovementOutput, len(movements))
		for i, movement := range movements {
			outputMovements[i] = *output.ToMovementOutput(movement)
		}

		c.JSON(http.StatusOK, outputMovements)
	}
}

func (h MovementHandler) parsePeriod(c *gin.Context) (domain.Period, error) {
	var period domain.Period
	var err error

	fromString := c.Query("from")
	if fromString != "" {
		period.From, err = time.Parse("2006-01-02", fromString)
		if err != nil {
			return domain.Period{}, domain.WrapInvalidInput(err, "invalid from date format")
		}
	}

	toString := c.Query("to")
	if toString != "" {
		period.To, err = time.Parse("2006-01-02", toString)
		if err != nil {
			return domain.Period{}, domain.WrapInvalidInput(err, "invalid to date format")
		}
	}

	err = period.Validate()
	if err != nil {
		return domain.Period{}, domain.WrapInvalidInput(err, "invalid period")
	}

	return period, nil
}
