package balance

import (
	"personal-finance/internal/bootstrap/registry"
	"personal-finance/internal/infrastructure/api"
	"personal-finance/internal/usecase"

	"github.com/gin-gonic/gin"
)

func Setup(r *gin.Engine, reg *registry.Registry) {
	movementRepo := reg.GetMovementRepository()
	estimateRepo := reg.GetEstimateRepository()
	balanceService := usecase.NewBalance(movementRepo, estimateRepo)
	api.NewBalanceV2Handlers(r, balanceService)
}
