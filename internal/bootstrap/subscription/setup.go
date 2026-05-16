package subscription

import (
	"personal-finance/internal/bootstrap/registry"
	"personal-finance/internal/infrastructure/api"
	"personal-finance/internal/infrastructure/gateway"
	"personal-finance/internal/usecase"

	"github.com/gin-gonic/gin"
)

func Setup(r *gin.Engine, registry *registry.Registry) {
	authenticator := registry.GetAuthenticator()
	authClient := authenticator.AuthClient()

	firebaseGateway := gateway.NewFirebaseGateway(authClient)
	mpGateway := gateway.NewMercadoPagoGateway()
	planRepo := registry.GetSubscriptionPlanRepository()
	subRepo := registry.GetSubscriptionRepository()

	subscriptionUseCase := usecase.NewSubscription(mpGateway, firebaseGateway, planRepo, subRepo)

	api.NewSubscriptionHandlers(r, subscriptionUseCase, authenticator.Authenticate())
	api.RegisterSubscriptionReturnRoute(r)
}
