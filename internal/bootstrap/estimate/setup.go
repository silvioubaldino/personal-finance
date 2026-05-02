package estimate

import (
	"personal-finance/internal/bootstrap/registry"
	"personal-finance/internal/infrastructure/api"
	"personal-finance/internal/usecase"

	"github.com/gin-gonic/gin"
)

func Setup(r *gin.Engine, reg *registry.Registry) {
	estimateRepo := reg.GetEstimateRepository()
	estimateService := usecase.NewEstimate(estimateRepo)
	api.NewEstimateV2Handlers(r, estimateService)
}
