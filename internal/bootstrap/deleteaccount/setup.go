package deleteaccount

import (
	"personal-finance/internal/bootstrap/registry"
	"personal-finance/internal/infrastructure/api"
	"personal-finance/internal/usecase"

	"github.com/gin-gonic/gin"
)

func Setup(r *gin.Engine, reg *registry.Registry) {
	txManager := reg.GetTransactionManager()
	authService := reg.GetAuthenticator()
	userRepo := reg.GetUserRepository()
	walletRepo := reg.GetWalletRepository()
	categoryRepo := reg.GetCategoryRepository()
	subCategoryRepo := reg.GetSubCategoryRepository()
	movementRepo := reg.GetMovementRepository()
	recurrentRepo := reg.GetRecurrentMovementRepository()
	creditCardRepo := reg.GetCreditCardRepository()
	invoiceRepo := reg.GetInvoiceRepository()
	estimateRepo := reg.GetEstimateRepository()

	deleteAccountUseCase := usecase.NewDeleteAccount(
		txManager,
		authService,
		userRepo,
		walletRepo,
		categoryRepo,
		subCategoryRepo,
		movementRepo,
		recurrentRepo,
		creditCardRepo,
		invoiceRepo,
		estimateRepo,
	)

	api.NewDeleteAccountHandlers(r, &deleteAccountUseCase)
}
