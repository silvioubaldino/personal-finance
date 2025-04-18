package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"personal-finance/internal/domain/category/service"
	"personal-finance/internal/model"
)

type handler struct {
	srv service.Service
}

func NewCategoryHandlers(r *gin.Engine, srv service.Service) {
	handler := handler{
		srv: srv,
	}

	r.GET("/categories", handler.FindAll())
	r.GET("/categories/:id", handler.FindByID())
	r.POST("/categories", handler.Add())
	r.PUT("/categories/:id", handler.Update())
	r.DELETE("/categories/:id", handler.Delete())
}

func (h handler) FindAll() gin.HandlerFunc {
	return func(c *gin.Context) {
		categories, err := h.srv.FindAll(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusNotFound, err.Error())
			return
		}

		outputCategory := make([]model.CategoryOutput, len(categories))
		for i, category := range categories {
			outputCategory[i] = model.ToCategoryOutput(category)
		}
		c.JSON(http.StatusOK, outputCategory)
	}
}

func (h handler) FindByID() gin.HandlerFunc {
	return func(c *gin.Context) {
		idString := c.Param("id")
		id, err := uuid.Parse(idString)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		categ, err := h.srv.FindByID(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, err.Error())
			return
		}

		c.JSON(http.StatusOK, model.ToCategoryOutput(categ))
	}
}

func (h handler) Add() gin.HandlerFunc {
	return func(c *gin.Context) {
		var categ model.Category
		err := c.ShouldBindJSON(&categ)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}

		savedCateg, err := h.srv.Add(c.Request.Context(), categ)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		c.JSON(http.StatusCreated, model.ToCategoryOutput(savedCateg))
	}
}

func (h handler) Update() gin.HandlerFunc {
	return func(c *gin.Context) {
		idString := c.Param("id")
		id, err := uuid.Parse(idString)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}

		var category model.Category
		err = c.ShouldBindJSON(&category)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}

		updatedCateg, err := h.srv.Update(c.Request.Context(), id, category)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, model.ToCategoryOutput(updatedCateg))
	}
}

func (h handler) Delete() gin.HandlerFunc {
	return func(c *gin.Context) {
		idString := c.Param("id")
		id, err := uuid.Parse(idString)
		if err != nil {
			c.JSON(http.StatusBadRequest, err)
			return
		}
		err = h.srv.Delete(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusNoContent, nil)
	}
}
