package api

import (
	"context"
	"net/http"
	"strconv"

	"personal-finance/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type (
	EstimateUsecase interface {
		FindByMonth(ctx context.Context, month int, year int) ([]domain.EstimateCategories, error)
		AddEstimateCategory(ctx context.Context, category domain.EstimateCategories) (domain.EstimateCategories, error)
		AddEstimateSubCategory(ctx context.Context, subEstimate domain.EstimateSubCategories) (domain.EstimateSubCategories, error)
		UpdateEstimateCategoryAmount(ctx context.Context, id *uuid.UUID, amount float64) (domain.EstimateCategories, error)
		UpdateEstimateSubCategoryAmount(ctx context.Context, id *uuid.UUID, amount float64) (domain.EstimateSubCategories, error)
		DeleteEstimateCategory(ctx context.Context, id *uuid.UUID) error
		DeleteEstimateSubCategory(ctx context.Context, id *uuid.UUID) error
	}

	EstimateHandler struct {
		usecase EstimateUsecase
	}
)

func NewEstimateV2Handlers(r *gin.Engine, srv EstimateUsecase) {
	handler := EstimateHandler{usecase: srv}

	group := r.Group("/v2/estimate")
	group.GET("/", handler.FindByMonth())
	group.POST("/", handler.AddEstimateCategory())
	group.PUT("/:id", handler.UpdateEstimateCategoryAmount())
	group.DELETE("/:id", handler.DeleteEstimateCategory())

	subGroup := r.Group("/v2/sub-estimate")
	subGroup.POST("/", handler.AddEstimateSubCategory())
	subGroup.PUT("/:id", handler.UpdateEstimateSubCategoryAmount())
	subGroup.DELETE("/:id", handler.DeleteEstimateSubCategory())
}

func (h EstimateHandler) FindByMonth() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		month, err := strconv.Atoi(c.Query("month"))
		if err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "month must be a valid integer"))
			return
		}

		year, err := strconv.Atoi(c.Query("year"))
		if err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "year must be a valid integer"))
			return
		}

		estimates, err := h.usecase.FindByMonth(ctx, month, year)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, estimates)
	}
}

func (h EstimateHandler) AddEstimateCategory() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		var category domain.EstimateCategories
		if err := c.ShouldBindJSON(&category); err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "invalid json body"))
			return
		}

		saved, err := h.usecase.AddEstimateCategory(ctx, category)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusCreated, saved)
	}
}

func (h EstimateHandler) AddEstimateSubCategory() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		var subEstimate domain.EstimateSubCategories
		if err := c.ShouldBindJSON(&subEstimate); err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "invalid json body"))
			return
		}

		saved, err := h.usecase.AddEstimateSubCategory(ctx, subEstimate)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusCreated, saved)
	}
}

func (h EstimateHandler) UpdateEstimateCategoryAmount() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "id must be valid"))
			return
		}

		var body struct {
			Amount float64 `json:"amount"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "invalid json body"))
			return
		}

		updated, err := h.usecase.UpdateEstimateCategoryAmount(ctx, &id, body.Amount)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, updated)
	}
}

func (h EstimateHandler) UpdateEstimateSubCategoryAmount() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "id must be valid"))
			return
		}

		var body struct {
			Amount float64 `json:"amount"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "invalid json body"))
			return
		}

		updated, err := h.usecase.UpdateEstimateSubCategoryAmount(ctx, &id, body.Amount)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, updated)
	}
}

func (h EstimateHandler) DeleteEstimateCategory() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "id must be valid"))
			return
		}

		if err := h.usecase.DeleteEstimateCategory(ctx, &id); err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func (h EstimateHandler) DeleteEstimateSubCategory() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "id must be valid"))
			return
		}

		if err := h.usecase.DeleteEstimateSubCategory(ctx, &id); err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.Status(http.StatusNoContent)
	}
}
