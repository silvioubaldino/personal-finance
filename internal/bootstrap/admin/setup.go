package admin

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
	adminUseCase := usecase.NewAdmin(firebaseGateway)
	appSettingsUseCase := usecase.NewAppSettings(registry.GetAppSettingsRepository())

	api.NewAdminHandlers(r, adminUseCase, appSettingsUseCase)
}
