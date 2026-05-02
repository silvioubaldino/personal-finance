package wallet

import (
	"personal-finance/internal/bootstrap/registry"
	"personal-finance/internal/infrastructure/api"
	"personal-finance/internal/usecase"

	"github.com/gin-gonic/gin"
)

func Setup(r *gin.Engine, reg *registry.Registry) {
	walletRepo := reg.GetWalletRepository()
	limitsValidator := reg.GetPlanLimitsValidator()
	walletService := usecase.NewWallet(walletRepo, limitsValidator)
	api.NewWalletV2Handlers(r, walletService)
}
