package api

import (
	"context"
	"net/http"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/plataform/authentication"

	"github.com/gin-gonic/gin"
)

type (
	CouponAdminUseCase interface {
		Create(ctx context.Context, coupon domain.Coupon) error
		Update(ctx context.Context, coupon domain.Coupon) error
		Deactivate(ctx context.Context, id string) error
		GetByID(ctx context.Context, id string) (domain.Coupon, error)
		List(ctx context.Context, onlyActive bool) ([]domain.Coupon, error)
	}

	CouponAdminHandler struct {
		usecase CouponAdminUseCase
	}

	CreateCouponRequest struct {
		ID                string    `json:"id" binding:"required"`
		Code              string    `json:"code" binding:"required"`
		Description       string    `json:"description"`
		DiscountType      string    `json:"discount_type" binding:"required"`
		DiscountValue     float64   `json:"discount_value" binding:"required,gt=0"`
		ValidFrom         time.Time `json:"valid_from" binding:"required"`
		ValidUntil        time.Time `json:"valid_until" binding:"required"`
		MaxRedemptions    *int      `json:"max_redemptions"`
		ApplicablePlanIDs []string  `json:"applicable_plan_ids"`
		IsActive          bool      `json:"is_active"`
	}

	UpdateCouponRequest struct {
		Description       string    `json:"description"`
		DiscountType      string    `json:"discount_type" binding:"required"`
		DiscountValue     float64   `json:"discount_value" binding:"required,gt=0"`
		ValidFrom         time.Time `json:"valid_from" binding:"required"`
		ValidUntil        time.Time `json:"valid_until" binding:"required"`
		MaxRedemptions    *int      `json:"max_redemptions"`
		ApplicablePlanIDs []string  `json:"applicable_plan_ids"`
		IsActive          bool      `json:"is_active"`
	}
)

func NewCouponAdminHandlers(r *gin.Engine, srv CouponAdminUseCase) {
	handler := CouponAdminHandler{usecase: srv}

	adminGroup := r.Group("/admin")
	adminGroup.Use(authentication.AdminAuth())

	adminGroup.POST("/coupons", handler.Create())
	adminGroup.GET("/coupons", handler.List())
	adminGroup.GET("/coupons/:id", handler.Get())
	adminGroup.PUT("/coupons/:id", handler.Update())
	adminGroup.DELETE("/coupons/:id", handler.Deactivate())
}

func (h CouponAdminHandler) Create() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		var req CreateCouponRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "invalid json body"))
			return
		}

		coupon := domain.Coupon{
			ID:                req.ID,
			Code:              req.Code,
			Description:       req.Description,
			DiscountType:      domain.CouponDiscountType(req.DiscountType),
			DiscountValue:     req.DiscountValue,
			ValidFrom:         req.ValidFrom,
			ValidUntil:        req.ValidUntil,
			MaxRedemptions:    req.MaxRedemptions,
			ApplicablePlanIDs: req.ApplicablePlanIDs,
			IsActive:          req.IsActive,
		}

		if err := h.usecase.Create(ctx, coupon); err != nil {
			HandleErr(c, ctx, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"message": "coupon created successfully"})
	}
}

func (h CouponAdminHandler) Update() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")
		if id == "" {
			HandleErr(c, ctx, domain.WrapInvalidInput(domain.New("id is required"), "update coupon"))
			return
		}

		var req UpdateCouponRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "invalid json body"))
			return
		}

		current, err := h.usecase.GetByID(ctx, id)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		current.Description = req.Description
		current.DiscountType = domain.CouponDiscountType(req.DiscountType)
		current.DiscountValue = req.DiscountValue
		current.ValidFrom = req.ValidFrom
		current.ValidUntil = req.ValidUntil
		current.MaxRedemptions = req.MaxRedemptions
		current.ApplicablePlanIDs = req.ApplicablePlanIDs
		current.IsActive = req.IsActive

		if err := h.usecase.Update(ctx, current); err != nil {
			HandleErr(c, ctx, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "coupon updated successfully"})
	}
}

func (h CouponAdminHandler) Deactivate() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")
		if id == "" {
			HandleErr(c, ctx, domain.WrapInvalidInput(domain.New("id is required"), "deactivate coupon"))
			return
		}
		if err := h.usecase.Deactivate(ctx, id); err != nil {
			HandleErr(c, ctx, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "coupon deactivated"})
	}
}

func (h CouponAdminHandler) Get() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")
		if id == "" {
			HandleErr(c, ctx, domain.WrapInvalidInput(domain.New("id is required"), "get coupon"))
			return
		}
		coupon, err := h.usecase.GetByID(ctx, id)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}
		c.JSON(http.StatusOK, coupon)
	}
}

func (h CouponAdminHandler) List() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		onlyActive := c.Query("active") == "true"
		coupons, err := h.usecase.List(ctx, onlyActive)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}
		c.JSON(http.StatusOK, coupons)
	}
}
