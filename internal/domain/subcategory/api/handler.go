package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"personal-finance/internal/domain/subcategory/repository"
	"personal-finance/internal/model"
)

type handler struct {
	repository repository.Repository
}

func NewSubCategoryHandlers(r *gin.Engine, repository repository.Repository) {
	handler := handler{
		repository: repository,
	}

	r.POST("/subcategories", handler.Add())
	r.PUT("/subcategories/:id", handler.Update())
}

func (h handler) Add() gin.HandlerFunc {
	return func(c *gin.Context) {
		var subCategory model.SubCategory
		err := c.ShouldBindJSON(&subCategory)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}

		savedSubCategory, err := h.repository.Add(c.Request.Context(), subCategory)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		c.JSON(http.StatusCreated, model.ToSubCategoryOutput(savedSubCategory))
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

		var subCategory model.SubCategory
		err = c.ShouldBindJSON(&subCategory)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}

		updatedSubCategory, err := h.repository.Update(c.Request.Context(), id, subCategory)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, model.ToSubCategoryOutput(updatedSubCategory))
	}
}
