package statement

import (
	"personal-finance/internal/bootstrap/registry"
	"personal-finance/internal/infrastructure/api"
	"personal-finance/internal/infrastructure/gateway"
	"personal-finance/internal/infrastructure/repository/transaction"
	"personal-finance/internal/usecase"

	"github.com/gin-gonic/gin"
)

// Gateways substitui os gateways de IA do import. Campo nil = Gemini, como em produção;
// só a suíte de aceite preenche (SPEC-009), para rodar o import sem rede e sem IA.
type Gateways struct {
	Vision         usecase.StatementVisionGateway
	Classification usecase.StatementClassificationGateway
}

func Setup(r *gin.Engine, reg *registry.Registry, gateways Gateways) {
	movementRepo := reg.GetMovementRepository()
	categoryRepo := reg.GetCategoryRepository()
	limitsValidator := reg.GetPlanLimitsValidator()
	creditCardRepo := reg.GetCreditCardRepository()
	invoiceRepo := reg.GetInvoiceRepository()
	walletRepo := reg.GetWalletRepository()
	txManager := transaction.NewGormManager(reg.GetDB())

	var visionGateway usecase.StatementVisionGateway = gateway.NewGeminiVisionGateway()
	if gateways.Vision != nil {
		visionGateway = gateways.Vision
	}
	var classificationGateway usecase.StatementClassificationGateway = gateway.NewGeminiClassificationGateway()
	if gateways.Classification != nil {
		classificationGateway = gateways.Classification
	}
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
		invoiceRepo,
		creditCardRepo,
		txManager,
		walletRepo,
	)

	api.NewStatementHandlers(r, statementUseCase)
}
