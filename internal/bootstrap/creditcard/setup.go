package creditcard

import (
	"personal-finance/internal/bootstrap/registry"
	"personal-finance/internal/infrastructure/api"
	"personal-finance/internal/usecase"

	"github.com/gin-gonic/gin"
)

func Setup(r *gin.Engine, registry *registry.Registry) {
	creditCardRepo := registry.GetCreditCardRepository()
	invoiceRepo := registry.GetInvoiceRepository()
	txManager := registry.GetTransactionManager()
	limitsValidator := registry.GetPlanLimitsValidator()

	creditCardService := usecase.NewCreditCard(
		creditCardRepo,
		invoiceRepo,
		txManager,
		limitsValidator,
	)

	api.NewCreditCardV2Handlers(r, &creditCardService)
}
