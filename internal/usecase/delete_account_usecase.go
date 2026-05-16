package usecase

import (
	"context"

	"personal-finance/internal/infrastructure/repository/transaction"
	"personal-finance/internal/plataform/authentication"

	"gorm.io/gorm"
)

type DeleteAccountWalletRepository interface {
	DeleteAllByUserID(ctx context.Context, tx *gorm.DB, userID string) error
}

type DeleteAccountCategoryRepository interface {
	DeleteAllByUserID(ctx context.Context, tx *gorm.DB, userID string) error
}

type DeleteAccountSubCategoryRepository interface {
	DeleteAllByUserID(ctx context.Context, tx *gorm.DB, userID string) error
}

type DeleteAccountMovementRepository interface {
	DeleteAllByUserID(ctx context.Context, tx *gorm.DB, userID string) error
}

type DeleteAccountRecurrentMovementRepository interface {
	DeleteAllByUserID(ctx context.Context, tx *gorm.DB, userID string) error
}

type DeleteAccountCreditCardRepository interface {
	DeleteAllByUserID(ctx context.Context, tx *gorm.DB, userID string) error
}

type DeleteAccountInvoiceRepository interface {
	DeleteAllByUserID(ctx context.Context, tx *gorm.DB, userID string) error
}

type DeleteAccountEstimateRepository interface {
	DeleteAllByUserID(ctx context.Context, tx *gorm.DB, userID string) error
}

type DeleteAccountUserRepository interface {
	Delete(ctx context.Context, tx *gorm.DB, userID string) error
}

type DeleteAccountAuthService interface {
	DeleteUser(ctx context.Context, userID string) error
}

type DeleteAccount struct {
	txManager       transaction.Manager
	authService     DeleteAccountAuthService
	userRepo        DeleteAccountUserRepository
	walletRepo      DeleteAccountWalletRepository
	categoryRepo    DeleteAccountCategoryRepository
	subCategoryRepo DeleteAccountSubCategoryRepository
	movementRepo    DeleteAccountMovementRepository
	recurrentRepo   DeleteAccountRecurrentMovementRepository
	creditCardRepo  DeleteAccountCreditCardRepository
	invoiceRepo     DeleteAccountInvoiceRepository
	estimateRepo    DeleteAccountEstimateRepository
}

func NewDeleteAccount(
	txManager transaction.Manager,
	authService DeleteAccountAuthService,
	userRepo DeleteAccountUserRepository,
	walletRepo DeleteAccountWalletRepository,
	categoryRepo DeleteAccountCategoryRepository,
	subCategoryRepo DeleteAccountSubCategoryRepository,
	movementRepo DeleteAccountMovementRepository,
	recurrentRepo DeleteAccountRecurrentMovementRepository,
	creditCardRepo DeleteAccountCreditCardRepository,
	invoiceRepo DeleteAccountInvoiceRepository,
	estimateRepo DeleteAccountEstimateRepository,
) DeleteAccount {
	return DeleteAccount{
		txManager:       txManager,
		authService:     authService,
		userRepo:        userRepo,
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

func (u *DeleteAccount) DeleteUserAccount(ctx context.Context) error {
	userID := ctx.Value(authentication.UserID).(string)

	return u.txManager.WithTransaction(ctx, func(tx *gorm.DB) error {
		if err := u.estimateRepo.DeleteAllByUserID(ctx, tx, userID); err != nil {
			return err
		}

		if err := u.movementRepo.DeleteAllByUserID(ctx, tx, userID); err != nil {
			return err
		}

		if err := u.recurrentRepo.DeleteAllByUserID(ctx, tx, userID); err != nil {
			return err
		}

		if err := u.invoiceRepo.DeleteAllByUserID(ctx, tx, userID); err != nil {
			return err
		}

		if err := u.creditCardRepo.DeleteAllByUserID(ctx, tx, userID); err != nil {
			return err
		}

		if err := u.walletRepo.DeleteAllByUserID(ctx, tx, userID); err != nil {
			return err
		}

		if err := u.subCategoryRepo.DeleteAllByUserID(ctx, tx, userID); err != nil {
			return err
		}

		if err := u.categoryRepo.DeleteAllByUserID(ctx, tx, userID); err != nil {
			return err
		}

		// FK cascade on users removes user_consents and user_devices.
		if err := u.userRepo.Delete(ctx, tx, userID); err != nil {
			return err
		}

		if err := u.authService.DeleteUser(ctx, userID); err != nil {
			return err
		}

		return nil
	})
}
