package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"personal-finance/internal/domain/movement/service"
	"personal-finance/internal/model"
)

type handler struct {
	service service.Movement
}

func NewMovementHandlers(r *gin.Engine, srv service.Movement) {
	handler := handler{
		service: srv,
	}

	movementGroup := r.Group("/movements")

	movementGroup.POST("/", handler.Add())
	movementGroup.PUT("/:id", handler.Update())
	movementGroup.DELETE("/:id", handler.Delete())
	movementGroup.GET("/period", handler.FindByPeriod())
}

func (h handler) Add() gin.HandlerFunc {
	return func(c *gin.Context) {
		var transaction model.Movement
		err := c.ShouldBindJSON(&transaction)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}

		isDoneString := c.Query("isDone")
		isDone, err := strconv.ParseBool(isDoneString)
		if err != nil {
			c.JSON(http.StatusBadRequest, fmt.Errorf("isDone must be 'true' of 'false'").Error())
			return
		}

		savedMovement, err := h.service.Add(context.Background(), transaction, isDone)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		c.JSON(http.StatusCreated, model.ToMovementOutput(&savedMovement))
	}
}

func (h handler) FindByPeriod() gin.HandlerFunc {
	return func(c *gin.Context) {
		var period model.Period
		var err error
		if fromString := c.Query("from"); fromString != "" {
			period.From, err = time.Parse("2006-01-02", fromString)
			if err != nil {
				c.JSON(http.StatusInternalServerError, err.Error())
				return
			}
		}
		if toString := c.Query("to"); toString != "" {
			period.To, err = time.Parse("2006-01-02", toString)
			if err != nil {
				c.JSON(http.StatusInternalServerError, err.Error())
				return
			}
		}

		err = period.Validate()
		if err != nil {
			c.JSON(http.StatusBadRequest, fmt.Sprintf("period invalid: %s", err.Error()))
			return
		}

		movements, err := h.service.FindByPeriod(c.Request.Context(), period)
		if err != nil {
			c.JSON(http.StatusNotFound, err.Error())
			return
		}

		output := make([]model.MovementOutput, len(movements))

		for i, movement := range movements {
			output[i] = *model.ToMovementOutput(&movement)
		}
		c.JSON(http.StatusOK, output)
	}
}

func (h handler) Update() gin.HandlerFunc {
	return func(c *gin.Context) {
		idParam := c.Param("id")

		id, err := uuid.Parse(idParam)
		if err != nil {
			handlerError(c, model.BuildErrValidation(fmt.Sprintf("id must be valid: %s", idParam)))
		}

		var transaction model.Movement
		err = c.ShouldBindJSON(&transaction)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}

		updatedCateg, err := h.service.Update(context.Background(), id, transaction)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, updatedCateg)
	}
}

func (h handler) Delete() gin.HandlerFunc {
	return func(c *gin.Context) {
		idParam := c.Param("id")

		id, err := uuid.Parse(idParam)
		if err != nil {
			handlerError(c, model.BuildErrValidation(fmt.Sprintf("id must be valid: %s", idParam)))
		}
		err = h.service.Delete(context.Background(), id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusNoContent, nil)
	}
}

func handlerError(c *gin.Context, err error) {
	var customError model.BusinessError
	if errors.As(err, &customError) {
		c.JSON(customError.HTTPCode, err.Error())
		return
	}
	c.JSON(http.StatusInternalServerError, model.BusinessError{Msg: "unexpected error"})
}
