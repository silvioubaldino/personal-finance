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
	categoryRepo := reg.GetCategoryRepository()
	limitsValidator := reg.GetPlanLimitsValidator()

	visionGateway := gateway.NewGeminiVisionGateway()
	classificationGateway := gateway.NewGeminiClassificationGateway()
	pdfDecryptor := gateway.NewPDFCPUDecryptor()

	statementUseCase := usecase.NewStatementUseCase(
		visionGateway,
		classificationGateway,
		movementRepo,
		categoryRepo,
		limitsValidator,
		pdfDecryptor,
	)

	api.NewStatementHandlers(r, statementUseCase)
}
