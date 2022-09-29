package api

import (
	"context"
	"net/http"
	"personal-finance/internal/domain/category/service"
	"strconv"

	"github.com/gin-gonic/gin"

	"personal-finance/internal/model"
)

type handler struct {
	srv service.Service
}

func NewCategoryHandlers(r *gin.Engine, srv service.Service) {
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

// FindAll godoc
// @Summary List categories
// @Tags Category
// @Description list all categories
// @Accept json
// @Produce json
// @Success 200 {object} []model.Category
// @Failure 404 {object} string
// @Router /categories [get]
func (h handler) FindAll() gin.HandlerFunc {
	return func(c *gin.Context) {
		categories, err := h.srv.FindAll(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusNotFound, err.Error())
			return
		}
		c.JSON(http.StatusOK, categories)
	}
}

// FindByID godoc
// @Summary category by ID
// @Tags Category
// @Description category by ID
// @Accept json
// @Produce json
// @Param id path string true "Category ID"
// @Success 200 {object} model.Category
// @Failure 404 {object} string
// @Failure 500 {object} string
// @Router /categories/:id [get]
func (h handler) FindByID() gin.HandlerFunc {
	return func(c *gin.Context) {
		idString := c.Param("id")
		base := 10
		id, err := strconv.ParseInt(idString, base, 64)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		categ, err := h.srv.FindByID(c.Request.Context(), int(id))
		if err != nil {
			c.JSON(http.StatusNotFound, err.Error())
			return
		}
		c.JSON(http.StatusOK, categ)
	}
}

// Add godoc
// @Summary Creates new category
// @Tags Category
// @Description Creates new category
// @Accept json
// @Produce json
// @Success 201 {object} model.Category
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Router /categories [post]
func (h handler) Add() gin.HandlerFunc {
	return func(c *gin.Context) {
		var categ model.Category
		err := c.ShouldBindJSON(&categ)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}

		savedCateg, err := h.srv.Add(context.Background(), categ)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		c.JSON(http.StatusCreated, savedCateg)
	}
}

// Update godoc
// @Summary Updates category
// @Tags Category
// @Description Updates existing category
// @Accept json
// @Produce json
// @Param id path string true "Category ID"
// @Success 200 {object} model.Category
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Router /categories/:id [put]
func (h handler) Update() gin.HandlerFunc {
	return func(c *gin.Context) {
		idString := c.Param("id")
		base := 10
		id, err := strconv.ParseInt(idString, base, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}

		var categ model.Category
		err = c.ShouldBindJSON(&categ)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}

		updatedCateg, err := h.srv.Update(context.Background(), int(id), categ)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, updatedCateg)
	}
}

// Delete godoc
// @Summary Delete category
// @Tags Category
// @Description Delete category
// @Accept json
// @Produce json
// @Param id path string true "Category ID"
// @Success 204 {object} string
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Router /categories/:id [delete]
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
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusNoContent, nil)
	}
}
