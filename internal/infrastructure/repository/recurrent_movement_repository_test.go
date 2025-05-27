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

func setupRecurrentTestDB() *gorm.DB {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})

	_ = db.AutoMigrate(&RecurrentMovementDB{})
	_ = db.AutoMigrate(&WalletDB{})
	_ = db.AutoMigrate(&CategoryDB{})
	_ = db.AutoMigrate(&SubCategoryDB{})

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

func TestRecurrentMovementRepository_FindByID(t *testing.T) {
	tests := map[string]struct {
		prepareDB        func() (*RecurrentMovementRepository, uuid.UUID)
		expectedErr      error
		expectedMovement domain.RecurrentMovement
	}{
		"should find recurrent movement by ID successfully": {
			prepareDB: func() (*RecurrentMovementRepository, uuid.UUID) {
				db := setupRecurrentTestDB()
				repo := NewRecurrentMovementRepository(db)

				id := uuid.New()
				recurrentMovement := fixture.RecurrentMovementMock(
					fixture.WithRecurrentMovementID(id),
					fixture.WithRecurrentMovementDescription("Test recurrent movement"),
					fixture.AsRecurrentMovementIncome(150.00),
					fixture.WithRecurrentMovementUserID("user-test-id"),
				)

				dbMovement := FromRecurrentMovementDomain(recurrentMovement)
				db.Create(&dbMovement)

				return repo, id
			},
			expectedMovement: fixture.RecurrentMovementMock(
				fixture.WithRecurrentMovementDescription("Test recurrent movement"),
				fixture.AsRecurrentMovementIncome(150.00),
				fixture.WithRecurrentMovementUserID("user-test-id"),
			),
			expectedErr: nil,
		},
		"should return error when recurrent movement not found": {
			prepareDB: func() (*RecurrentMovementRepository, uuid.UUID) {
				db := setupRecurrentTestDB()
				repo := NewRecurrentMovementRepository(db)
				return repo, uuid.New()
			},
			expectedMovement: domain.RecurrentMovement{},
			expectedErr: fmt.Errorf("error finding recurrent movement: %w: %s",
				ErrRecurrentMovementNotFound,
				gorm.ErrRecordNotFound.Error(),
			),
		},
		"should return error when database query fails": {
			prepareDB: func() (*RecurrentMovementRepository, uuid.UUID) {
				db := setupRecurrentTestDB()
				_ = db.Callback().Query().Before("gorm:query").Register("force_error", func(db *gorm.DB) {
					_ = db.AddError(assert.AnError)
				})
				repo := NewRecurrentMovementRepository(db)
				return repo, uuid.New()
			},
			expectedMovement: domain.RecurrentMovement{},
			expectedErr: fmt.Errorf("error finding recurrent movement: %w: %s",
				errors.New("internal system error"),
				assert.AnError.Error(),
			),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			repo, id := tc.prepareDB()
			ctx := context.WithValue(context.Background(), authentication.UserID, "user-test-id")

			result, err := repo.FindByID(ctx, id)

			assert.Equal(t, tc.expectedMovement.Description, result.Description)
			assert.Equal(t, tc.expectedMovement.Amount, result.Amount)
			assert.Equal(t, tc.expectedMovement.UserID, result.UserID)
			assert.Equal(t, tc.expectedErr, err)
		})
	}
}

