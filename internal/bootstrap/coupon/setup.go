package coupon

import (
	"personal-finance/internal/bootstrap/registry"
	"personal-finance/internal/infrastructure/api"
	"personal-finance/internal/usecase"

	"github.com/gin-gonic/gin"
)

// NewUseCase builds the coupon usecase from the registry. Used by both the
// subscription bootstrap (which needs it as a CouponCheckoutUseCase) and the
// HTTP routes setup below.
func NewUseCase(registry *registry.Registry) *usecase.Coupon {
	return usecase.NewCoupon(
		registry.GetCouponRepository(),
		registry.GetCouponRedemptionRepository(),
		registry.GetSubscriptionPlanRepository(),
		registry.GetDB(),
	)
}

// Setup registers the admin CRUD and authenticated preview endpoints.
func Setup(r *gin.Engine, registry *registry.Registry) {
	couponUseCase := NewUseCase(registry)
	authenticator := registry.GetAuthenticator()

	api.NewCouponAdminHandlers(r, couponUseCase)
	api.NewCouponPublicHandlers(r, couponUseCase, authenticator.Authenticate())
}
