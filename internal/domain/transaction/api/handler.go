package api

import (
	"context"
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

func NewTransactionHandlers(r *gin.Engine, srv service.Service) {
	handler := handler{srv: srv}

	r.GET("/transactions", handler.FindAll())
	r.GET("/transactions/:id", handler.FindByID())
	r.GET("/transactions/period", handler.FindByMonth())
	r.POST("/transactions", handler.Add())
	r.PUT("/transactions/:id", handler.Update())
	r.DELETE("/transactions/:id", handler.Delete())
}

// FindAll godoc
// @Summary List transactions
// @Tags Transaction
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
			c.JSON(http.StatusNotFound, err.Error())
			return
		}
		c.JSON(http.StatusOK, transactions)
	}
}

// FindByID godoc
// @Summary Transaction by ID
// @Tags Transaction
// @Description Transaction by ID
// @Accept json
// @Produce json
// @Param id path string true "Transaction ID"
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
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		transaction, err := h.srv.FindByID(c.Request.Context(), int(id))
		if err != nil {
			c.JSON(http.StatusNotFound, err.Error())
			return
		}
		c.JSON(http.StatusOK, transaction)
	}
}

// FindByMonth godoc
// @Summary Transaction by Month
// @Tags Transaction
// @Description Transaction by month
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
		now := time.Now()
		from := now
		to := now
		if fromString := c.Query("from"); fromString != "" {
			var err error
			from, err = time.Parse("2006-01-02", fromString)
			if err != nil {
				c.JSON(http.StatusInternalServerError, err.Error())
				return
			}
		}
		if toString := c.Query("to"); toString != "" {
			var err error
			to, err = time.Parse("2006-01-02", toString)
			if err != nil {
				c.JSON(http.StatusInternalServerError, err.Error())
				return
			}
		}
		if from == now && to == now {
			c.JSON(http.StatusBadRequest, "date must be informed")
			return
		}
		if from.After(to) {
			c.JSON(http.StatusBadRequest, "'from' must be before 'to'")
			return
		}

		transactions, err := h.srv.FindByMonth(c.Request.Context(), from, to)
		if err != nil {
			c.JSON(http.StatusNotFound, err.Error())
			return
		}
		c.JSON(http.StatusOK, transactions)
	}
}

// Add godoc
// @Summary Creates new transaction
// @Tags Transaction
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
// @Tags Transaction
// @Description Updates existing transaction
// @Accept json
// @Produce json
// @Param id path string true "Transaction ID"
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
// @Tags Transaction
// @Description Delete transaction
// @Accept json
// @Produce json
// @Param id path string true "Transaction ID"
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
