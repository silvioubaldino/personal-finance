package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"personal-finance/internal/domain/transaction/service"
	"personal-finance/internal/model"
)

type handler struct {
	srv service.Service
}

const (
	// Base URIs
	_transactions = "/transactions"
	_balance      = "/balance"

	// URIs
	_period = "/period"
)

func NewTransactionHandlers(r *gin.Engine, srv service.Service) {
	handler := handler{srv: srv}

	transactionGroup := r.Group(_transactions)

	transactionGroup.GET("/", handler.FindAll())
	transactionGroup.GET("/:id", handler.FindByID())
	transactionGroup.GET("/period", handler.FindByMonth())
	transactionGroup.GET("/parent/:id", handler.FindParentByID())
	transactionGroup.POST("/", handler.Add())
	transactionGroup.PUT("/:id", handler.Update())
	transactionGroup.DELETE("/:id", handler.Delete())

	r.GET(_balance+_period, handler.BalanceByPeriod())
}

// FindAll godoc
// @Summary List transactions
// @Tags ParentTransaction
// @Description list all transactions
// @Accept json
// @Produce json
// @Success 200 {object} []model.Transaction
// @Failure 404 {object} string
// @Router /transactions [get]
func (h handler) FindAll() gin.HandlerFunc {
	return func(c *gin.Context) {
		transactions, err := h.srv.FindAll(c.Request.Context())
		if err != nil {
			handlerError(c, err)
			return
		}
		c.JSON(http.StatusOK, transactions)
	}
}

// FindByID godoc
// @Summary ParentTransaction by ID
// @Tags ParentTransaction
// @Description ParentTransaction by ID
// @Accept json
// @Produce json
// @Param id path string true "ParentTransaction ID"
// @Success 200 {object} model.Transaction
// @Failure 404 {object} string
// @Failure 500 {object} string
// @Router /transactions/:id [get]
func (h handler) FindByID() gin.HandlerFunc {
	return func(c *gin.Context) {
		idString := c.Param("id")
		base := 10
		bitSize := 64
		id, err := strconv.ParseInt(idString, base, bitSize)
		if err != nil {
			handlerError(c, model.BuildErrValidation(fmt.Sprintf("id must be valid: %s", idString)))
			return
		}

		transaction, err := h.srv.FindByID(c.Request.Context(), int(id))
		if err != nil {
			handlerError(c, err)
			return
		}
		c.JSON(http.StatusOK, transaction)
	}
}

// FindByMonth godoc
// @Summary ParentTransaction by Month
// @Tags ParentTransaction
// @Description ParentTransaction by month
// @Accept json
// @Produce json
// @Param from path string true "From date"
// @Param to path string true "To date"
// @Success 200 {object} model.Transaction
// @Failure 404 {object} string
// @Failure 500 {object} string
// @Router /transactions/:month [get]
func (h handler) FindByMonth() gin.HandlerFunc {
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

		transactions, err := h.srv.FindByMonth(c.Request.Context(), period)
		if err != nil {
			c.JSON(http.StatusNotFound, err.Error())
			return
		}
		c.JSON(http.StatusOK, transactions)
	}
}

func (h handler) FindParentByID() gin.HandlerFunc {
	return func(c *gin.Context) {
		idString := c.Param("id")
		base := 10
		bitSize := 64
		id, err := strconv.ParseInt(idString, base, bitSize)
		if err != nil {
			handlerError(c, model.BuildErrValidation(fmt.Sprintf("id must be valid: %s", idString)))
			return
		}

		parentTransaction, err := h.srv.FindParentTransactionByID(c.Request.Context(), int(id))
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

		balance, err := h.srv.BalanceByPeriod(c.Request.Context(), period)
		if err != nil {
			c.JSON(http.StatusNotFound, err.Error())
			return
		}
		c.JSON(http.StatusOK, balance)
	}
}

// Add godoc
// @Summary Creates new transaction
// @Tags ParentTransaction
// @Description Creates new transaction
// @Accept json
// @Produce json
// @Success 201 {object} model.Transaction
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Router /transactions [post]
func (h handler) Add() gin.HandlerFunc {
	return func(c *gin.Context) {
		var transaction model.Transaction
		err := c.ShouldBindJSON(&transaction)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}

		savedCategory, err := h.srv.Add(context.Background(), transaction)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		c.JSON(http.StatusCreated, savedCategory)
	}
}

// Update godoc
// @Summary Updates transaction
// @Tags ParentTransaction
// @Description Updates existing transaction
// @Accept json
// @Produce json
// @Param id path string true "ParentTransaction ID"
// @Success 200 {object} model.Transaction
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Router /transactions/:id [put]
func (h handler) Update() gin.HandlerFunc {
	return func(c *gin.Context) {
		idString := c.Param("id")
		base := 10
		bitSize := 64
		id, err := strconv.ParseInt(idString, base, bitSize)
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

		updatedCateg, err := h.srv.Update(context.Background(), int(id), transaction)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, updatedCateg)
	}
}

// Delete godoc
// @Summary Delete transaction
// @Tags ParentTransaction
// @Description Delete transaction
// @Accept json
// @Produce json
// @Param id path string true "ParentTransaction ID"
// @Success 204 {object} string
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Router /transactions/:id [delete]
func (h handler) Delete() gin.HandlerFunc {
	return func(c *gin.Context) {
		idString := c.Param("id")
		base := 10
		bitSize := 64
		id, err := strconv.ParseInt(idString, base, bitSize)
		if err != nil {
			c.JSON(http.StatusBadRequest, err)
			return
		}
		err = h.srv.Delete(context.Background(), int(id))
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
