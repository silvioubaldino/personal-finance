package repository_test

import (
	"context"
	"testing"
	"time"

	"personal-finance/internal/domain/estimate/repository"
	subRepo "personal-finance/internal/domain/subcategory/repository"
	"personal-finance/internal/model"
	"personal-finance/internal/plataform/authentication"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type stubSubCategoryRepo struct {
	subRepo.Repository
	store map[uuid.UUID]model.SubCategory
}

func (s *stubSubCategoryRepo) FindByID(ctx context.Context, id uuid.UUID) (model.SubCategory, error) {
	if v, ok := s.store[id]; ok {
		return v, nil
	}
	return model.SubCategory{}, gorm.ErrRecordNotFound
}

func setupTestDB() *gorm.DB {
	db, _ := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	_ = db.AutoMigrate(
		&model.EstimateCategories{},
		&model.EstimateSubCategories{},
		&model.Category{},
		&model.SubCategory{},
	)
	return db
}

func TestPgRepository_AddSubEstimate_AutoCreateParent(t *testing.T) {
	db := setupTestDB()

	categoryID := uuid.New()
	subCategoryID := uuid.New()
	userID := "user-123"

	// Create Category in DB to avoid FK issues if enforced (sqlite defaults often don't enforce, but good practice)
	// and to allow joins if needed.
	db.Create(&model.Category{
		ID:          &categoryID,
		Description: "Test Category",
		UserID:      userID,
		IsIncome:    false,
	})

	subCat := model.SubCategory{
		ID:          &subCategoryID,
		CategoryID:  &categoryID,
		UserID:      userID,
		Description: "SubCat 1",
	}

	stubRepo := &stubSubCategoryRepo{
		store: map[uuid.UUID]model.SubCategory{
			subCategoryID: subCat,
		},
	}

	repo := repository.NewPgRepository(db, stubRepo)
	ctx := context.WithValue(context.Background(), authentication.UserID, userID)

	// Test Case: Add SubEstimate without existing Parent
	subEst := model.EstimateSubCategories{
		SubCategoryID: &subCategoryID,
		Month:         time.January,
		Year:          2024,
		Amount:        100.0,
		UserID:        userID,
	}

	createdSub, err := repo.AddSubEstimate(ctx, subEst)
	require.NoError(t, err)
	assert.NotNil(t, createdSub.EstimateCategoryID)
	assert.Equal(t, 100.0, createdSub.Amount)

	// Verify Parent was Created
	var parent model.EstimateCategories
	err = db.First(&parent, "id = ?", createdSub.EstimateCategoryID).Error
	require.NoError(t, err)
	assert.Equal(t, categoryID, *parent.CategoryID)
	assert.Equal(t, 100.0, parent.Amount)
	assert.Equal(t, time.January, parent.Month)
	assert.Equal(t, 2024, parent.Year)
}
