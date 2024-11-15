package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"personal-finance/internal/domain/movement/service"
	"personal-finance/internal/model"
	"personal-finance/internal/plataform/authentication"
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
	movementGroup.POST("/simple", handler.AddSimple())
	movementGroup.POST("/:id/pay", handler.Pay())
	movementGroup.POST("/:id/pay/revert", handler.RevertPay())
	movementGroup.PUT("/:id", handler.Update())
	movementGroup.DELETE("/:id", handler.Delete())
	movementGroup.GET("/period", handler.FindByPeriod())
}

func (h handler) Add() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := authentication.GetUserIDFromContext(c)
		if err != nil {
			log.Printf("Error: %v", err)
			c.JSON(http.StatusUnauthorized, err)
			return
		}

		var transaction model.Movement
		err = c.ShouldBindJSON(&transaction)
		if err != nil {
			log.Printf("Error: %v", err)
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}

		if transaction.StatusID == 0 {
			log.Printf("Error: %v", model.ErrInvalidStatusID)
			c.JSON(http.StatusBadRequest, model.ErrInvalidStatusID.Error())
			return
		}
		savedMovement, err := h.service.Add(context.Background(), transaction, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		c.JSON(http.StatusCreated, model.ToMovementOutput(&savedMovement))
	}
}

func (h handler) AddSimple() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := authentication.GetUserIDFromContext(c)
		if err != nil {
			log.Printf("Error: %v", err)
			c.JSON(http.StatusUnauthorized, err)
			return
		}

		var movement model.Movement
		err = c.ShouldBindJSON(&movement)
		if err != nil {
			log.Printf("Error: %v", err)
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}

		ctx := context.WithValue(c.Request.Context(), "user_id", userID)
		savedMovement, err := h.service.AddSimple(ctx, movement, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		c.JSON(http.StatusCreated, model.ToMovementOutput(&savedMovement))
	}
}

func (h handler) FindByPeriod() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := authentication.GetUserIDFromContext(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, err)
			return
		}
		ctx := context.WithValue(c.Request.Context(), "user_id", userID)

		var period model.Period
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

		movements, err := h.service.FindByPeriod(ctx, period, userID)
		if err != nil {
			c.JSON(http.StatusNotFound, err.Error())
			return
		}

		outputMovement := make([]model.MovementOutput, len(movements))

		for i, movement := range movements {
			outputMovement[i] = *model.ToMovementOutput(&movement)
		}
		c.JSON(http.StatusOK, outputMovement)
	}
}

func (h handler) Pay() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := authentication.GetUserIDFromContext(c)
		if err != nil {
			log.Printf("Error: %v", err)
			c.JSON(http.StatusUnauthorized, err)
			return
		}
		ctx := context.WithValue(c.Request.Context(), "user_id", userID)
		idParam := c.Param("id")

		id, err := uuid.Parse(idParam)
		if err != nil {
			handlerError(c, model.BuildErrValidation(fmt.Sprintf("id must be valid: %s", idParam)))
		}

		var date time.Time
		if dateString := c.Query("date"); dateString != "" {
			date, err = time.Parse("2006-01-02", dateString)
		}

		paid, err := h.service.Pay(ctx, id, date, userID)
		if err != nil {
			log.Printf("Error: %v", err)
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, model.ToMovementOutput(&paid))
	}
}

func (h handler) RevertPay() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := authentication.GetUserIDFromContext(c)
		if err != nil {
			log.Printf("Error: %v", err)
			c.JSON(http.StatusUnauthorized, err)
			return
		}

		idParam := c.Param("id")

		id, err := uuid.Parse(idParam)
		if err != nil {
			handlerError(c, model.BuildErrValidation(fmt.Sprintf("id must be valid: %s", idParam)))
		}

		paid, err := h.service.RevertPay(c.Request.Context(), id, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}
		if err != nil {
			log.Printf("Error: %v", err)
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, paid)
	}
}

func (h handler) Update() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := authentication.GetUserIDFromContext(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, err)
			return
		}

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

		updatedMovement, err := h.service.Update(context.Background(), id, transaction, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}
		outputMovement := model.ToMovementOutput(&updatedMovement)
		c.JSON(http.StatusOK, outputMovement)
	}
}

func (h handler) Delete() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := authentication.GetUserIDFromContext(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, err)
			return
		}

		idParam := c.Param("id")

		id, err := uuid.Parse(idParam)
		if err != nil {
			handlerError(c, model.BuildErrValidation(fmt.Sprintf("id must be valid: %s", idParam)))
		}
		err = h.service.Delete(context.Background(), id, userID)
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
