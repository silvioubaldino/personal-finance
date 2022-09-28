package api

import (
	"context"
	"net/http"
	"strconv"

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
	r.POST("/transactions", handler.Add())
	r.PUT("/transactions/:id", handler.Update())
	r.DELETE("/transactions/:id", handler.Delete())
}

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
