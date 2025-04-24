package repository

import (
	"context"
	"fmt"
	"testing"

	"personal-finance/internal/domain"
	"personal-finance/internal/plataform/authentication"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB() *gorm.DB {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	_ = db.AutoMigrate(&MovementModel{})

	return db
}

func TestMovementRepository_Add(t *testing.T) {
	tests := map[string]struct {
		prepareDB        func() *MovementRepository
		input            domain.Movement
		inputTx          func(repository *MovementRepository) *gorm.DB
		expectedErr      error
		expectedMovement domain.Movement
	}{
		"should add movement successfully": {
			prepareDB: func() *MovementRepository {
				db := setupTestDB()
				return NewMovementRepository(db)
			},
			input: domain.MovementMock(),
			inputTx: func(repository *MovementRepository) *gorm.DB {
				tx := repository.db.Begin()
				return tx
			},
			expectedMovement: domain.MovementMock(
				domain.WithMovementUserID("user-test-id"),
			),
			expectedErr: nil,
		},
		"should fail when adding movement with database error": {
			prepareDB: func() *MovementRepository {
				db := setupTestDB()
				_ = db.Callback().Create().Before("gorm:create").Register("force_error", func(db *gorm.DB) {
					_ = db.AddError(assert.AnError)
				})
				return NewMovementRepository(db)
			},
			input: domain.MovementMock(),
			inputTx: func(repository *MovementRepository) *gorm.DB {
				tx := repository.db.Begin()
				return tx
			},
			expectedMovement: domain.Movement{},
			expectedErr:      fmt.Errorf("error creating movement: %w", assert.AnError),
		},
		"should add movement with external transaction": {
			prepareDB: func() *MovementRepository {
				db := setupTestDB()
				return NewMovementRepository(db)
			},
			input: domain.MovementMock(
				domain.WithMovementDescription("Test movement with external transaction"),
				domain.AsMovementIncome(200.00),
			),
			inputTx: func(repository *MovementRepository) *gorm.DB {
				return nil
			},
			expectedMovement: domain.MovementMock(
				domain.WithMovementDescription("Test movement with external transaction"),
				domain.AsMovementIncome(200.00),
			),
			expectedErr: nil,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			repo := tc.prepareDB()
			tx := tc.inputTx(repo)
			ctx := context.WithValue(context.Background(), authentication.UserID, "user-test-id")
			defer func() {
				if tx != nil {
					tx.Rollback()
				}
			}()

			result, err := repo.Add(ctx, tx, tc.input)

			assert.Equal(t, tc.expectedMovement.Description, result.Description)
			assert.Equal(t, tc.expectedMovement.Amount, result.Amount)
			assert.Equal(t, tc.expectedMovement.UserID, result.UserID)
			assert.Equal(t, tc.expectedErr, err)
		})
	}
}

func TestToMovementModel(t *testing.T) {
	// Arrange
	domainMovement := domain.MovementMock()

	// Act
	dbModel := ToMovementModel(domainMovement)

	// Assert
	assert.Equal(t, *domainMovement.ID, *dbModel.ID)
	assert.Equal(t, domainMovement.Description, dbModel.Description)
	assert.Equal(t, domainMovement.Amount, dbModel.Amount)
	assert.Equal(t, domainMovement.UserID, dbModel.UserID)
	assert.Equal(t, domainMovement.IsPaid, dbModel.IsPaid)
	assert.Equal(t, domainMovement.IsRecurrent, dbModel.IsRecurrent)
	assert.Equal(t, *domainMovement.WalletID, *dbModel.WalletID)
	assert.Equal(t, *domainMovement.CategoryID, *dbModel.CategoryID)
	assert.Equal(t, domainMovement.TypePaymentID, dbModel.TypePaymentID)
}

func TestMovementModelToDomain(t *testing.T) {
	// Arrange
	domainMovement := domain.MovementMock()
	dbModel := ToMovementModel(domainMovement)

	// Act
	resultDomain := dbModel.ToDomain()

	// Assert
	assert.Equal(t, *domainMovement.ID, *resultDomain.ID)
	assert.Equal(t, domainMovement.Description, resultDomain.Description)
	assert.Equal(t, domainMovement.Amount, resultDomain.Amount)
	assert.Equal(t, domainMovement.UserID, resultDomain.UserID)
	assert.Equal(t, domainMovement.IsPaid, resultDomain.IsPaid)
	assert.Equal(t, domainMovement.IsRecurrent, resultDomain.IsRecurrent)
	assert.Equal(t, *domainMovement.WalletID, *resultDomain.WalletID)
	assert.Equal(t, *domainMovement.CategoryID, *resultDomain.CategoryID)
	assert.Equal(t, domainMovement.TypePaymentID, resultDomain.TypePaymentID)
}
