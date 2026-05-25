package user

import (
	"personal-finance/internal/bootstrap/registry"
	"personal-finance/internal/infrastructure/api"
	"personal-finance/internal/usecase"

	"github.com/gin-gonic/gin"
)

func Setup(r *gin.Engine, registry *registry.Registry) {
	userRepo := registry.GetUserRepository()

	userUseCase := usecase.NewUser(userRepo)

	api.NewUserHandlers(r, &userUseCase)
}
