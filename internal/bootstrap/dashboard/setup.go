package dashboard

import (
	"personal-finance/internal/bootstrap/registry"
	"personal-finance/internal/infrastructure/api"
	"personal-finance/internal/usecase"

	"github.com/gin-gonic/gin"
)

func Setup(r *gin.Engine, reg *registry.Registry) {
	movementRepo := reg.GetMovementRepository()
	estimateRepo := reg.GetEstimateRepository()
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

	dashboardService := usecase.NewDashboard(movementRepo, estimateRepo, &invoiceService)
	api.NewDashboardV2Handlers(r, dashboardService)
}
