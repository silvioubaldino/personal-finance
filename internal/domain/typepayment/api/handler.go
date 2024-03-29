package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"personal-finance/internal/domain/typepayment/service"
	"personal-finance/internal/model"
	"personal-finance/internal/plataform/authentication"
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

func (h handler) FindAll() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := authentication.GetUserIDFromContext(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, err)
			return
		}

		typePayments, err := h.srv.FindAll(c.Request.Context(), userID)
		if err != nil {
			c.JSON(http.StatusNotFound, err.Error())
			return
		}

		outputTypePayment := make([]model.TypePaymentOutput, len(typePayments))
		for i, typePayment := range typePayments {
			outputTypePayment[i] = model.ToTypePaymentOutput(typePayment)
		}
		c.JSON(http.StatusOK, outputTypePayment)
	}
}

func (h handler) FindByID() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := authentication.GetUserIDFromContext(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, err)
			return
		}

		idString := c.Param("id")
		id, err := strconv.ParseInt(idString, 10, 64)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		typePayment, err := h.srv.FindByID(c.Request.Context(), int(id), userID)
		if err != nil {
			c.JSON(http.StatusNotFound, err.Error())
			return
		}
		c.JSON(http.StatusOK, model.ToTypePaymentOutput(typePayment))
	}
}

func (h handler) Add() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := authentication.GetUserIDFromContext(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, err)
			return
		}

		var typePayment model.TypePayment
		err = c.ShouldBindJSON(&typePayment)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}

		savedTypePayment, err := h.srv.Add(context.Background(), typePayment, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		c.JSON(http.StatusCreated, model.ToTypePaymentOutput(savedTypePayment))
	}
}

func (h handler) Update() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := authentication.GetUserIDFromContext(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, err)
			return
		}

		idString := c.Param("id")
		id, err := strconv.ParseInt(idString, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}

		var typePayment model.TypePayment
		err = c.ShouldBindJSON(&typePayment)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}

		updateTypePayment, err := h.srv.Update(context.Background(), int(id), typePayment, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, model.ToTypePaymentOutput(updateTypePayment))
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
		err = h.srv.Delete(context.Background(), int(id))
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusNoContent, nil)
	}
}
