package transfer

import (
	"personal-finance/internal/bootstrap/registry"
	"personal-finance/internal/infrastructure/api"
	"personal-finance/internal/usecase"

	"github.com/gin-gonic/gin"
)

func Setup(r *gin.Engine, registry *registry.Registry) {
	movementRepo := registry.GetMovementRepository()
	walletRepo := registry.GetWalletRepository()
	txManager := registry.GetTransactionManager()

	transferService := usecase.NewTransfer(
		movementRepo,
		walletRepo,
		txManager,
	)

	api.NewTransferHandlers(r, &transferService)
}
