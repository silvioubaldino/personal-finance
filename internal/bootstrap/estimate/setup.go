package estimate

import (
	"personal-finance/internal/bootstrap/registry"
	"personal-finance/internal/infrastructure/api"
	"personal-finance/internal/usecase"

	"github.com/gin-gonic/gin"
)

func Setup(r *gin.Engine, reg *registry.Registry) {
	estimateRepo := reg.GetEstimateRepository()
	categoryRepo := reg.GetCategoryRepository()
	subCategoryRepo := reg.GetSubCategoryRepository()
	movementRepo := reg.GetMovementRepository()
	invoiceRepo := reg.GetInvoiceRepository()
	creditCardRepo := reg.GetCreditCardRepository()
	walletRepo := reg.GetWalletRepository()
	txManager := reg.GetTransactionManager()

	invoiceService := usecase.NewInvoice(
		invoiceRepo,
		creditCardRepo,
		walletRepo,
		movementRepo,
		txManager,
	)

	estimateService := usecase.NewEstimate(estimateRepo, categoryRepo, subCategoryRepo, movementRepo, &invoiceService)
	api.NewEstimateV2Handlers(r, estimateService)
}
