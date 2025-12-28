package registry

import (
	"personal-finance/internal/infrastructure/repository"
	"personal-finance/internal/infrastructure/repository/transaction"

	"gorm.io/gorm"
)

type Registry struct {
	db                          *gorm.DB
	transactionManager          transaction.Manager
	walletRepository            *repository.WalletRepository
	subCategoryRepository       *repository.SubCategoryRepository
	recurrentMovementRepository *repository.RecurrentMovementRepository
	movementRepository          *repository.MovementRepository
	creditCardRepository        *repository.CreditCardRepository
	invoiceRepository           *repository.InvoiceRepository
	userPreferencesRepository   *repository.UserPreferencesRepository
}

func NewRegistry(db *gorm.DB) *Registry {
	return &Registry{
		db: db,
	}
}

func (r *Registry) GetDB() *gorm.DB {
	return r.db
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
