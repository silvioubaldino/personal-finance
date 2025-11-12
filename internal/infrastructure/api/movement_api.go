package api

import (
	"context"
	"net/http"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/domain/output"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type (
	MovementUsecase interface {
		Add(ctx context.Context, movement domain.Movement) (domain.Movement, error)
		FindByPeriod(ctx context.Context, period domain.Period) (domain.MovementList, error)
		Pay(ctx context.Context, id uuid.UUID, date time.Time) (domain.Movement, error)
		RevertPay(ctx context.Context, id uuid.UUID) (domain.Movement, error)
		UpdateOne(ctx context.Context, id uuid.UUID, newMovement domain.Movement) (domain.Movement, error)
		DeleteOne(ctx context.Context, id uuid.UUID) error
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
	movementGroup.POST("/:id/pay", handler.Pay())
	movementGroup.POST("/:id/pay/revert", handler.RevertPay())
	movementGroup.PUT("/:id", handler.UpdateOne())
	movementGroup.DELETE("/:id", handler.DeleteOne())
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

func (h MovementHandler) Pay() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		idParam := c.Param("id")

		id, err := uuid.Parse(idParam)
		if err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "id must be valid"))
			return
		}

		var date time.Time
		if dateString := c.Query("date"); dateString != "" {
			date, err = time.Parse("2006-01-02", dateString)
			if err != nil {
				HandleErr(c, ctx, domain.WrapInvalidInput(err, "invalid date format"))
				return
			}
		}

		paid, err := h.usecase.Pay(ctx, id, date)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, output.ToMovementOutput(paid))
	}
}

func (h MovementHandler) RevertPay() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		idParam := c.Param("id")

		id, err := uuid.Parse(idParam)
		if err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "id must be valid"))
			return
		}

		reverted, err := h.usecase.RevertPay(ctx, id)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, output.ToMovementOutput(reverted))
	}
}

func (h MovementHandler) UpdateOne() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		idParam := c.Param("id")

		id, err := uuid.Parse(idParam)
		if err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "id must be valid"))
			return
		}

		var movement domain.Movement
		err = c.ShouldBindJSON(&movement)
		if err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "error unmarshalling input"))
			return
		}

		updatedMovement, err := h.usecase.UpdateOne(ctx, id, movement)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, output.ToMovementOutput(updatedMovement))
	}
}

func (h MovementHandler) DeleteOne() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		idParam := c.Param("id")

		id, err := uuid.Parse(idParam)
		if err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "id must be valid"))
			return
		}

		err = h.usecase.DeleteOne(ctx, id)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.Status(http.StatusNoContent)
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
