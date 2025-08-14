package repository

import (
	"context"
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

func setupCreditCardTestDB() *gorm.DB {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	_ = db.AutoMigrate(&CreditCardDB{}, &WalletDB{})

	return db
}

func createCreditCardTestContext() context.Context {
	return context.WithValue(context.Background(), authentication.UserID, "user-test-id")
}

func TestCreditCardRepository_Add(t *testing.T) {
	tests := map[string]struct {
		prepareDB          func() *CreditCardRepository
		input              domain.CreditCard
		inputTx            func(repository *CreditCardRepository) *gorm.DB
		expectedErr        error
		expectedCreditCard domain.CreditCard
	}{
		"should add credit card successfully": {
			prepareDB: func() *CreditCardRepository {
				db := setupCreditCardTestDB()
				return NewCreditCardRepository(db)
			},
			input: fixture.CreditCardMock(),
			inputTx: func(repository *CreditCardRepository) *gorm.DB {
				tx := repository.db.Begin()
				return tx
			},
			expectedCreditCard: fixture.CreditCardMock(
				fixture.WithCreditCardUserID("user-test-id"),
			),
			expectedErr: nil,
		},
		"should fail when adding credit card with database error": {
			prepareDB: func() *CreditCardRepository {
				db := setupCreditCardTestDB()
				_ = db.Callback().Create().Before("gorm:create").Register("force_error", func(db *gorm.DB) {
					_ = db.AddError(assert.AnError)
				})
				return NewCreditCardRepository(db)
			},
			input: fixture.CreditCardMock(),
			inputTx: func(repository *CreditCardRepository) *gorm.DB {
				tx := repository.db.Begin()
				return tx
			},
			expectedCreditCard: domain.CreditCard{},
			expectedErr:        fmt.Errorf("error creating credit card: %w: %s", ErrDatabaseError, assert.AnError.Error()),
		},
		"should add credit card with external transaction": {
			prepareDB: func() *CreditCardRepository {
				db := setupCreditCardTestDB()
				return NewCreditCardRepository(db)
			},
			input: fixture.CreditCardMock(
				fixture.WithCreditCardName("Cartão Premium"),
				fixture.WithCreditCardLimit(10000.0),
			),
			inputTx: func(repository *CreditCardRepository) *gorm.DB {
				return nil
			},
			expectedCreditCard: fixture.CreditCardMock(
				fixture.WithCreditCardName("Cartão Premium"),
				fixture.WithCreditCardLimit(10000.0),
			),
			expectedErr: nil,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			repo := tc.prepareDB()
			tx := tc.inputTx(repo)
			ctx := createCreditCardTestContext()
			defer func() {
				if tx != nil {
					tx.Rollback()
				}
			}()

			result, err := repo.Add(ctx, tx, tc.input)

			assert.Equal(t, tc.expectedErr, err)
			if tc.expectedErr == nil {
				assert.NotNil(t, result.ID)
				assert.NotZero(t, result.DateCreate)
				assert.NotZero(t, result.DateUpdate)
				assert.Equal(t, tc.expectedCreditCard.UserID, result.UserID)
				assert.Equal(t, tc.expectedCreditCard.Name, result.Name)
				assert.Equal(t, tc.expectedCreditCard.CreditLimit, result.CreditLimit)
			}
		})
	}
}

func TestCreditCardRepository_FindByID(t *testing.T) {
	tests := map[string]struct {
		prepareDB          func() (*CreditCardRepository, uuid.UUID)
		expectedErr        error
		expectedCreditCard domain.CreditCard
	}{
		"should find credit card by id successfully": {
			prepareDB: func() (*CreditCardRepository, uuid.UUID) {
				db := setupCreditCardTestDB()
				repo := NewCreditCardRepository(db)

				creditCard := fixture.CreditCardMock()
				dbCreditCard := FromCreditCardDomain(creditCard)
				_ = db.Create(&dbCreditCard)

				return repo, *creditCard.ID
			},
			expectedCreditCard: fixture.CreditCardMock(),
			expectedErr:        nil,
		},
		"should fail when credit card not found": {
			prepareDB: func() (*CreditCardRepository, uuid.UUID) {
				db := setupCreditCardTestDB()
				repo := NewCreditCardRepository(db)
				return repo, uuid.New()
			},
			expectedCreditCard: domain.CreditCard{},
			expectedErr:        fmt.Errorf("error finding credit card: %w: %s", ErrCreditCardNotFound, "record not found"),
		},
		"should fail when database query fails": {
			prepareDB: func() (*CreditCardRepository, uuid.UUID) {
				db := setupCreditCardTestDB()
				_ = db.Callback().Query().Before("gorm:query").Register("force_error", func(db *gorm.DB) {
					_ = db.AddError(assert.AnError)
				})
				repo := NewCreditCardRepository(db)
				return repo, uuid.New()
			},
			expectedCreditCard: domain.CreditCard{},
			expectedErr:        fmt.Errorf("error finding credit card: %w: %s", ErrDatabaseError, assert.AnError.Error()),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			repo, id := tc.prepareDB()
			ctx := createCreditCardTestContext()

			result, err := repo.FindByID(ctx, id)

			assert.Equal(t, tc.expectedErr, err)
			if tc.expectedErr == nil {
				assert.Equal(t, tc.expectedCreditCard.Name, result.Name)
				assert.Equal(t, tc.expectedCreditCard.CreditLimit, result.CreditLimit)
				assert.Equal(t, tc.expectedCreditCard.ClosingDay, result.ClosingDay)
				assert.Equal(t, tc.expectedCreditCard.DueDay, result.DueDay)
			}
		})
	}
}

