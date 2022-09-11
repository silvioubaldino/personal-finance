package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"personal-finance/internal/business/model"
	"personal-finance/internal/business/service/category"
)

type handler struct {
	srv category.Service
}

func AddHandlers(r *gin.Engine, srv category.Service) {
	handler := handler{srv: srv}

	r.GET("/ping", handler.ping())
	r.GET("/categories", handler.FindAll())
	r.GET("/categories/:id", handler.FindByID())
	r.POST("/categories", handler.Add())
	r.PUT("/categories/:id", handler.Update())
	r.DELETE("/categories/:id", handler.Delete())
}

func (h handler) ping() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, "pong")
	}
}

func (h handler) FindAll() gin.HandlerFunc {
	return func(c *gin.Context) {
		categories, err := h.srv.FindAll(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusNotFound, err)
		}
		c.JSON(http.StatusOK, categories)
	}
}

func (h handler) FindByID() gin.HandlerFunc {
	return func(c *gin.Context) {
		idString := c.Param("id")
		base := 10
		id, err := strconv.ParseInt(idString, base, 64)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err)
			return
		}

		sel, err := h.srv.FindByID(c.Request.Context(), int(id))
		if err != nil {
			c.JSON(http.StatusNotFound, err)
			return
		}
		c.JSON(http.StatusOK, sel)
	}
}

func (h handler) Add() gin.HandlerFunc {
	return func(c *gin.Context) {
		var categ model.Category
		err := c.ShouldBindJSON(&categ)
		if err != nil {
			c.JSON(http.StatusBadRequest, err)
			return
		}

		savedCateg, err := h.srv.Add(context.Background(), categ)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err)
			return
		}

		c.JSON(http.StatusCreated, savedCateg)
	}
}

func (h handler) Update() gin.HandlerFunc {
	return func(c *gin.Context) {
		idString := c.Param("id")
		base := 10
		id, err := strconv.ParseInt(idString, base, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, err)
			return
		}

		var categ model.Category
		err = c.ShouldBindJSON(&categ)
		if err != nil {
			c.JSON(http.StatusBadRequest, err)
			return
		}

		updatedCateg, err := h.srv.Update(context.Background(), int(id), categ)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, updatedCateg)
	}
}

func (h handler) Delete() gin.HandlerFunc {
	return func(c *gin.Context) {
		idString := c.Param("id")
		base := 10
		id, err := strconv.ParseInt(idString, base, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, err)
			return
		}
		err = h.srv.Delete(context.Background(), int(id))
		if err != nil {
			c.JSON(http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusNoContent, nil)
	}
}
