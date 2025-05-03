package repository

import (
	"testing"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/domain/fixture"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestToMovementModel(t *testing.T) {
	domainMovement := fixture.MovementMock()

	dbModel := FromMovementDomain(domainMovement)

	assert.Equal(t, *domainMovement.ID, *dbModel.ID)
	assert.Equal(t, domainMovement.Description, dbModel.Description)
	assert.Equal(t, domainMovement.Amount, dbModel.Amount)
	assert.Equal(t, domainMovement.UserID, dbModel.UserID)
	assert.Equal(t, domainMovement.IsPaid, dbModel.IsPaid)
	assert.Equal(t, domainMovement.IsRecurrent, dbModel.RecurrentID != nil)
	assert.Equal(t, *domainMovement.WalletID, *dbModel.WalletID)
	assert.Equal(t, *domainMovement.CategoryID, *dbModel.CategoryID)
	assert.Equal(t, string(domainMovement.TypePayment), dbModel.TypePayment)
}

func TestMovementModelToDomain(t *testing.T) {
	domainMovement := fixture.MovementMock()
	dbModel := FromMovementDomain(domainMovement)

	resultDomain := dbModel.ToDomain()

	assert.Equal(t, *domainMovement.ID, *resultDomain.ID)
	assert.Equal(t, domainMovement.Description, resultDomain.Description)
	assert.Equal(t, domainMovement.Amount, resultDomain.Amount)
	assert.Equal(t, domainMovement.UserID, resultDomain.UserID)
	assert.Equal(t, domainMovement.IsPaid, resultDomain.IsPaid)
	assert.Equal(t, domainMovement.IsRecurrent, resultDomain.IsRecurrent)
	assert.Equal(t, *domainMovement.WalletID, *resultDomain.WalletID)
	assert.Equal(t, *domainMovement.CategoryID, *resultDomain.CategoryID)
	assert.Equal(t, domainMovement.TypePayment, resultDomain.TypePayment)
}

func TestToSubCategoryModel(t *testing.T) {
	domainSubCategory := fixture.SubCategoryMock()

	dbModel := FromSubCategoryDomain(domainSubCategory)

	assert.Equal(t, *domainSubCategory.ID, *dbModel.ID)
	assert.Equal(t, domainSubCategory.Description, dbModel.Description)
	assert.Equal(t, domainSubCategory.UserID, dbModel.UserID)
	assert.Equal(t, *domainSubCategory.CategoryID, *dbModel.CategoryID)
	assert.Equal(t, domainSubCategory.DateCreate, dbModel.DateCreate)
	assert.Equal(t, domainSubCategory.DateUpdate, dbModel.DateUpdate)
}

func TestSubCategoryModelToDomain(t *testing.T) {
	domainSubCategory := fixture.SubCategoryMock()
	dbModel := FromSubCategoryDomain(domainSubCategory)

	resultDomain := dbModel.ToDomain()

	assert.Equal(t, *domainSubCategory.ID, *resultDomain.ID)
	assert.Equal(t, domainSubCategory.Description, resultDomain.Description)
	assert.Equal(t, domainSubCategory.UserID, resultDomain.UserID)
	assert.Equal(t, *domainSubCategory.CategoryID, *resultDomain.CategoryID)
	assert.Equal(t, domainSubCategory.DateCreate, resultDomain.DateCreate)
	assert.Equal(t, domainSubCategory.DateUpdate, resultDomain.DateUpdate)
}

func TestCategoryDBMethods(t *testing.T) {
	t.Run("should convert from domain to DB model and back", func(t *testing.T) {
		// Arrange
		id := uuid.New()
		now := time.Now()

		domainCategory := domain.Category{
			ID:          &id,
			Description: "Test Category",
			UserID:      "test-user-id",
			IsIncome:    true,
			DateCreate:  now,
			DateUpdate:  now,
		}

		// Act
		dbModel := FromCategoryDomain(domainCategory)
		resultDomain := dbModel.ToDomain()

		// Assert
		assert.Equal(t, *domainCategory.ID, *resultDomain.ID)
		assert.Equal(t, domainCategory.Description, resultDomain.Description)
		assert.Equal(t, domainCategory.UserID, resultDomain.UserID)
		assert.Equal(t, domainCategory.IsIncome, resultDomain.IsIncome)
		assert.Equal(t, domainCategory.DateCreate.Unix(), resultDomain.DateCreate.Unix())
		assert.Equal(t, domainCategory.DateUpdate.Unix(), resultDomain.DateUpdate.Unix())
	})
}
