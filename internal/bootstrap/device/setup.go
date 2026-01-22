package device

import (
	"personal-finance/internal/bootstrap/registry"
	"personal-finance/internal/infrastructure/api"
	"personal-finance/internal/usecase"

	"github.com/gin-gonic/gin"
)

func Setup(r *gin.Engine, registry *registry.Registry) {
	deviceRepo := registry.GetDeviceRepository()

	deviceService := usecase.NewDevice(deviceRepo)

	api.NewDeviceHandlers(r, &deviceService)
}
