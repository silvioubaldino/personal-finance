package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/gin-gonic/gin"

	"personal-finance/internal/domain/transaction/service"
	"personal-finance/internal/model"
)

type handler struct {
	service             service.Service
	consolidatedService service.ConsolidatedService
}

const (
	// Base URIs
	_transactions = "/transactions"
	_balance      = "/balance"

	// URIs
	_period = "/period"
)

func NewTransactionHandlers(r *gin.Engine, srv service.Service, consolidatedService service.ConsolidatedService) {
	handler := handler{
		service:             srv,
		consolidatedService: consolidatedService,
	}

	transactionGroup := r.Group(_transactions)

	transactionGroup.GET("/", handler.FindAll())
	transactionGroup.GET("/:id", handler.FindByID())
	transactionGroup.GET("/period", handler.FindByPeriod())
	transactionGroup.GET("/parent/:id", handler.FindConsolidatedByID())
	transactionGroup.GET("/parent/period", handler.FindConsolidatedByPeriod())
	transactionGroup.POST("/", handler.Add())
	transactionGroup.PUT("/:id", handler.Update())
	transactionGroup.DELETE("/:id", handler.Delete())

	r.GET(_balance+_period, handler.BalanceByPeriod())
}

func (h handler) FindAll() gin.HandlerFunc {
	return func(c *gin.Context) {
		transactions, err := h.service.FindAll(c.Request.Context())
		if err != nil {
			handlerError(c, err)
			return
		}
		c.JSON(http.StatusOK, transactions)
	}
}

func (h handler) FindByID() gin.HandlerFunc {
	return func(c *gin.Context) {
		idString := c.Param("id")
		id, err := strconv.ParseInt(idString, 10, 64)
		if err != nil {
			handlerError(c, model.BuildErrValidation(fmt.Sprintf("id must be valid: %s", idString)))
			return
		}

		transaction, err := h.service.FindByID(c.Request.Context(), int(id))
		if err != nil {
			handlerError(c, err)
			return
		}
		c.JSON(http.StatusOK, transaction)
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

		transactions, err := h.service.FindByMonth(c.Request.Context(), period)
		if err != nil {
			c.JSON(http.StatusNotFound, err.Error())
			return
		}
		c.JSON(http.StatusOK, transactions)
	}
}

func (h handler) FindConsolidatedByPeriod() gin.HandlerFunc {
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

		transactions, err := h.consolidatedService.FindConsolidatedTransactionByPeriod(c.Request.Context(), period)
		if err != nil {
			c.JSON(http.StatusNotFound, err.Error())
			return
		}
		c.JSON(http.StatusOK, transactions)
	}
}

func (h handler) FindConsolidatedByID() gin.HandlerFunc {
	return func(c *gin.Context) {
		idParam := c.Param("id")

		id, err := uuid.Parse(idParam)
		if err != nil {
			handlerError(c, model.BuildErrValidation(fmt.Sprintf("id must be valid: %s", idParam)))
		}
		parentTransaction, err := h.consolidatedService.FindConsolidatedTransactionByID(c.Request.Context(), id)
		if err != nil {
			handlerError(c, err)
			return
		}
		c.JSON(http.StatusOK, parentTransaction)
	}
}

func (h handler) BalanceByPeriod() gin.HandlerFunc {
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

		balance, err := h.service.BalanceByPeriod(c.Request.Context(), period)
		if err != nil {
			c.JSON(http.StatusNotFound, err.Error())
			return
		}
		c.JSON(http.StatusOK, balance)
	}
}

func (h handler) Add() gin.HandlerFunc {
	return func(c *gin.Context) {
		var transaction model.Transaction
		err := c.ShouldBindJSON(&transaction)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}

		savedCategory, err := h.service.Add(context.Background(), transaction)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		c.JSON(http.StatusCreated, savedCategory)
	}
}

func (h handler) Update() gin.HandlerFunc {
	return func(c *gin.Context) {
		idString := c.Param("id")
		id, err := strconv.ParseInt(idString, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}

		var transaction model.Transaction
		err = c.ShouldBindJSON(&transaction)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}

		updatedCateg, err := h.service.Update(context.Background(), int(id), transaction)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, updatedCateg)
	}
}

func (h handler) Delete() gin.HandlerFunc {
	return func(c *gin.Context) {
		idString := c.Param("id")
		id, err := strconv.ParseInt(idString, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, err)
			return
		}
		err = h.service.Delete(context.Background(), int(id))
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
