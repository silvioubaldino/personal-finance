package invoice

import (
	"personal-finance/internal/bootstrap/registry"
	"personal-finance/internal/infrastructure/api"
	"personal-finance/internal/usecase"

	"github.com/gin-gonic/gin"
)

func Setup(r *gin.Engine, registry *registry.Registry) {
	invoiceRepo := registry.GetInvoiceRepository()
	creditCardRepo := registry.GetCreditCardRepository()
	walletRepo := registry.GetWalletRepository()
	txManager := registry.GetTransactionManager()

	invoiceService := usecase.NewInvoice(
		invoiceRepo,
		creditCardRepo,
		walletRepo,
		txManager,
	)

	api.NewInvoiceV2Handlers(r, &invoiceService)
}
