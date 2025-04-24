package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"personal-finance/internal/domain/fixture"
	"personal-finance/internal/plataform/authentication"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupSubCategoryTestDB() *gorm.DB {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	_ = db.AutoMigrate(&SubCategoryDB{})

	return db
}

func TestSubCategoryRepository_IsSubCategoryBelongsToCategory(t *testing.T) {
	tests := map[string]struct {
		prepareDB      func() *SubCategoryRepository
		subCategoryID  uuid.UUID
		categoryID     uuid.UUID
		expectedResult bool
		expectedErr    error
	}{
		"should return true when subcategory belongs to category": {
			prepareDB: func() *SubCategoryRepository {
				db := setupSubCategoryTestDB()
				repo := NewSubCategoryRepository(db)

				subCategory := FromSubCategoryDomain(fixture.SubCategoryMock())
				repo.db.Create(&subCategory)

				return repo
			},
			subCategoryID:  fixture.SCID,
			categoryID:     fixture.SCCategoryID,
			expectedResult: true,
			expectedErr:    nil,
		},
		"should return false when subcategory does not belong to category": {
			prepareDB: func() *SubCategoryRepository {
				db := setupSubCategoryTestDB()
				repo := NewSubCategoryRepository(db)

				subCategory := FromSubCategoryDomain(fixture.SubCategoryMock(
					fixture.WithSubCategoryCategoryID(fixture.SCCategoryID),
				))
				repo.db.Create(&subCategory)

				return repo
			},
			subCategoryID:  fixture.SCID,
			categoryID:     fixture.SCOtherCategoryID,
			expectedResult: false,
			expectedErr:    nil,
		},
		"should return false when subcategory does not exist": {
			prepareDB: func() *SubCategoryRepository {
				db := setupSubCategoryTestDB()
				repo := NewSubCategoryRepository(db)

				return repo
			},
			subCategoryID:  uuid.New(),
			categoryID:     fixture.SCCategoryID,
			expectedResult: false,
			expectedErr:    nil,
		},
		"should return error when database query fails": {
			prepareDB: func() *SubCategoryRepository {
				db := setupSubCategoryTestDB()

				_ = db.Callback().Query().Before("gorm:query").Register("error_callback", func(db *gorm.DB) {
					_ = db.AddError(errors.New("database error"))
				})

				return NewSubCategoryRepository(db)
			},
			subCategoryID:  fixture.SCID,
			categoryID:     fixture.SCCategoryID,
			expectedResult: false,
			expectedErr:    fmt.Errorf("error checking if subcategory belongs to category: %w", errors.New("database error")),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			repo := tc.prepareDB()
			ctx := context.WithValue(context.Background(), authentication.UserID, "user-test-id")

			result, err := repo.IsSubCategoryBelongsToCategory(ctx, tc.subCategoryID, tc.categoryID)

			assert.Equal(t, tc.expectedErr, err)
			assert.Equal(t, tc.expectedResult, result)
		})
	}
}
