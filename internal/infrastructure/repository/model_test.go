package repository

import (
	"testing"

	"personal-finance/internal/domain"

	"github.com/stretchr/testify/assert"
)

func TestToMovementModel(t *testing.T) {
	// Arrange
	domainMovement := domain.MovementMock()

	// Act
	dbModel := ToMovementModel(domainMovement)

	// Assert
	assert.Equal(t, *domainMovement.ID, *dbModel.ID)
	assert.Equal(t, domainMovement.Description, dbModel.Description)
	assert.Equal(t, domainMovement.Amount, dbModel.Amount)
	assert.Equal(t, domainMovement.UserID, dbModel.UserID)
	assert.Equal(t, domainMovement.IsPaid, dbModel.IsPaid)
	assert.Equal(t, domainMovement.IsRecurrent, dbModel.IsRecurrent)
	assert.Equal(t, *domainMovement.WalletID, *dbModel.WalletID)
	assert.Equal(t, *domainMovement.CategoryID, *dbModel.CategoryID)
	assert.Equal(t, domainMovement.TypePaymentID, dbModel.TypePaymentID)
}

func TestMovementModelToDomain(t *testing.T) {
	// Arrange
	domainMovement := domain.MovementMock()
	dbModel := ToMovementModel(domainMovement)

	// Act
	resultDomain := dbModel.ToDomain()

	// Assert
	assert.Equal(t, *domainMovement.ID, *resultDomain.ID)
	assert.Equal(t, domainMovement.Description, resultDomain.Description)
	assert.Equal(t, domainMovement.Amount, resultDomain.Amount)
	assert.Equal(t, domainMovement.UserID, resultDomain.UserID)
	assert.Equal(t, domainMovement.IsPaid, resultDomain.IsPaid)
	assert.Equal(t, domainMovement.IsRecurrent, resultDomain.IsRecurrent)
	assert.Equal(t, *domainMovement.WalletID, *resultDomain.WalletID)
	assert.Equal(t, *domainMovement.CategoryID, *resultDomain.CategoryID)
	assert.Equal(t, domainMovement.TypePaymentID, resultDomain.TypePaymentID)
}
