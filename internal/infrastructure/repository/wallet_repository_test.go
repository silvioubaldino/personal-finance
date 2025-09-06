package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"personal-finance/internal/domain"
	"personal-finance/internal/domain/fixture"
	"personal-finance/internal/plataform/authentication"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupWalletTestDB() *gorm.DB {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	_ = db.AutoMigrate(&WalletDB{})

	return db
}

func TestWalletRepository_FindByID(t *testing.T) {
	tests := map[string]struct {
		prepareDB      func() (*WalletRepository, *uuid.UUID)
		expectedErr    error
		expectedWallet domain.Wallet
	}{
		"should find wallet by ID successfully": {
			prepareDB: func() (*WalletRepository, *uuid.UUID) {
				db := setupWalletTestDB()
				repo := NewWalletRepository(db)

				walletMock := fixture.WalletMock()
				walletDB := FromWalletDomain(walletMock)

				db.Create(&walletDB)

				return repo, walletMock.ID
			},
			expectedErr:    nil,
			expectedWallet: fixture.WalletMock(),
		},
		"should return error when wallet not found": {
			prepareDB: func() (*WalletRepository, *uuid.UUID) {
				db := setupWalletTestDB()
				repo := NewWalletRepository(db)

				invalidID := uuid.New()

				return repo, &invalidID
			},
			expectedErr: fmt.Errorf("wallet: %w: %s",
				errors.New("resource not found"),
				gorm.ErrRecordNotFound,
			),
			expectedWallet: domain.Wallet{},
		},
		"should return error on database failure": {
			prepareDB: func() (*WalletRepository, *uuid.UUID) {
				db := setupWalletTestDB()

				_ = db.Callback().Query().Before("gorm:query").Register("force_error", func(db *gorm.DB) {
					_ = db.AddError(assert.AnError)
				})

				repo := NewWalletRepository(db)
				id := uuid.New()

				return repo, &id
			},
			expectedErr: fmt.Errorf("error finding wallet: %w: %s",
				errors.New("internal system error"),
				assert.AnError.Error(),
			),
			expectedWallet: domain.Wallet{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			repo, id := tc.prepareDB()
			ctx := context.WithValue(context.Background(), authentication.UserID, "user-test-id")

			wallet, err := repo.FindByID(ctx, id)

			assert.Equal(t, tc.expectedWallet.ID, wallet.ID)
			assert.Equal(t, tc.expectedWallet.Description, wallet.Description)
			assert.Equal(t, tc.expectedWallet.Balance, wallet.Balance)
			assert.Equal(t, tc.expectedWallet.UserID, wallet.UserID)
			assert.Equal(t, tc.expectedWallet.InitialBalance, wallet.InitialBalance)
			assert.Equal(t, tc.expectedErr, err)
		})
	}
}

func TestWalletRepository_UpdateAmount(t *testing.T) {
	walletInvalidID := uuid.New()

	tests := map[string]struct {
		prepareDB    func() (*WalletRepository, *uuid.UUID)
		inputTx      func(repository *WalletRepository) *gorm.DB
		inputBalance float64
		expectedErr  error
	}{
		"should update wallet amount successfully": {
			prepareDB: func() (*WalletRepository, *uuid.UUID) {
				db := setupWalletTestDB()
				repo := NewWalletRepository(db)

				walletMock := fixture.WalletMock()
				walletDB := FromWalletDomain(walletMock)

				db.Create(&walletDB)

				return repo, walletMock.ID
			},
			inputTx: func(repository *WalletRepository) *gorm.DB {
				tx := repository.db.Begin()
				return tx
			},
			inputBalance: 2500.0,
			expectedErr:  nil,
		},
		"should update wallet amount with nil transaction": {
			prepareDB: func() (*WalletRepository, *uuid.UUID) {
				db := setupWalletTestDB()
				repo := NewWalletRepository(db)

				walletMock := fixture.WalletMock()
				walletDB := FromWalletDomain(walletMock)

				db.Create(&walletDB)

				return repo, walletMock.ID
			},
			inputTx: func(repository *WalletRepository) *gorm.DB {
				return nil
			},
			inputBalance: 1750.0,
			expectedErr:  nil,
		},
		"should return error when wallet not found": {
			prepareDB: func() (*WalletRepository, *uuid.UUID) {
				db := setupWalletTestDB()
				repo := NewWalletRepository(db)

				return repo, &walletInvalidID
			},
			inputTx: func(repository *WalletRepository) *gorm.DB {
				return nil
			},
			inputBalance: 0,
			expectedErr: fmt.Errorf("wallet: %w: %s",
				errors.New("resource not found"),
				"wallet not found in repository",
			),
		},
		"should return error when wallet belongs to another user": {
			prepareDB: func() (*WalletRepository, *uuid.UUID) {
				db := setupWalletTestDB()
				repo := NewWalletRepository(db)

				walletMock := fixture.WalletMock(
					fixture.WithWalletUserID("another-user-id"),
				)
				walletDB := FromWalletDomain(walletMock)

				db.Create(&walletDB)

				return repo, walletMock.ID
			},
			inputTx: func(repository *WalletRepository) *gorm.DB {
				return nil
			},
			inputBalance: 0,
			expectedErr: fmt.Errorf("wallet: %w: %s",
				errors.New("resource not found"),
				"wallet not found in repository",
			),
		},
		"should return error on database update failure": {
			prepareDB: func() (*WalletRepository, *uuid.UUID) {
				db := setupWalletTestDB()
				repo := NewWalletRepository(db)

				walletMock := fixture.WalletMock()
				walletDB := FromWalletDomain(walletMock)

				db.Create(&walletDB)

				_ = db.Callback().Update().Before("gorm:update").Register("force_update_error", func(db *gorm.DB) {
					_ = db.AddError(assert.AnError)
				})

				return repo, walletMock.ID
			},
			inputTx: func(repository *WalletRepository) *gorm.DB {
				return nil
			},
			inputBalance: 1000.0,
			expectedErr: fmt.Errorf("error updating wallet amount: %w: %s",
				errors.New("internal system error"),
				assert.AnError.Error(),
			),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			repo, id := tc.prepareDB()
			tx := tc.inputTx(repo)
			ctx := context.WithValue(context.Background(), authentication.UserID, "user-test-id")
			defer func() {
				if tx != nil {
					tx.Rollback()
				}
			}()

			err := repo.UpdateAmount(ctx, tx, id, tc.inputBalance)
			if tx != nil {
				tx.Commit()
			}

			assert.Equal(t, tc.expectedErr, err)

			updatedWallet, _ := repo.FindByID(ctx, id)
			assert.Equal(t, tc.inputBalance, updatedWallet.Balance)
		})
	}
}