func TestRecurrentMovementRepository_FindByMonth(t *testing.T) {
	now := time.Now()
	pastDate := now.AddDate(0, -1, 0)
	futureDate := now.AddDate(0, 1, 0)

	tests := map[string]struct {
		prepareDB         func() *RecurrentMovementRepository
		date              time.Time
		expectedErr       error
		expectedMovements int
	}{
		"should find recurrent movements by month successfully": {
			prepareDB: func() *RecurrentMovementRepository {
				db := setupRecurrentTestDB()
				repo := NewRecurrentMovementRepository(db)

				id1 := uuid.New()
				initialDate := pastDate
				endDate := futureDate
				recurrent1 := fixture.RecurrentMovementMock(
					fixture.WithRecurrentMovementID(id1),
					fixture.WithRecurrentMovementDescription("Test recurrent movement 1"),
					fixture.AsRecurrentMovementIncome(150.00),
					fixture.WithRecurrentMovementInitialDate(initialDate),
					fixture.WithRecurrentMovementEndDate(endDate),
					fixture.WithRecurrentMovementUserID("user-test-id"),
				)
				dbMovement1 := FromRecurrentMovementDomain(recurrent1)
				db.Create(&dbMovement1)

				id2 := uuid.New()
				endDate2 := futureDate.AddDate(0, 2, 0)
				recurrent2 := fixture.RecurrentMovementMock(
					fixture.WithRecurrentMovementID(id2),
					fixture.WithRecurrentMovementDescription("Test recurrent movement 2"),
					fixture.AsRecurrentMovementIncome(200.00),
					fixture.WithRecurrentMovementInitialDate(pastDate),
					fixture.WithRecurrentMovementEndDate(endDate2),
					fixture.WithRecurrentMovementUserID("user-test-id"),
				)
				dbMovement2 := FromRecurrentMovementDomain(recurrent2)
				db.Create(&dbMovement2)

				id3 := uuid.New()
				futureDatePlus := futureDate.AddDate(0, 1, 0)
				recurrent3 := fixture.RecurrentMovementMock(
					fixture.WithRecurrentMovementID(id3),
					fixture.WithRecurrentMovementDescription("Should not appear"),
					fixture.AsRecurrentMovementIncome(100.00),
					fixture.WithRecurrentMovementInitialDate(futureDatePlus),
					fixture.WithRecurrentMovementUserID("user-test-id"),
				)
				dbMovement3 := FromRecurrentMovementDomain(recurrent3)
				db.Create(&dbMovement3)

				return repo
			},
			date:              now,
			expectedMovements: 2,
			expectedErr:       nil,
		},
		"should return empty slice when no recurrent movements match criteria": {
			prepareDB: func() *RecurrentMovementRepository {
				db := setupRecurrentTestDB()
				repo := NewRecurrentMovementRepository(db)

				// Inserir movimento que não deve aparecer
				id := uuid.New()
				futureDatePlus := futureDate.AddDate(0, 1, 0)
				endDate := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
				recurrent := fixture.RecurrentMovementMock(
					fixture.WithRecurrentMovementID(id),
					fixture.WithRecurrentMovementDescription("Should not appear"),
					fixture.AsRecurrentMovementIncome(100.00),
					fixture.WithRecurrentMovementInitialDate(futureDatePlus),
					fixture.WithRecurrentMovementEndDate(endDate),
					fixture.WithRecurrentMovementUserID("user-test-id"),
				)
				dbMovement := FromRecurrentMovementDomain(recurrent)
				db.Create(&dbMovement)

				return repo
			},
			date:              now,
			expectedMovements: 0,
			expectedErr:       nil,
		},
		"should return error when database query fails": {
			prepareDB: func() *RecurrentMovementRepository {
				db := setupRecurrentTestDB()
				_ = db.Callback().Query().Before("gorm:query").Register("force_error", func(db *gorm.DB) {
					_ = db.AddError(assert.AnError)
				})
				repo := NewRecurrentMovementRepository(db)
				return repo
			},
			date:              now,
			expectedMovements: 0,
			expectedErr: fmt.Errorf("error finding recurrent movements: %w: %s",
				errors.New("internal system error"),
				assert.AnError.Error(),
			),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			repo := tc.prepareDB()
			ctx := context.WithValue(context.Background(), authentication.UserID, "user-test-id")

			results, err := repo.FindByMonth(ctx, tc.date)

			assert.Equal(t, tc.expectedMovements, len(results))
			assert.Equal(t, tc.expectedErr, err)
		})
	}
}
