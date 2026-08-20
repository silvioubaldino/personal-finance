package repository

import (
	"context"
	"errors"
	"testing"

	"personal-finance/internal/domain"
	"personal-finance/internal/plataform/authentication"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupCategoryTestDB() *gorm.DB {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	_ = db.AutoMigrate(&CategoryDB{}, &SubCategoryDB{})

	return db
}

func TestCategoryRepository_Delete(t *testing.T) {
	userID := "user-1"
	ctx := context.WithValue(context.Background(), authentication.UserID, userID)

	t.Run("returns a conflict error when the category has subcategories", func(t *testing.T) {
		db := setupCategoryTestDB()
		id := uuid.New()
		db.Create(&CategoryDB{ID: &id, UserID: userID})

		_ = db.Callback().Delete().Before("gorm:delete").Register("force_fk_violation", func(tx *gorm.DB) {
			_ = tx.AddError(gorm.ErrForeignKeyViolated)
		})

		repo := NewCategoryRepository(db)
		err := repo.Delete(ctx, id)

		assert.True(t, errors.Is(err, ErrCategoryHasSubcategories))
		assert.True(t, errors.Is(err, domain.ErrConflict))
	})

	t.Run("returns not found when the category does not exist", func(t *testing.T) {
		db := setupCategoryTestDB()
		repo := NewCategoryRepository(db)

		err := repo.Delete(ctx, uuid.New())

		assert.True(t, errors.Is(err, domain.ErrNotFound))
		assert.False(t, errors.Is(err, ErrCategoryHasSubcategories))
	})

	t.Run("deletes the category successfully", func(t *testing.T) {
		db := setupCategoryTestDB()
		id := uuid.New()
		db.Create(&CategoryDB{ID: &id, UserID: userID})

		repo := NewCategoryRepository(db)
		err := repo.Delete(ctx, id)

		assert.NoError(t, err)
	})
}

func TestCategoryRepository_FindAll_ExcludesInternalTransferCategories(t *testing.T) {
	userID := "user-1"
	ctx := context.WithValue(context.Background(), authentication.UserID, userID)

	db := setupCategoryTestDB()

	userCategoryID := uuid.New()
	db.Create(&CategoryDB{ID: &userCategoryID, UserID: userID, Description: "Alimentação"})

	defaultCategoryID := uuid.New()
	db.Create(&CategoryDB{ID: &defaultCategoryID, UserID: DefaultCategoryUserID, Description: "Salário"})

	transferOutID := uuid.MustParse(domain.InternalTransferOutCategoryID)
	db.Create(&CategoryDB{ID: &transferOutID, UserID: DefaultCategoryUserID, Description: "Transferência interna - saída"})

	transferInID := uuid.MustParse(domain.InternalTransferInCategoryID)
	db.Create(&CategoryDB{ID: &transferInID, UserID: DefaultCategoryUserID, Description: "Transferência interna - entrada"})

	repo := NewCategoryRepository(db)
	categories, err := repo.FindAll(ctx)

	assert.NoError(t, err)
	ids := make([]uuid.UUID, 0, len(categories))
	for _, c := range categories {
		ids = append(ids, *c.ID)
	}
	assert.ElementsMatch(t, []uuid.UUID{userCategoryID, defaultCategoryID}, ids)
	assert.NotContains(t, ids, transferOutID)
	assert.NotContains(t, ids, transferInID)
}
