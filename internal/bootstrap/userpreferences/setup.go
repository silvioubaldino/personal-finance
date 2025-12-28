package userpreferences

import (
	"personal-finance/internal/bootstrap/registry"
	"personal-finance/internal/infrastructure/api"
	"personal-finance/internal/usecase"

	"github.com/gin-gonic/gin"
)

func Setup(r *gin.Engine, registry *registry.Registry) {
	userPreferencesRepo := registry.GetUserPreferencesRepository()

	userPreferencesUseCase := usecase.NewUserPreferences(userPreferencesRepo)

	api.NewUserPreferencesHandlers(r, &userPreferencesUseCase)
}
