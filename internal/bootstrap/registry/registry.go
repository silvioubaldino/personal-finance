package registry

import (
	"personal-finance/internal/infrastructure/repository"
	"personal-finance/internal/infrastructure/repository/transaction"
	"personal-finance/internal/plataform/authentication"

	"gorm.io/gorm"
)

type Registry struct {
	db                          *gorm.DB
	authenticator               authentication.Authenticator
	transactionManager          transaction.Manager
	walletRepository            *repository.WalletRepository
	categoryRepository          *repository.CategoryRepository
	subCategoryRepository       *repository.SubCategoryRepository
	recurrentMovementRepository *repository.RecurrentMovementRepository
	movementRepository          *repository.MovementRepository
	creditCardRepository        *repository.CreditCardRepository
	invoiceRepository           *repository.InvoiceRepository
	userPreferencesRepository   *repository.UserPreferencesRepository
	userConsentRepository       *repository.UserConsentRepository
	estimateRepository          *repository.EstimateRepository
	deviceRepository            *repository.DeviceRepository
}

func NewRegistry(db *gorm.DB) *Registry {
	return &Registry{
		db: db,
	}
}

func (r *Registry) GetDB() *gorm.DB {
	return r.db
}

func (r *Registry) SetAuthenticator(auth authentication.Authenticator) {
	r.authenticator = auth
}

func (r *Registry) GetAuthenticator() authentication.Authenticator {
	return r.authenticator
}

func (r *Registry) GetTransactionManager() transaction.Manager {
	if r.transactionManager == nil {
		r.transactionManager = transaction.NewGormManager(r.db)
	}
	return r.transactionManager
}

func (r *Registry) GetWalletRepository() *repository.WalletRepository {
	if r.walletRepository == nil {
		r.walletRepository = repository.NewWalletRepository(r.db)
	}
	return r.walletRepository
}

func (r *Registry) GetSubCategoryRepository() *repository.SubCategoryRepository {
	if r.subCategoryRepository == nil {
		r.subCategoryRepository = repository.NewSubCategoryRepository(r.db)
	}
	return r.subCategoryRepository
}

func (r *Registry) GetRecurrentMovementRepository() *repository.RecurrentMovementRepository {
	if r.recurrentMovementRepository == nil {
		r.recurrentMovementRepository = repository.NewRecurrentMovementRepository(r.db)
	}
	return r.recurrentMovementRepository
}

func (r *Registry) GetMovementRepository() *repository.MovementRepository {
	if r.movementRepository == nil {
		r.movementRepository = repository.NewMovementRepository(r.db)
	}
	return r.movementRepository
}

func (r *Registry) GetCreditCardRepository() *repository.CreditCardRepository {
	if r.creditCardRepository == nil {
		r.creditCardRepository = repository.NewCreditCardRepository(r.db)
	}
	return r.creditCardRepository
}

func (r *Registry) GetInvoiceRepository() *repository.InvoiceRepository {
	if r.invoiceRepository == nil {
		r.invoiceRepository = repository.NewInvoiceRepository(r.db)
	}
	return r.invoiceRepository
}

func (r *Registry) GetUserPreferencesRepository() *repository.UserPreferencesRepository {
	if r.userPreferencesRepository == nil {
		r.userPreferencesRepository = repository.NewUserPreferencesRepository(r.db)
	}
	return r.userPreferencesRepository
}

func (r *Registry) GetUserConsentRepository() *repository.UserConsentRepository {
	if r.userConsentRepository == nil {
		r.userConsentRepository = repository.NewUserConsentRepository(r.db)
	}
	return r.userConsentRepository
}

func (r *Registry) GetCategoryRepository() *repository.CategoryRepository {
	if r.categoryRepository == nil {
		r.categoryRepository = repository.NewCategoryRepository(r.db)
	}
	return r.categoryRepository
}

func (r *Registry) GetEstimateRepository() *repository.EstimateRepository {
	if r.estimateRepository == nil {
		r.estimateRepository = repository.NewEstimateRepository(r.db)
	}
	return r.estimateRepository
}

func (r *Registry) GetDeviceRepository() *repository.DeviceRepository {
	if r.deviceRepository == nil {
		r.deviceRepository = repository.NewDeviceRepository(r.db)
	}
	return r.deviceRepository
}
