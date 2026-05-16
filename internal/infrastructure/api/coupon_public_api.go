package api

import (
	"context"
	"net/http"

	"personal-finance/internal/domain"
	"personal-finance/internal/plataform/authentication"
	"personal-finance/internal/usecase"

	"github.com/gin-gonic/gin"
)

type (
	CouponPublicUseCase interface {
		Preview(ctx context.Context, userID, planID, code string) (usecase.CouponPreview, error)
	}

	CouponPublicHandler struct {
		usecase CouponPublicUseCase
	}

	previewCouponRequest struct {
		PlanID string `json:"plan_id" binding:"required"`
		Code   string `json:"code" binding:"required"`
	}
)

func NewCouponPublicHandlers(r *gin.Engine, srv CouponPublicUseCase, auth gin.HandlerFunc) {
	handler := CouponPublicHandler{usecase: srv}

	group := r.Group("/v2/subscriptions/coupons")
	group.Use(auth)
	group.POST("/preview", handler.Preview())
}

func (h CouponPublicHandler) Preview() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		userID, _ := ctx.Value(authentication.UserID).(string)

		var req previewCouponRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "invalid json body"))
			return
		}

		preview, err := h.usecase.Preview(ctx, userID, req.PlanID, req.Code)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}
		c.JSON(http.StatusOK, preview)
	}
}
