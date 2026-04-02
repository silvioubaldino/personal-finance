package statement

import (
	"personal-finance/internal/bootstrap/registry"
	"personal-finance/internal/infrastructure/api"
	"personal-finance/internal/infrastructure/gateway"
	"personal-finance/internal/usecase"

	"github.com/gin-gonic/gin"
)

func Setup(r *gin.Engine, reg *registry.Registry) {
	movementRepo := reg.GetMovementRepository()
	limitsValidator := reg.GetPlanLimitsValidator()

	// Gateway: Gemini Vision (reuses Vertex AI config)
	visionGateway := gateway.NewGeminiVisionGateway()

	// Use case
	statementUseCase := usecase.NewStatementUseCase(
		visionGateway,
		movementRepo,
		limitsValidator,
	)

	// API handlers (authenticated routes)
	api.NewStatementHandlers(r, statementUseCase)
}
