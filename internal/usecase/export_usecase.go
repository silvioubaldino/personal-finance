package usecase

import (
	"context"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/plataform/authentication"
)

type ExportWalletRepository interface {
	FindAll(ctx context.Context) ([]domain.Wallet, error)
}

type ExportCategoryRepository interface {
	FindAll(ctx context.Context) ([]domain.Category, error)
}

type ExportSubCategoryRepository interface {
	FindAll(ctx context.Context) (domain.SubCategoryList, error)
}

type ExportMovementRepository interface {
	FindAllByUserID(ctx context.Context) ([]domain.Movement, error)
}

type ExportRecurrentMovementRepository interface {
	FindAllByUserID(ctx context.Context) ([]domain.RecurrentMovement, error)
}

type ExportCreditCardRepository interface {
	FindAll(ctx context.Context) ([]domain.CreditCard, error)
}

type ExportInvoiceRepository interface {
	FindAllByUserID(ctx context.Context) ([]domain.Invoice, error)
}

type ExportEstimateRepository interface {
	FindAllCategoriesByUserID(ctx context.Context) ([]domain.EstimateCategories, error)
	FindAllSubCategoriesByUserID(ctx context.Context) ([]domain.EstimateSubCategories, error)
}

type Export struct {
	userRepo        UserRepository
	userConsentRepo UserConsentRepository
	walletRepo      ExportWalletRepository
	categoryRepo    ExportCategoryRepository
	subCategoryRepo ExportSubCategoryRepository
	movementRepo    ExportMovementRepository
	recurrentRepo   ExportRecurrentMovementRepository
	creditCardRepo  ExportCreditCardRepository
	invoiceRepo     ExportInvoiceRepository
	estimateRepo    ExportEstimateRepository
}

func NewExport(
	userRepo UserRepository,
	userConsentRepo UserConsentRepository,
	walletRepo ExportWalletRepository,
	categoryRepo ExportCategoryRepository,
	subCategoryRepo ExportSubCategoryRepository,
	movementRepo ExportMovementRepository,
	recurrentRepo ExportRecurrentMovementRepository,
	creditCardRepo ExportCreditCardRepository,
	invoiceRepo ExportInvoiceRepository,
	estimateRepo ExportEstimateRepository,
) Export {
	return Export{
		userRepo:        userRepo,
		userConsentRepo: userConsentRepo,
		walletRepo:      walletRepo,
		categoryRepo:    categoryRepo,
		subCategoryRepo: subCategoryRepo,
		movementRepo:    movementRepo,
		recurrentRepo:   recurrentRepo,
		creditCardRepo:  creditCardRepo,
		invoiceRepo:     invoiceRepo,
		estimateRepo:    estimateRepo,
	}
}

func (u *Export) ExportUserData(ctx context.Context) (domain.UserDataExport, error) {
	userID := ctx.Value(authentication.UserID).(string)

	export := domain.UserDataExport{
		ExportedAt: time.Now(),
		UserID:     userID,
	}

	user, err := u.userRepo.Get(ctx)
	if err == nil {
		export.Preferences = &user
	}

	consents, err := u.userConsentRepo.FindByUserID(ctx)
	if err == nil {
		export.Consents = consents
	}

	wallets, err := u.walletRepo.FindAll(ctx)
	if err == nil {
		export.Wallets = wallets
	}

	categories, err := u.categoryRepo.FindAll(ctx)
	if err == nil {
		userCategories := filterUserCategories(categories, userID)
		export.Categories = userCategories
	}

	subCategories, err := u.subCategoryRepo.FindAll(ctx)
	if err == nil {
		userSubCategories := filterUserSubCategories(subCategories, userID)
		export.SubCategories = userSubCategories
	}

	movements, err := u.movementRepo.FindAllByUserID(ctx)
	if err == nil {
		export.Movements = movements
	}

	recurrents, err := u.recurrentRepo.FindAllByUserID(ctx)
	if err == nil {
		export.Recurrents = recurrents
	}

	creditCards, err := u.creditCardRepo.FindAll(ctx)
	if err == nil {
		export.CreditCards = creditCards
	}

	invoices, err := u.invoiceRepo.FindAllByUserID(ctx)
	if err == nil {
		export.Invoices = invoices
	}

	estCategories, err := u.estimateRepo.FindAllCategoriesByUserID(ctx)
	if err == nil {
		export.Estimates.Categories = estCategories
	}

	estSubCategories, err := u.estimateRepo.FindAllSubCategoriesByUserID(ctx)
	if err == nil {
		export.Estimates.SubCategories = estSubCategories
	}

	return export, nil
}

func filterUserCategories(categories []domain.Category, userID string) []domain.Category {
	result := make([]domain.Category, 0)
	for _, cat := range categories {
		if cat.UserID == userID {
			result = append(result, cat)
		}
	}
	return result
}

func filterUserSubCategories(subCategories []domain.SubCategory, userID string) []domain.SubCategory {
	result := make([]domain.SubCategory, 0)
	for _, subCat := range subCategories {
		if subCat.UserID == userID {
			result = append(result, subCat)
		}
	}
	return result
}
