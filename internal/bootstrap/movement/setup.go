package movement

import (
	"personal-finance/internal/bootstrap/registry"
	"personal-finance/internal/infrastructure/api"
	"personal-finance/internal/usecase"

	"github.com/gin-gonic/gin"
)

func Setup(r *gin.Engine, registry *registry.Registry) {
	movementRepo := registry.GetMovementRepository()
	recurrentRepo := registry.GetRecurrentMovementRepository()
	walletRepo := registry.GetWalletRepository()
	subCategoryRepo := registry.GetSubCategoryRepository()
	txManager := registry.GetTransactionManager()

	movementService := usecase.NewMovement(
		movementRepo,
		recurrentRepo,
		walletRepo,
		subCategoryRepo,
		txManager,
	)

	api.NewMovementV2Handlers(r, &movementService)
}
