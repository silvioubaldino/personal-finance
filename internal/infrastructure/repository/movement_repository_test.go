package repository

import (
	"context"
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
	_ = db.AutoMigrate(&MovementDB{}, &WalletDB{}, &CategoryDB{}, &SubCategoryDB{}, &RecurrentMovementDB{})

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
			expectedErr:      fmt.Errorf("error creating movement: %w: %s", ErrDatabaseError, assert.AnError.Error()),
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
			expectedErr:      fmt.Errorf("error finding movement: %w: %s", ErrMovementNotFound, gorm.ErrRecordNotFound.Error()),
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
			expectedErr:      fmt.Errorf("error finding movement: %w: %s", ErrDatabaseError, assert.AnError.Error()),
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
			expectedErr:   fmt.Errorf("error finding movements by period: %w: %s", ErrDatabaseError, assert.AnError.Error()),
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

func TestMovementRepository_UpdateIsPaid(t *testing.T) {
	tests := map[string]struct {
		prepareDB        func(ctx context.Context) (*MovementRepository, uuid.UUID, domain.Movement)
		inputTx          func(repository *MovementRepository) *gorm.DB
		updateMovement   domain.Movement
		expectedErr      error
		expectedMovement domain.Movement
	}{
		"should update movement is_paid successfully with local transaction": {
			prepareDB: func(ctx context.Context) (*MovementRepository, uuid.UUID, domain.Movement) {
				db := setupTestDB()
				repo := NewMovementRepository(db)

				original := fixture.MovementMock(
					fixture.WithMovementUserID("user-test-id"),
					fixture.WithMovementIsPaid(false),
				)
				dbMovement := FromMovementDomain(original)
				db.WithContext(ctx).Create(&dbMovement)

				return repo, *original.ID, original
			},
			inputTx: func(repository *MovementRepository) *gorm.DB {
				return nil
			},
			updateMovement: fixture.MovementMock(
				fixture.WithMovementIsPaid(true),
			),
			expectedErr: nil,
			expectedMovement: fixture.MovementMock(
				fixture.WithMovementUserID("user-test-id"),
				fixture.WithMovementIsPaid(true),
			),
		},
		"should update movement is_paid successfully with external transaction": {
			prepareDB: func(ctx context.Context) (*MovementRepository, uuid.UUID, domain.Movement) {
				db := setupTestDB()
				repo := NewMovementRepository(db)

				original := fixture.MovementMock(
					fixture.WithMovementUserID("user-test-id"),
					fixture.WithMovementIsPaid(true),
				)
				dbMovement := FromMovementDomain(original)
				db.WithContext(ctx).Create(&dbMovement)

				return repo, *original.ID, original
			},
			inputTx: func(repository *MovementRepository) *gorm.DB {
				return repository.db.Begin()
			},
			updateMovement: fixture.MovementMock(
				fixture.WithMovementIsPaid(false),
			),
			expectedErr: nil,
			expectedMovement: fixture.MovementMock(
				fixture.WithMovementUserID("user-test-id"),
				fixture.WithMovementIsPaid(false),
			),
		},
		"should return error when movement not found": {
			prepareDB: func(ctx context.Context) (*MovementRepository, uuid.UUID, domain.Movement) {
				db := setupTestDB()
				repo := NewMovementRepository(db)

				return repo, uuid.New(), domain.Movement{}
			},
			inputTx: func(repository *MovementRepository) *gorm.DB {
				return nil
			},
			updateMovement: fixture.MovementMock(
				fixture.WithMovementIsPaid(true),
			),
			expectedErr:      fmt.Errorf("error updating movement: %w", ErrMovementNotFound),
			expectedMovement: domain.Movement{},
		},
		"should return error when database fails": {
			prepareDB: func(ctx context.Context) (*MovementRepository, uuid.UUID, domain.Movement) {
				db := setupTestDB()
				_ = db.Callback().Update().Before("gorm:update").Register("force_error", func(db *gorm.DB) {
					_ = db.AddError(assert.AnError)
				})
				repo := NewMovementRepository(db)

				original := fixture.MovementMock(
					fixture.WithMovementUserID("user-test-id"),
					fixture.WithMovementIsPaid(false),
				)
				dbMovement := FromMovementDomain(original)
				db.WithContext(ctx).Create(&dbMovement)

				return repo, *original.ID, original
			},
			inputTx: func(repository *MovementRepository) *gorm.DB {
				return nil
			},
			updateMovement: fixture.MovementMock(
				fixture.WithMovementIsPaid(true),
			),
			expectedErr:      fmt.Errorf("error updating movement: %w: %s", ErrDatabaseError, assert.AnError.Error()),
			expectedMovement: domain.Movement{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := createTestContext()
			repo, id, _ := tc.prepareDB(ctx)
			tx := tc.inputTx(repo)
			defer func() {
				if tx != nil {
					tx.Rollback()
				}
			}()

			result, err := repo.UpdateIsPaid(ctx, tx, id, tc.updateMovement)

			assert.Equal(t, tc.expectedMovement.Description, result.Description)
			assert.Equal(t, tc.expectedMovement.Amount, result.Amount)
			assert.Equal(t, tc.expectedMovement.UserID, result.UserID)
			assert.Equal(t, tc.expectedMovement.IsPaid, result.IsPaid)
			assert.Equal(t, tc.expectedErr, err)
		})
	}
}

func TestMovementRepository_UpdateOne(t *testing.T) {
	tests := map[string]struct {
		prepareDB        func(ctx context.Context) (*MovementRepository, uuid.UUID, domain.Movement)
		inputTx          func(repository *MovementRepository) *gorm.DB
		updateMovement   domain.Movement
		expectedErr      error
		expectedMovement domain.Movement
	}{
		"should update movement successfully with local transaction": {
			prepareDB: func(ctx context.Context) (*MovementRepository, uuid.UUID, domain.Movement) {
				db := setupTestDB()
				repo := NewMovementRepository(db)

				original := fixture.MovementMock(
					fixture.WithMovementUserID("user-test-id"),
					fixture.WithMovementDescription("Original Description"),
					fixture.AsMovementExpense(100.00),
				)
				dbMovement := FromMovementDomain(original)
				db.WithContext(ctx).Create(&dbMovement)

				return repo, *original.ID, original
			},
			inputTx: func(repository *MovementRepository) *gorm.DB {
				return nil
			},
			updateMovement: fixture.MovementMock(
				fixture.WithMovementDescription("Updated Description"),
				fixture.AsMovementExpense(200.00),
			),
			expectedErr: nil,
			expectedMovement: fixture.MovementMock(
				fixture.WithMovementUserID("user-test-id"),
				fixture.WithMovementDescription("Updated Description"),
				fixture.AsMovementExpense(200.00),
			),
		},
		"should update movement successfully with external transaction": {
			prepareDB: func(ctx context.Context) (*MovementRepository, uuid.UUID, domain.Movement) {
				db := setupTestDB()
				repo := NewMovementRepository(db)

				original := fixture.MovementMock(
					fixture.WithMovementUserID("user-test-id"),
					fixture.WithMovementDescription("Original Description"),
					fixture.AsMovementExpense(100.00),
				)
				dbMovement := FromMovementDomain(original)
				db.WithContext(ctx).Create(&dbMovement)

				dbModel := MovementDB{}
				db.First(&dbModel, fmt.Sprintf("%s.id = ?", "movements"), uuid.MustParse("11111111-1111-1111-1111-111111111111"))
				return repo, *original.ID, original
			},
			inputTx: func(repository *MovementRepository) *gorm.DB {
				return repository.db.Begin()
			},
			updateMovement: fixture.MovementMock(
				fixture.WithMovementDescription("Updated Description"),
				fixture.AsMovementExpense(300.00),
			),
			expectedErr: nil,
			expectedMovement: fixture.MovementMock(
				fixture.WithMovementUserID("user-test-id"),
				fixture.WithMovementDescription("Updated Description"),
				fixture.AsMovementExpense(300.00),
			),
		},
		"should return error when movement not found": {
			prepareDB: func(ctx context.Context) (*MovementRepository, uuid.UUID, domain.Movement) {
				db := setupTestDB()
				repo := NewMovementRepository(db)

				return repo, uuid.New(), domain.Movement{}
			},
			inputTx: func(repository *MovementRepository) *gorm.DB {
				return nil
			},
			updateMovement:   fixture.MovementMock(),
			expectedErr:      fmt.Errorf("error updating movement: %w", ErrMovementNotFound),
			expectedMovement: domain.Movement{},
		},
		"should return error when database fails": {
			prepareDB: func(ctx context.Context) (*MovementRepository, uuid.UUID, domain.Movement) {
				db := setupTestDB()
				_ = db.Callback().Update().Before("gorm:update").Register("force_error", func(db *gorm.DB) {
					_ = db.AddError(assert.AnError)
				})
				repo := NewMovementRepository(db)

				original := fixture.MovementMock(
					fixture.WithMovementUserID("user-test-id"),
				)
				dbMovement := FromMovementDomain(original)
				db.WithContext(ctx).Create(&dbMovement)

				return repo, *original.ID, original
			},
			inputTx: func(repository *MovementRepository) *gorm.DB {
				return nil
			},
			updateMovement: fixture.MovementMock(
				fixture.WithMovementDescription("Updated Description"),
			),
			expectedErr:      fmt.Errorf("error updating movement: %w: %s", ErrDatabaseError, assert.AnError.Error()),
			expectedMovement: domain.Movement{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := createTestContext()
			repo, id, _ := tc.prepareDB(ctx)
			tx := tc.inputTx(repo)
			defer func() {
				if tx != nil {
					tx.Rollback()
				}
			}()

			result, err := repo.UpdateOne(ctx, tx, id, tc.updateMovement)

			assert.Equal(t, tc.expectedMovement.Description, result.Description)
			assert.Equal(t, tc.expectedMovement.Amount, result.Amount)
			assert.Equal(t, tc.expectedMovement.UserID, result.UserID)
			assert.Equal(t, tc.expectedErr, err)
		})
	}
}

func TestMovementRepository_FindByInvoiceID(t *testing.T) {
	tests := map[string]struct {
		prepareDB         func() (*MovementRepository, uuid.UUID)
		expectedMovements int
		expectedErr       error
	}{
		"should find movements by invoice id successfully": {
			prepareDB: func() (*MovementRepository, uuid.UUID) {
				db := setupTestDB()
				repo := NewMovementRepository(db)
				ctx := createTestContext()

				invoiceID := uuid.New()

				// Movimento 1 - associado à fatura
				movement1 := fixture.MovementMock(
					fixture.WithMovementUserID("user-test-id"),
					fixture.WithMovementDescription("Compra 1"),
					fixture.AsMovementExpense(100.00),
				)
				movement1.InvoiceID = &invoiceID
				dbMovement1 := FromMovementDomain(movement1)
				db.WithContext(ctx).Create(&dbMovement1)

				// Movimento 2 - associado à fatura
				movement2 := fixture.MovementMock(
					fixture.WithMovementUserID("user-test-id"),
					fixture.WithMovementDescription("Compra 2"),
					fixture.AsMovementExpense(200.00),
				)
				movement2.ID = &[]uuid.UUID{uuid.New()}[0]
				movement2.InvoiceID = &invoiceID
				dbMovement2 := FromMovementDomain(movement2)
				db.WithContext(ctx).Create(&dbMovement2)

				// Movimento 3 - de outro usuário (não deve aparecer)
				movement3 := fixture.MovementMock(
					fixture.WithMovementUserID("other-user"),
					fixture.WithMovementDescription("Compra 3"),
					fixture.AsMovementExpense(300.00),
				)
				movement3.ID = &[]uuid.UUID{uuid.New()}[0]
				movement3.InvoiceID = &invoiceID
				dbMovement3 := FromMovementDomain(movement3)
				db.WithContext(ctx).Create(&dbMovement3)

				// Movimento 4 - de outra fatura (não deve aparecer)
				otherInvoiceID := uuid.New()
				movement4 := fixture.MovementMock(
					fixture.WithMovementUserID("user-test-id"),
					fixture.WithMovementDescription("Compra 4"),
					fixture.AsMovementExpense(400.00),
				)
				movement4.ID = &[]uuid.UUID{uuid.New()}[0]
				movement4.InvoiceID = &otherInvoiceID
				dbMovement4 := FromMovementDomain(movement4)
				db.WithContext(ctx).Create(&dbMovement4)

				return repo, invoiceID
			},
			expectedMovements: 2,
			expectedErr:       nil,
		},
		"should return empty list when no movements found for invoice": {
			prepareDB: func() (*MovementRepository, uuid.UUID) {
				db := setupTestDB()
				repo := NewMovementRepository(db)
				return repo, uuid.New()
			},
			expectedMovements: 0,
			expectedErr:       nil,
		},
		"should fail when database query fails": {
			prepareDB: func() (*MovementRepository, uuid.UUID) {
				db := setupTestDB()
				_ = db.Callback().Query().Before("gorm:query").Register("force_error", func(db *gorm.DB) {
					_ = db.AddError(assert.AnError)
				})
				repo := NewMovementRepository(db)
				return repo, uuid.New()
			},
			expectedMovements: 0,
			expectedErr:       fmt.Errorf("error finding movements by invoice id: %w: %s", ErrDatabaseError, assert.AnError.Error()),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			repo, invoiceID := tc.prepareDB()
			ctx := createTestContext()

			results, err := repo.FindByInvoiceID(ctx, invoiceID)

			assert.Equal(t, tc.expectedErr, err)
			if tc.expectedErr == nil {
				assert.Len(t, results, tc.expectedMovements)

				// Verificar se todas as movimentações retornadas pertencem à fatura correta
				for _, movement := range results {
					if movement.InvoiceID != nil {
						assert.Equal(t, invoiceID, *movement.InvoiceID)
					}
					assert.Equal(t, "user-test-id", movement.UserID)
				}
			}
		})
	}
}
