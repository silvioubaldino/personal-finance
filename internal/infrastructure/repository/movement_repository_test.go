package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/domain/fixture"
	"personal-finance/internal/plataform/authentication"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB() *gorm.DB {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	_ = db.AutoMigrate(&MovementDB{}, &WalletDB{}, &CategoryDB{}, &SubCategoryDB{})

	return db
}

func createTestContext() context.Context {
	return context.WithValue(context.Background(), authentication.UserID, "user-test-id")
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
			input: fixture.MovementMock(),
			inputTx: func(repository *MovementRepository) *gorm.DB {
				tx := repository.db.Begin()
				return tx
			},
			expectedMovement: fixture.MovementMock(
				fixture.WithMovementUserID("user-test-id"),
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
			input: fixture.MovementMock(),
			inputTx: func(repository *MovementRepository) *gorm.DB {
				tx := repository.db.Begin()
				return tx
			},
			expectedMovement: domain.Movement{},
			expectedErr: fmt.Errorf("error creating movement: %w: %s",
				errors.New("internal system error"),
				assert.AnError.Error(),
			),
		},
		"should add movement with external transaction": {
			prepareDB: func() *MovementRepository {
				db := setupTestDB()
				return NewMovementRepository(db)
			},
			input: fixture.MovementMock(
				fixture.WithMovementDescription("Test movement with external transaction"),
				fixture.AsMovementIncome(200.00),
			),
			inputTx: func(repository *MovementRepository) *gorm.DB {
				return nil
			},
			expectedMovement: fixture.MovementMock(
				fixture.WithMovementDescription("Test movement with external transaction"),
				fixture.AsMovementIncome(200.00),
			),
			expectedErr: nil,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			repo := tc.prepareDB()
			tx := tc.inputTx(repo)
			ctx := createTestContext()
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

func TestMovementRepository_FindByID(t *testing.T) {
	tests := map[string]struct {
		prepareDB        func(ctx context.Context) (*MovementRepository, uuid.UUID)
		expectedErr      error
		expectedMovement domain.Movement
	}{
		"should find movement by ID": {
			prepareDB: func(ctx context.Context) (*MovementRepository, uuid.UUID) {
				db := setupTestDB()
				repo := NewMovementRepository(db)

				movement := fixture.MovementMock(
					fixture.WithMovementUserID("user-test-id"),
				)
				dbMovement := FromMovementDomain(movement)

				db.WithContext(ctx).Create(&dbMovement)

				return repo, *movement.ID
			},
			expectedErr: nil,
			expectedMovement: fixture.MovementMock(
				fixture.WithMovementUserID("user-test-id"),
			),
		},
		"should return error when movement not found": {
			prepareDB: func(ctx context.Context) (*MovementRepository, uuid.UUID) {
				db := setupTestDB()
				repo := NewMovementRepository(db)

				return repo, uuid.New()
			},
			expectedErr: domain.WrapNotFound(
				errors.New("record not found"),
				"movement not found",
			),
			expectedMovement: domain.Movement{},
		},
		"should return error when database fails": {
			prepareDB: func(ctx context.Context) (*MovementRepository, uuid.UUID) {
				db := setupTestDB()
				_ = db.Callback().Query().Before("gorm:query").Register("force_error", func(db *gorm.DB) {
					_ = db.AddError(assert.AnError)
				})
				repo := NewMovementRepository(db)

				return repo, uuid.New()
			},
			expectedErr: domain.WrapInternalError(
				assert.AnError,
				"error finding movement",
			),
			expectedMovement: domain.Movement{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := createTestContext()
			repo, id := tc.prepareDB(ctx)

			// Act
			result, err := repo.FindByID(ctx, id)

			// Assert
			assert.Equal(t, tc.expectedMovement.ID, result.ID)
			assert.Equal(t, tc.expectedMovement.Description, result.Description)
			assert.Equal(t, tc.expectedMovement.Amount, result.Amount)
			assert.Equal(t, tc.expectedErr, err)
		})
	}
}

func TestMovementRepository_FindByPeriod(t *testing.T) {
	tests := map[string]struct {
		prepareDB     func(ctx context.Context) (*MovementRepository, domain.Period)
		expectedErr   error
		expectedCount int
	}{
		"should find movements by period": {
			prepareDB: func(ctx context.Context) (*MovementRepository, domain.Period) {
				db := setupTestDB()
				repo := NewMovementRepository(db)

				now := time.Now()
				yesterday := now.AddDate(0, 0, -1)
				tomorrow := now.AddDate(0, 0, 1)

				// Create 3 movements with different dates
				movement1 := fixture.MovementMock(
					fixture.WithMovementUserID("user-test-id"),
					fixture.WithMovementDate(yesterday),
				)
				movement2 := fixture.MovementMock(
					fixture.WithMovementUserID("user-test-id"),
					fixture.WithMovementDate(now),
					fixture.WithMovementID(uuid.New()),
				)
				movement3 := fixture.MovementMock(
					fixture.WithMovementUserID("user-test-id"),
					fixture.WithMovementDate(tomorrow),
					fixture.WithMovementID(uuid.New()),
				)

				dbMovement1 := FromMovementDomain(movement1)
				dbMovement2 := FromMovementDomain(movement2)
				dbMovement3 := FromMovementDomain(movement3)

				db.WithContext(ctx).Create(&dbMovement1)
				db.WithContext(ctx).Create(&dbMovement2)
				db.WithContext(ctx).Create(&dbMovement3)

				period := domain.Period{
					From: yesterday,
					To:   now,
				}
				return repo, period
			},
			expectedErr:   nil,
			expectedCount: 2,
		},
		"should return empty list when no movements found in period": {
			prepareDB: func(ctx context.Context) (*MovementRepository, domain.Period) {
				db := setupTestDB()
				repo := NewMovementRepository(db)

				now := time.Now()
				pastPeriod := domain.Period{
					From: now.AddDate(0, -1, 0),
					To:   now.AddDate(0, -1, 5),
				}

				return repo, pastPeriod
			},
			expectedErr:   nil,
			expectedCount: 0,
		},
		"should return error when database fails": {
			prepareDB: func(ctx context.Context) (*MovementRepository, domain.Period) {
				db := setupTestDB()
				_ = db.Callback().Query().Before("gorm:query").Register("force_error", func(db *gorm.DB) {
					_ = db.AddError(assert.AnError)
				})
				repo := NewMovementRepository(db)

				now := time.Now()
				period := domain.Period{
					From: now.AddDate(0, 0, -1),
					To:   now,
				}

				return repo, period
			},
			expectedErr: domain.WrapInternalError(
				assert.AnError,
				"error finding movements by period",
			),
			expectedCount: 0,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := createTestContext()
			repo, period := tc.prepareDB(ctx)

			// Act
			results, err := repo.FindByPeriod(ctx, period)

			// Assert
			assert.Equal(t, tc.expectedCount, len(results))
			assert.Equal(t, tc.expectedErr, err)
		})
	}
}
