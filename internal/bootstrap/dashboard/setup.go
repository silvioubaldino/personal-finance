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
	dashboardService := usecase.NewDashboard(movementRepo, estimateRepo)
	api.NewDashboardV2Handlers(r, dashboardService)
}
