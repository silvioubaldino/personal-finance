package api

import (
	"context"
	"net/http"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/plataform/authentication"
	"personal-finance/internal/usecase"

	"github.com/gin-gonic/gin"
)


type (
	AdminUseCase interface {
		GetUserClaims(ctx context.Context, userID string) (usecase.UserClaimsResponse, error)
		SetUserPlan(ctx context.Context, userID string, plan string, expiresAt *time.Time) error
		SetUserRole(ctx context.Context, userID string, role string) error
	}

	SettingsUseCase interface {
		SetPlusPrice(ctx context.Context, price float64) error
	}

	AdminHandler struct {
		usecase AdminUseCase
	}

	SettingsHandler struct {
		usecase SettingsUseCase
	}

	SetPlanRequest struct {
		Plan          string     `json:"plan" binding:"required"`
		PlanExpiresAt *time.Time `json:"plan_expires_at,omitempty"`
	}

	SetRoleRequest struct {
		Role string `json:"role" binding:"required"`
	}

	UpdatePlusPriceRequest struct {
		Price float64 `json:"price" binding:"required,gt=0"`
	}

	SubscriptionPlanAdminUseCase interface {
		CreatePlan(ctx context.Context, plan domain.SubscriptionPlan) error
	}

	SubscriptionPlanAdminHandler struct {
		usecase SubscriptionPlanAdminUseCase
	}

	CreatePlanRequest struct {
		ID            string  `json:"id" binding:"required"`
		Name          string  `json:"name" binding:"required"`
		Price         float64 `json:"price" binding:"required,gt=0"`
		Currency      string  `json:"currency"`
		Frequency     int     `json:"frequency" binding:"required,gt=0"`
		FrequencyType string  `json:"frequency_type" binding:"required"`
		IsActive      bool    `json:"is_active"`
	}
)

func NewAdminHandlers(r *gin.Engine, adminSrv AdminUseCase, settingsSrv SettingsUseCase, planSrv SubscriptionPlanAdminUseCase) {
	adminHandler := AdminHandler{usecase: adminSrv}
	settingsHandler := SettingsHandler{usecase: settingsSrv}
	planHandler := SubscriptionPlanAdminHandler{usecase: planSrv}

	adminGroup := r.Group("/admin")
	adminGroup.Use(authentication.AdminAuth())

	adminGroup.GET("/users/:id/claims", adminHandler.GetUserClaims())
	adminGroup.PUT("/users/:id/plan", adminHandler.SetUserPlan())
	adminGroup.PUT("/users/:id/role", adminHandler.SetUserRole())
	adminGroup.PUT("/settings/plus-price", settingsHandler.UpdatePlusPrice())
	adminGroup.POST("/subscription-plans", planHandler.CreatePlan())
}

func (h SubscriptionPlanAdminHandler) CreatePlan() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		var req CreatePlanRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "invalid json body"))
			return
		}

		plan := domain.SubscriptionPlan{
			ID:            req.ID,
			Name:          req.Name,
			Price:         req.Price,
			Currency:      req.Currency,
			Frequency:     req.Frequency,
			FrequencyType: req.FrequencyType,
			IsActive:      req.IsActive,
		}

		if err := h.usecase.CreatePlan(ctx, plan); err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusCreated, gin.H{"message": "plan created successfully"})
	}
}

func (h SettingsHandler) UpdatePlusPrice() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		var req UpdatePlusPriceRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "invalid json body"))
			return
		}

		if err := h.usecase.SetPlusPrice(ctx, req.Price); err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "price updated successfully"})
	}
}

func (h AdminHandler) GetUserClaims() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		userID := c.Param("id")

		if userID == "" {
			HandleErr(c, ctx, domain.WrapInvalidInput(domain.New("user_id is required"), "get user claims"))
			return
		}

		claims, err := h.usecase.GetUserClaims(ctx, userID)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, claims)
	}
}

func (h AdminHandler) SetUserPlan() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		userID := c.Param("id")

		if userID == "" {
			HandleErr(c, ctx, domain.WrapInvalidInput(domain.New("user_id is required"), "set user plan"))
			return
		}

		var req SetPlanRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "invalid json body"))
			return
		}

		err := h.usecase.SetUserPlan(ctx, userID, req.Plan, req.PlanExpiresAt)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "plan updated successfully"})
	}
}

func (h AdminHandler) SetUserRole() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		userID := c.Param("id")

		if userID == "" {
			HandleErr(c, ctx, domain.WrapInvalidInput(domain.New("user_id is required"), "set user role"))
			return
		}

		var req SetRoleRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "invalid json body"))
			return
		}

		err := h.usecase.SetUserRole(ctx, userID, req.Role)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "role updated successfully"})
	}
}
