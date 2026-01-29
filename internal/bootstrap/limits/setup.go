package limits

import (
	"personal-finance/internal/bootstrap/registry"
	"personal-finance/internal/infrastructure/api"
	"personal-finance/internal/usecase"

	"github.com/gin-gonic/gin"
)

func Setup(r *gin.Engine, registry *registry.Registry) {
	walletRepo := registry.GetWalletRepository()
	creditCardRepo := registry.GetCreditCardRepository()
	movementRepo := registry.GetMovementRepository()
	recurrentRepo := registry.GetRecurrentMovementRepository()

	limitsUseCase := usecase.NewLimits(walletRepo, creditCardRepo, movementRepo, recurrentRepo)

	api.NewLimitsHandlers(r, limitsUseCase)
}
