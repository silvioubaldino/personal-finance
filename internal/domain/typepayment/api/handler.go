package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"personal-finance/internal/domain/typepayment/service"
	"personal-finance/internal/model"
)

type handler struct {
	srv service.Service
}

func NewTypePaymentHandlers(r *gin.Engine, srv service.Service) {
	handler := handler{srv: srv}

	r.GET("/typePayments", handler.FindAll())
	r.GET("/typePayments/:id", handler.FindByID())
	r.POST("/typePayments", handler.Add())
	r.PUT("/typePayments/:id", handler.Update())
	r.DELETE("/typePayments/:id", handler.Delete())
}

// FindAll godoc
// @Summary List typePayments
// @Tags TypePayments
// @Description list all typePayments
// @Accept json
// @Produce json
// @Success 200 {object} []model.TypePayment
// @Failure 404 {object} string
// @Router /typePayments [get]
func (h handler) FindAll() gin.HandlerFunc {
	return func(c *gin.Context) {
		typePayments, err := h.srv.FindAll(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusNotFound, err.Error())
			return
		}
		c.JSON(http.StatusOK, typePayments)
	}
}

// FindByID godoc
// @Summary typePayment by ID
// @Tags TypePayments
// @Description typePayment by ID
// @Accept json
// @Produce json
// @Param id path string true "TypePayment ID"
// @Success 200 {object} model.TypePayment
// @Failure 404 {object} string
// @Failure 500 {object} string
// @Router /typePayments/:id [get]
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

		wallet, err := h.srv.FindByID(c.Request.Context(), int(id))
		if err != nil {
			c.JSON(http.StatusNotFound, err.Error())
			return
		}
		c.JSON(http.StatusOK, wallet)
	}
}

// Add godoc
// @Summary Creates new typePayment
// @Tags TypePayments
// @Description Creates new typePayment
// @Accept json
// @Produce json
// @Success 201 {object} model.TypePayment
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Router /typePayments [post]
func (h handler) Add() gin.HandlerFunc {
	return func(c *gin.Context) {
		var wallet model.TypePayment
		err := c.ShouldBindJSON(&wallet)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}

		savedCateg, err := h.srv.Add(context.Background(), wallet)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		c.JSON(http.StatusCreated, savedCateg)
	}
}

// Update godoc
// @Summary Updates typePayment
// @Tags TypePayments
// @Description Updates existing typePayment
// @Accept json
// @Produce json
// @Param id path string true "TypePayment ID"
// @Success 200 {object} model.TypePayment
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Router /typePayments/:id [put]
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

		var wallet model.TypePayment
		err = c.ShouldBindJSON(&wallet)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}

		updatedCateg, err := h.srv.Update(context.Background(), int(id), wallet)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, updatedCateg)
	}
}

// Delete godoc
// @Summary Delete typePayment
// @Tags TypePayments
// @Description Delete typePayment
// @Accept json
// @Produce json
// @Param id path string true "TypePayment ID"
// @Success 204 {object} string
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Router /typePayments/:id [delete]
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
