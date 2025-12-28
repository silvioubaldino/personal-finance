package repository

import (
	"context"
	"testing"

	"personal-finance/internal/domain"
	"personal-finance/internal/plataform/authentication"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupUserPreferencesTestDB() *gorm.DB {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	_ = db.AutoMigrate(&UserPreferencesDB{})
	return db
}

func createUserPreferencesTestContext() context.Context {
	return context.WithValue(context.Background(), authentication.UserID, "user-test-id")
}

func TestUserPreferencesRepository_GetOrCreateDefaults(t *testing.T) {
	tests := map[string]struct {
		prepareDB       func() *UserPreferencesRepository
		expectedErr     error
		expectedPrefs   domain.UserPreferences
		checkOnlyFields bool
	}{
		"should create defaults when no preferences exist": {
			prepareDB: func() *UserPreferencesRepository {
				db := setupUserPreferencesTestDB()
				return NewUserPreferencesRepository(db)
			},
			expectedPrefs: domain.UserPreferences{
				UserID:   "user-test-id",
				Language: domain.DefaultLanguage,
				Currency: domain.DefaultCurrency,
			},
			checkOnlyFields: true,
			expectedErr:     nil,
		},
		"should return existing preferences": {
			prepareDB: func() *UserPreferencesRepository {
				db := setupUserPreferencesTestDB()
				repo := NewUserPreferencesRepository(db)

				// Create existing preferences
				ctx := createUserPreferencesTestContext()
				_, _ = repo.Upsert(ctx, domain.UserPreferences{
					Language: "en-US",
					Currency: "USD",
				})

				return repo
			},
			expectedPrefs: domain.UserPreferences{
				UserID:   "user-test-id",
				Language: "en-US",
				Currency: "USD",
			},
			checkOnlyFields: true,
			expectedErr:     nil,
		},
		"should not overwrite existing preferences with defaults": {
			prepareDB: func() *UserPreferencesRepository {
				db := setupUserPreferencesTestDB()
				repo := NewUserPreferencesRepository(db)

				// Create existing custom preferences
				ctx := createUserPreferencesTestContext()
				_, _ = repo.Upsert(ctx, domain.UserPreferences{
					Language: "es-ES",
					Currency: "EUR",
				})

				return repo
			},
			expectedPrefs: domain.UserPreferences{
				UserID:   "user-test-id",
				Language: "es-ES",
				Currency: "EUR",
			},
			checkOnlyFields: true,
			expectedErr:     nil,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			repo := tc.prepareDB()
			ctx := createUserPreferencesTestContext()

			result, err := repo.GetOrCreateDefaults(ctx)

			if tc.expectedErr != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tc.checkOnlyFields {
					assert.Equal(t, tc.expectedPrefs.UserID, result.UserID)
					assert.Equal(t, tc.expectedPrefs.Language, result.Language)
					assert.Equal(t, tc.expectedPrefs.Currency, result.Currency)
					assert.NotZero(t, result.DateCreate)
					assert.NotZero(t, result.DateUpdate)
				}
			}
		})
	}
}

func TestUserPreferencesRepository_Upsert(t *testing.T) {
	tests := map[string]struct {
		prepareDB     func() *UserPreferencesRepository
		input         domain.UserPreferences
		expectedErr   error
		expectedPrefs domain.UserPreferences
	}{
		"should insert new preferences": {
			prepareDB: func() *UserPreferencesRepository {
				db := setupUserPreferencesTestDB()
				return NewUserPreferencesRepository(db)
			},
			input: domain.UserPreferences{
				Language: "pt-BR",
				Currency: "BRL",
			},
			expectedPrefs: domain.UserPreferences{
				UserID:   "user-test-id",
				Language: "pt-BR",
				Currency: "BRL",
			},
			expectedErr: nil,
		},
		"should update existing preferences": {
			prepareDB: func() *UserPreferencesRepository {
				db := setupUserPreferencesTestDB()
				repo := NewUserPreferencesRepository(db)

				// Create initial preferences
				ctx := createUserPreferencesTestContext()
				_, _ = repo.Upsert(ctx, domain.UserPreferences{
					Language: "pt-BR",
					Currency: "BRL",
				})

				return repo
			},
			input: domain.UserPreferences{
				Language: "en-US",
				Currency: "USD",
			},
			expectedPrefs: domain.UserPreferences{
				UserID:   "user-test-id",
				Language: "en-US",
				Currency: "USD",
			},
			expectedErr: nil,
		},
		"should preserve user_id from context": {
			prepareDB: func() *UserPreferencesRepository {
				db := setupUserPreferencesTestDB()
				return NewUserPreferencesRepository(db)
			},
			input: domain.UserPreferences{
				UserID:   "should-be-ignored",
				Language: "fr-FR",
				Currency: "EUR",
			},
			expectedPrefs: domain.UserPreferences{
				UserID:   "user-test-id",
				Language: "fr-FR",
				Currency: "EUR",
			},
			expectedErr: nil,
		},
		"should use default currency when not provided": {
			prepareDB: func() *UserPreferencesRepository {
				db := setupUserPreferencesTestDB()
				return NewUserPreferencesRepository(db)
			},
			input: domain.UserPreferences{
				Language: "en-US",
				Currency: "",
			},
			expectedPrefs: domain.UserPreferences{
				UserID:   "user-test-id",
				Language: "en-US",
				Currency: domain.DefaultCurrency,
			},
			expectedErr: nil,
		},
		"should preserve existing currency when updating without currency": {
			prepareDB: func() *UserPreferencesRepository {
				db := setupUserPreferencesTestDB()
				repo := NewUserPreferencesRepository(db)

				// Create initial preferences with custom currency
				ctx := createUserPreferencesTestContext()
				_, _ = repo.Upsert(ctx, domain.UserPreferences{
					Language: "pt-BR",
					Currency: "EUR",
				})

				return repo
			},
			input: domain.UserPreferences{
				Language: "en-US",
				Currency: "",
			},
			expectedPrefs: domain.UserPreferences{
				UserID:   "user-test-id",
				Language: "en-US",
				Currency: "EUR",
			},
			expectedErr: nil,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			repo := tc.prepareDB()
			ctx := createUserPreferencesTestContext()

			result, err := repo.Upsert(ctx, tc.input)

			if tc.expectedErr != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedPrefs.UserID, result.UserID)
				assert.Equal(t, tc.expectedPrefs.Language, result.Language)
				assert.Equal(t, tc.expectedPrefs.Currency, result.Currency)
				assert.NotZero(t, result.DateCreate)
				assert.NotZero(t, result.DateUpdate)
			}
		})
	}
}
