package userconsent

import (
	"personal-finance/internal/bootstrap/registry"
	"personal-finance/internal/infrastructure/api"
	"personal-finance/internal/usecase"

	"github.com/gin-gonic/gin"
)

func Setup(r *gin.Engine, registry *registry.Registry) {
	userConsentRepo := registry.GetUserConsentRepository()

	userConsentUseCase := usecase.NewUserConsent(userConsentRepo)

	api.NewUserConsentHandlers(r, &userConsentUseCase)
}
