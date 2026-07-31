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

func setupUserTestDB() *gorm.DB {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	_ = db.AutoMigrate(&UserDB{})
	return db
}

func createUserTestContext() context.Context {
	return context.WithValue(context.Background(), authentication.UserID, "user-test-id")
}

func TestUserRepository_EnsureExists(t *testing.T) {
	t.Run("creates row with defaults when missing", func(t *testing.T) {
		db := setupUserTestDB()
		repo := NewUserRepository(db)

		created, err := repo.EnsureExists(context.Background(), "user-test-id")
		assert.NoError(t, err)
		assert.True(t, created)

		var saved UserDB
		_ = db.Where("id = ?", "user-test-id").First(&saved).Error
		assert.Equal(t, domain.DefaultLanguage, saved.Language)
		assert.Equal(t, domain.DefaultCurrency, saved.Currency)
	})

	t.Run("reports created only on the first of repeated calls for the same new user", func(t *testing.T) {
		db := setupUserTestDB()
		repo := NewUserRepository(db)

		first, err := repo.EnsureExists(context.Background(), "user-test-id")
		assert.NoError(t, err)
		assert.True(t, first)

		second, err := repo.EnsureExists(context.Background(), "user-test-id")
		assert.NoError(t, err)
		assert.False(t, second)
	})

	t.Run("is idempotent, does not overwrite, and reports no creation", func(t *testing.T) {
		db := setupUserTestDB()
		repo := NewUserRepository(db)
		ctx := createUserTestContext()

		_, _ = repo.Update(ctx, domain.User{Language: "en-US", Currency: "USD"})

		created, err := repo.EnsureExists(context.Background(), "user-test-id")
		assert.NoError(t, err)
		assert.False(t, created)

		var saved UserDB
		_ = db.Where("id = ?", "user-test-id").First(&saved).Error
		assert.Equal(t, "en-US", saved.Language)
		assert.Equal(t, "USD", saved.Currency)
	})
}

func TestUserRepository_Get(t *testing.T) {
	tests := map[string]struct {
		prepare      func() *UserRepository
		expectedLang string
		expectedCurr string
	}{
		"creates defaults when missing": {
			prepare: func() *UserRepository {
				return NewUserRepository(setupUserTestDB())
			},
			expectedLang: domain.DefaultLanguage,
			expectedCurr: domain.DefaultCurrency,
		},
		"returns existing values": {
			prepare: func() *UserRepository {
				db := setupUserTestDB()
				repo := NewUserRepository(db)
				ctx := createUserTestContext()
				_, _ = repo.Update(ctx, domain.User{Language: "en-US", Currency: "USD"})
				return repo
			},
			expectedLang: "en-US",
			expectedCurr: "USD",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			repo := tc.prepare()
			ctx := createUserTestContext()

			result, err := repo.Get(ctx)

			assert.NoError(t, err)
			assert.Equal(t, "user-test-id", result.ID)
			assert.Equal(t, tc.expectedLang, result.Language)
			assert.Equal(t, tc.expectedCurr, result.Currency)
			assert.NotZero(t, result.CreatedAt)
			assert.NotZero(t, result.UpdatedAt)
		})
	}
}

func TestUserRepository_Update(t *testing.T) {
	tests := map[string]struct {
		prepare      func() *UserRepository
		input        domain.User
		expectedLang string
		expectedCurr string
	}{
		"insert new with both fields": {
			prepare: func() *UserRepository {
				return NewUserRepository(setupUserTestDB())
			},
			input:        domain.User{Language: "pt-BR", Currency: "BRL"},
			expectedLang: "pt-BR",
			expectedCurr: "BRL",
		},
		"update existing both fields": {
			prepare: func() *UserRepository {
				db := setupUserTestDB()
				repo := NewUserRepository(db)
				ctx := createUserTestContext()
				_, _ = repo.Update(ctx, domain.User{Language: "pt-BR", Currency: "BRL"})
				return repo
			},
			input:        domain.User{Language: "en-US", Currency: "USD"},
			expectedLang: "en-US",
			expectedCurr: "USD",
		},
		"insert with only language uses default currency": {
			prepare: func() *UserRepository {
				return NewUserRepository(setupUserTestDB())
			},
			input:        domain.User{Language: "en-US"},
			expectedLang: "en-US",
			expectedCurr: domain.DefaultCurrency,
		},
		"insert with only currency uses default language": {
			prepare: func() *UserRepository {
				return NewUserRepository(setupUserTestDB())
			},
			input:        domain.User{Currency: "USD"},
			expectedLang: domain.DefaultLanguage,
			expectedCurr: "USD",
		},
		"update preserves existing currency when only language given": {
			prepare: func() *UserRepository {
				db := setupUserTestDB()
				repo := NewUserRepository(db)
				ctx := createUserTestContext()
				_, _ = repo.Update(ctx, domain.User{Language: "pt-BR", Currency: "EUR"})
				return repo
			},
			input:        domain.User{Language: "en-US"},
			expectedLang: "en-US",
			expectedCurr: "EUR",
		},
		"update preserves existing language when only currency given": {
			prepare: func() *UserRepository {
				db := setupUserTestDB()
				repo := NewUserRepository(db)
				ctx := createUserTestContext()
				_, _ = repo.Update(ctx, domain.User{Language: "es-ES", Currency: "EUR"})
				return repo
			},
			input:        domain.User{Currency: "USD"},
			expectedLang: "es-ES",
			expectedCurr: "USD",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			repo := tc.prepare()
			ctx := createUserTestContext()

			result, err := repo.Update(ctx, tc.input)

			assert.NoError(t, err)
			assert.Equal(t, "user-test-id", result.ID)
			assert.Equal(t, tc.expectedLang, result.Language)
			assert.Equal(t, tc.expectedCurr, result.Currency)
			assert.NotZero(t, result.CreatedAt)
			assert.NotZero(t, result.UpdatedAt)
		})
	}
}

func TestUserRepository_Delete(t *testing.T) {
	db := setupUserTestDB()
	repo := NewUserRepository(db)
	ctx := createUserTestContext()

	_, _ = repo.Update(ctx, domain.User{Language: "en-US", Currency: "USD"})

	err := repo.Delete(context.Background(), nil, "user-test-id")
	assert.NoError(t, err)

	var count int64
	_ = db.Model(&UserDB{}).Where("id = ?", "user-test-id").Count(&count).Error
	assert.Equal(t, int64(0), count)
}