func TestCreditCardRepository_FindAll(t *testing.T) {
	tests := map[string]struct {
		prepareDB           func() *CreditCardRepository
		expectedCreditCards int
		expectedErr         error
	}{
		"should find all credit cards successfully": {
			prepareDB: func() *CreditCardRepository {
				db := setupCreditCardTestDB()
				repo := NewCreditCardRepository(db)

				// Cartão 1
				creditCard1 := fixture.CreditCardMock()
				dbCreditCard1 := FromCreditCardDomain(creditCard1)
				_ = db.Create(&dbCreditCard1)

				// Cartão 2
				creditCard2 := fixture.CreditCardMock(
					fixture.WithCreditCardName("Cartão 2"),
				)
				creditCard2.ID = &[]uuid.UUID{uuid.New()}[0]
				dbCreditCard2 := FromCreditCardDomain(creditCard2)
				_ = db.Create(&dbCreditCard2)

				// Cartão de outro usuário (não deve aparecer)
				creditCard3 := fixture.CreditCardMock(
					fixture.WithCreditCardUserID("other-user"),
				)
				creditCard3.ID = &[]uuid.UUID{uuid.New()}[0]
				dbCreditCard3 := FromCreditCardDomain(creditCard3)
				_ = db.Create(&dbCreditCard3)

				return repo
			},
			expectedCreditCards: 2,
			expectedErr:         nil,
		},
		"should return empty list when no credit cards found": {
			prepareDB: func() *CreditCardRepository {
				db := setupCreditCardTestDB()
				return NewCreditCardRepository(db)
			},
			expectedCreditCards: 0,
			expectedErr:         nil,
		},
		"should fail when database query fails": {
			prepareDB: func() *CreditCardRepository {
				db := setupCreditCardTestDB()
				_ = db.Callback().Query().Before("gorm:query").Register("force_error", func(db *gorm.DB) {
					_ = db.AddError(assert.AnError)
				})
				return NewCreditCardRepository(db)
			},
			expectedCreditCards: 0,
			expectedErr:         fmt.Errorf("error finding credit cards: %w: %s", ErrDatabaseError, assert.AnError.Error()),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			repo := tc.prepareDB()
			ctx := createCreditCardTestContext()

			results, err := repo.FindAll(ctx)

			assert.Equal(t, tc.expectedErr, err)
			if tc.expectedErr == nil {
				assert.Len(t, results, tc.expectedCreditCards)
			}
		})
	}
}

