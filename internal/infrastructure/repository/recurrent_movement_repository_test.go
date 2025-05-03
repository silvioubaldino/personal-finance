package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"personal-finance/internal/domain"
	"personal-finance/internal/domain/fixture"
	"personal-finance/internal/plataform/authentication"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupRecurrentTestDB() *gorm.DB {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	_ = db.AutoMigrate(&RecurrentMovementDB{})

	return db
}

func TestRecurrentMovementRepository_Add(t *testing.T) {
	tests := map[string]struct {
		prepareDB        func() *RecurrentMovementRepository
		input            domain.RecurrentMovement
		inputTx          func(repository *RecurrentMovementRepository) *gorm.DB
		expectedErr      error
		expectedMovement domain.RecurrentMovement
	}{
		"should add recurrent movement successfully": {
			prepareDB: func() *RecurrentMovementRepository {
				db := setupRecurrentTestDB()
				return NewRecurrentMovementRepository(db)
			},
			input: fixture.RecurrentMovementMock(),
			inputTx: func(repository *RecurrentMovementRepository) *gorm.DB {
				tx := repository.db.Begin()
				return tx
			},
			expectedMovement: fixture.RecurrentMovementMock(
				fixture.WithRecurrentMovementUserID("user-test-id"),
			),
			expectedErr: nil,
		},
		"should fail when adding recurrent movement with database error": {
			prepareDB: func() *RecurrentMovementRepository {
				db := setupRecurrentTestDB()
				_ = db.Callback().Create().Before("gorm:create").Register("force_error", func(db *gorm.DB) {
					_ = db.AddError(assert.AnError)
				})
				return NewRecurrentMovementRepository(db)
			},
			input: fixture.RecurrentMovementMock(),
			inputTx: func(repository *RecurrentMovementRepository) *gorm.DB {
				tx := repository.db.Begin()
				return tx
			},
			expectedMovement: domain.RecurrentMovement{},
			expectedErr: fmt.Errorf("error creating recurrent movement: %w: %s",
				errors.New("internal system error"),
				assert.AnError.Error(),
			),
		},
		"should add recurrent movement with external transaction": {
			prepareDB: func() *RecurrentMovementRepository {
				db := setupRecurrentTestDB()
				return NewRecurrentMovementRepository(db)
			},
			input: fixture.RecurrentMovementMock(
				fixture.WithRecurrentMovementDescription("Test recurrent movement with external transaction"),
				fixture.AsRecurrentMovementIncome(200.00),
			),
			inputTx: func(repository *RecurrentMovementRepository) *gorm.DB {
				return nil
			},
			expectedMovement: fixture.RecurrentMovementMock(
				fixture.WithRecurrentMovementDescription("Test recurrent movement with external transaction"),
				fixture.AsRecurrentMovementIncome(200.00),
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
