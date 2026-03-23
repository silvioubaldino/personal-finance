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

	AdminHandler struct {
		usecase AdminUseCase
	}

	SetPlanRequest struct {
		Plan          string     `json:"plan" binding:"required"`
		PlanExpiresAt *time.Time `json:"plan_expires_at,omitempty"`
	}

	SetRoleRequest struct {
		Role string `json:"role" binding:"required"`
	}
)

func NewAdminHandlers(r *gin.Engine, srv AdminUseCase) {
	handler := AdminHandler{
		usecase: srv,
	}

	adminGroup := r.Group("/admin")
	adminGroup.Use(authentication.AdminAuth())

	adminGroup.GET("/users/:id/claims", handler.GetUserClaims())
	adminGroup.PUT("/users/:id/plan", handler.SetUserPlan())
	adminGroup.PUT("/users/:id/role", handler.SetUserRole())
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