func TestCreditCardRepository_Update(t *testing.T) {
	tests := map[string]struct {
		prepareDB          func() (*CreditCardRepository, uuid.UUID)
		input              domain.CreditCard
		expectedErr        error
		expectedCreditCard domain.CreditCard
	}{
		"should update credit card successfully": {
			prepareDB: func() (*CreditCardRepository, uuid.UUID) {
				db := setupCreditCardTestDB()
				repo := NewCreditCardRepository(db)

				creditCard := fixture.CreditCardMock()
				dbCreditCard := FromCreditCardDomain(creditCard)
				_ = db.Create(&dbCreditCard)

				return repo, *creditCard.ID
			},
			input: fixture.CreditCardMock(
				fixture.WithCreditCardName("Cartão Atualizado"),
				fixture.WithCreditCardLimit(15000.0),
			),
			expectedCreditCard: fixture.CreditCardMock(
				fixture.WithCreditCardName("Cartão Atualizado"),
				fixture.WithCreditCardLimit(15000.0),
			),
			expectedErr: nil,
		},
		"should fail when credit card not found": {
			prepareDB: func() (*CreditCardRepository, uuid.UUID) {
				db := setupCreditCardTestDB()
				repo := NewCreditCardRepository(db)
				return repo, uuid.New()
			},
			input:              fixture.CreditCardMock(),
			expectedCreditCard: domain.CreditCard{},
			expectedErr:        fmt.Errorf("error finding credit card: %w: %s", ErrCreditCardNotFound, "record not found"),
		},
		"should fail when database query fails": {
			prepareDB: func() (*CreditCardRepository, uuid.UUID) {
				db := setupCreditCardTestDB()
				_ = db.Callback().Query().Before("gorm:query").Register("force_error", func(db *gorm.DB) {
					_ = db.AddError(assert.AnError)
				})
				repo := NewCreditCardRepository(db)
				return repo, uuid.New()
			},
			input:              fixture.CreditCardMock(),
			expectedCreditCard: domain.CreditCard{},
			expectedErr:        fmt.Errorf("error finding credit card: %w: %s", ErrDatabaseError, assert.AnError.Error()),
		},
		"should fail when database update fails": {
			prepareDB: func() (*CreditCardRepository, uuid.UUID) {
				db := setupCreditCardTestDB()
				repo := NewCreditCardRepository(db)

				creditCard := fixture.CreditCardMock()
				dbCreditCard := FromCreditCardDomain(creditCard)
				_ = db.Create(&dbCreditCard)

				_ = db.Callback().Update().Before("gorm:update").Register("force_error", func(db *gorm.DB) {
					_ = db.AddError(assert.AnError)
				})

				return repo, *creditCard.ID
			},
			input:              fixture.CreditCardMock(),
			expectedCreditCard: domain.CreditCard{},
			expectedErr:        fmt.Errorf("error updating credit card: %w: %s", ErrDatabaseError, assert.AnError.Error()),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			repo, id := tc.prepareDB()
			ctx := createCreditCardTestContext()

			result, err := repo.Update(ctx, nil, id, tc.input)

			assert.Equal(t, tc.expectedErr, err)
			if tc.expectedErr == nil {
				assert.Equal(t, tc.expectedCreditCard.Name, result.Name)
				assert.Equal(t, tc.expectedCreditCard.CreditLimit, result.CreditLimit)
			}
		})
	}
}

func TestCreditCardRepository_Delete(t *testing.T) {
	tests := map[string]struct {
		prepareDB   func() (*CreditCardRepository, uuid.UUID)
		expectedErr error
	}{
		"should delete credit card successfully": {
			prepareDB: func() (*CreditCardRepository, uuid.UUID) {
				db := setupCreditCardTestDB()
				repo := NewCreditCardRepository(db)

				creditCard := fixture.CreditCardMock()
				dbCreditCard := FromCreditCardDomain(creditCard)
				_ = db.Create(&dbCreditCard)

				return repo, *creditCard.ID
			},
			expectedErr: nil,
		},
		"should fail when credit card not found": {
			prepareDB: func() (*CreditCardRepository, uuid.UUID) {
				db := setupCreditCardTestDB()
				repo := NewCreditCardRepository(db)
				return repo, uuid.New()
			},
			expectedErr: fmt.Errorf("error finding credit card: %w: %s", ErrCreditCardNotFound, "record not found"),
		},
		"should fail when database query fails": {
			prepareDB: func() (*CreditCardRepository, uuid.UUID) {
				db := setupCreditCardTestDB()
				_ = db.Callback().Query().Before("gorm:query").Register("force_error", func(db *gorm.DB) {
					_ = db.AddError(assert.AnError)
				})
				repo := NewCreditCardRepository(db)
				return repo, uuid.New()
			},
			expectedErr: fmt.Errorf("error finding credit card: %w: %s", ErrDatabaseError, assert.AnError.Error()),
		},
		"should fail when database delete fails": {
			prepareDB: func() (*CreditCardRepository, uuid.UUID) {
				db := setupCreditCardTestDB()
				repo := NewCreditCardRepository(db)

				creditCard := fixture.CreditCardMock()
				dbCreditCard := FromCreditCardDomain(creditCard)
				_ = db.Create(&dbCreditCard)

				_ = db.Callback().Delete().Before("gorm:delete").Register("force_error", func(db *gorm.DB) {
					_ = db.AddError(assert.AnError)
				})

				return repo, *creditCard.ID
			},
			expectedErr: fmt.Errorf("error deleting credit card: %w: %s", ErrDatabaseError, assert.AnError.Error()),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			repo, id := tc.prepareDB()
			ctx := createCreditCardTestContext()

			err := repo.Delete(ctx, nil, id)

			assert.Equal(t, tc.expectedErr, err)
		})
	}
}
