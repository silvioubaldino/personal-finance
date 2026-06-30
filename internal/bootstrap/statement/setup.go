package statement

import (
	"personal-finance/internal/bootstrap/registry"
	"personal-finance/internal/infrastructure/api"
	"personal-finance/internal/infrastructure/gateway"
	"personal-finance/internal/infrastructure/repository/transaction"
	"personal-finance/internal/usecase"

	"github.com/gin-gonic/gin"
)

func Setup(r *gin.Engine, reg *registry.Registry) {
	movementRepo := reg.GetMovementRepository()
	categoryRepo := reg.GetCategoryRepository()
	limitsValidator := reg.GetPlanLimitsValidator()
	creditCardRepo := reg.GetCreditCardRepository()
	invoiceRepo := reg.GetInvoiceRepository()
	walletRepo := reg.GetWalletRepository()
	txManager := transaction.NewGormManager(reg.GetDB())

	visionGateway := gateway.NewGeminiVisionGateway()
	classificationGateway := gateway.NewGeminiClassificationGateway()
	pdfDecryptor := gateway.NewPDFCPUDecryptor()

	invoiceUseCase := usecase.NewInvoice(
		invoiceRepo,
		creditCardRepo,
		walletRepo,
		movementRepo,
		txManager,
	)

	statementUseCase := usecase.NewStatementUseCase(
		visionGateway,
		classificationGateway,
		movementRepo,
		categoryRepo,
		limitsValidator,
		pdfDecryptor,
		&invoiceUseCase,
		creditCardRepo,
	)

	api.NewStatementHandlers(r, statementUseCase)
}
