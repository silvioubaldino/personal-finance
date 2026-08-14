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
	dashboardService := usecase.NewDashboard(movementRepo, estimateRepo, invoiceRepo)
	api.NewDashboardV2Handlers(r, dashboardService)
}
