package deleteaccount

import (
	"personal-finance/internal/bootstrap/registry"
	"personal-finance/internal/infrastructure/api"
	"personal-finance/internal/usecase"

	"github.com/gin-gonic/gin"
)

func Setup(r *gin.Engine, reg *registry.Registry) {
	txManager := reg.GetTransactionManager()
	userPrefsRepo := reg.GetUserPreferencesRepository()
	userConsentRepo := reg.GetUserConsentRepository()
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
		userPrefsRepo,
		userConsentRepo,
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
