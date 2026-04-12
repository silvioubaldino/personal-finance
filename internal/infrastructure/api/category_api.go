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
	CategoryUsecase interface {
		Add(ctx context.Context, category domain.Category) (domain.Category, error)
		FindAll(ctx context.Context) ([]domain.Category, error)
		FindByID(ctx context.Context, id uuid.UUID) (domain.Category, error)
		Update(ctx context.Context, id uuid.UUID, category domain.Category) (domain.Category, error)
		Delete(ctx context.Context, id uuid.UUID) error
	}

	CategoryHandler struct {
		usecase CategoryUsecase
	}
)

func NewCategoryV2Handlers(r *gin.Engine, srv CategoryUsecase) {
	handler := CategoryHandler{usecase: srv}

	group := r.Group("/v2/categories")
	group.POST("/", handler.Add())
	group.GET("/", handler.FindAll())
	group.GET("/:id", handler.FindByID())
	group.PUT("/:id", handler.Update())
	group.DELETE("/:id", handler.Delete())
}

func (h CategoryHandler) Add() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		var category domain.Category
		if err := c.ShouldBindJSON(&category); err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "invalid json body"))
			return
		}

		saved, err := h.usecase.Add(ctx, category)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusCreated, output.ToCategoryOutput(saved))
	}
}

func (h CategoryHandler) FindAll() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		categories, err := h.usecase.FindAll(ctx)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		result := make([]output.CategoryOutput, len(categories))
		for i, cat := range categories {
			result[i] = output.ToCategoryOutput(cat)
		}

		c.JSON(http.StatusOK, result)
	}
}

func (h CategoryHandler) FindByID() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "id must be valid"))
			return
		}

		category, err := h.usecase.FindByID(ctx, id)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, output.ToCategoryOutput(category))
	}
}

func (h CategoryHandler) Update() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "id must be valid"))
			return
		}

		var category domain.Category
		if err := c.ShouldBindJSON(&category); err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "invalid json body"))
			return
		}

		updated, err := h.usecase.Update(ctx, id, category)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, output.ToCategoryOutput(updated))
	}
}

func (h CategoryHandler) Delete() gin.HandlerFunc {
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
