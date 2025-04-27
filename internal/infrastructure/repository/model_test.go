package repository

import (
	"testing"

	"personal-finance/internal/domain/fixture"

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
