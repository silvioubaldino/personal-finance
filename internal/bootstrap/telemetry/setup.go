package telemetry

import (
	"personal-finance/internal/bootstrap/registry"
	"personal-finance/internal/infrastructure/api"
	"personal-finance/internal/usecase"

	"github.com/gin-gonic/gin"
)

func Setup(r *gin.Engine, _ *registry.Registry) {
	telemetryUseCase := usecase.NewTelemetryUseCase()

	api.NewTelemetryHandlers(r, telemetryUseCase)
}
