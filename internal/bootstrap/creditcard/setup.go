package creditcard

import (
	"personal-finance/internal/bootstrap/registry"
	"personal-finance/internal/infrastructure/api"
	"personal-finance/internal/usecase"

	"github.com/gin-gonic/gin"
)

func Setup(r *gin.Engine, registry *registry.Registry) {
	creditCardRepo := registry.GetCreditCardRepository()
	txManager := registry.GetTransactionManager()

	creditCardService := usecase.NewCreditCard(
		creditCardRepo,
		txManager,
	)

	api.NewCreditCardV2Handlers(r, &creditCardService)
}
