package api

import (
	"context"
	"net/http"

	"personal-finance/internal/domain"
	"personal-finance/internal/domain/output"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type (
	SubCategoryUsecase interface {
		Add(ctx context.Context, subcategory domain.SubCategory) (domain.SubCategory, error)
		FindAll(ctx context.Context) (domain.SubCategoryList, error)
		FindByID(ctx context.Context, id uuid.UUID) (domain.SubCategory, error)
		FindByCategoryID(ctx context.Context, categoryID uuid.UUID) (domain.SubCategoryList, error)
		Update(ctx context.Context, id uuid.UUID, subcategory domain.SubCategory) (domain.SubCategory, error)
		Delete(ctx context.Context, id uuid.UUID) error
	}

	SubCategoryHandler struct {
		usecase SubCategoryUsecase
	}
)

func NewSubCategoryV2Handlers(r *gin.Engine, srv SubCategoryUsecase) {
	handler := SubCategoryHandler{usecase: srv}

	group := r.Group("/v2/subcategories")
	group.POST("/", handler.Add())
	group.GET("/", handler.FindAll())
	group.GET("/:id", handler.FindByID())
	group.GET("/by-category/:categoryId", handler.FindByCategoryID())
	group.PUT("/:id", handler.Update())
	group.DELETE("/:id", handler.Delete())
}

func (h SubCategoryHandler) Add() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		var subcategory domain.SubCategory
		if err := c.ShouldBindJSON(&subcategory); err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "invalid json body"))
			return
		}

		saved, err := h.usecase.Add(ctx, subcategory)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusCreated, output.ToSubCategoryOutput(saved))
	}
}

func (h SubCategoryHandler) FindAll() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		subcategories, err := h.usecase.FindAll(ctx)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		result := make([]output.SubCategoryOutput, len(subcategories))
		for i, sub := range subcategories {
			result[i] = output.ToSubCategoryOutput(sub)
		}

		c.JSON(http.StatusOK, result)
	}
}

func (h SubCategoryHandler) FindByID() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "id must be valid"))
			return
		}

		subcategory, err := h.usecase.FindByID(ctx, id)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, output.ToSubCategoryOutput(subcategory))
	}
}

func (h SubCategoryHandler) FindByCategoryID() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		categoryID, err := uuid.Parse(c.Param("categoryId"))
		if err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "categoryId must be valid"))
			return
		}

		subcategories, err := h.usecase.FindByCategoryID(ctx, categoryID)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		result := make([]output.SubCategoryOutput, len(subcategories))
		for i, sub := range subcategories {
			result[i] = output.ToSubCategoryOutput(sub)
		}

		c.JSON(http.StatusOK, result)
	}
}

func (h SubCategoryHandler) Update() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "id must be valid"))
			return
		}

		var subcategory domain.SubCategory
		if err := c.ShouldBindJSON(&subcategory); err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "invalid json body"))
			return
		}

		updated, err := h.usecase.Update(ctx, id, subcategory)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, output.ToSubCategoryOutput(updated))
	}
}

func (h SubCategoryHandler) Delete() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "id must be valid"))
			return
		}

		if err := h.usecase.Delete(ctx, id); err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.Status(http.StatusNoContent)
	}
}
