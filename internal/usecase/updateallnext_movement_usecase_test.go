package usecase

import (
	"context"
	"testing"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/domain/fixture"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

func TestMovement_UpdateAllNext(t *testing.T) {
	updateDate := time.Date(2023, 3, 15, 0, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		id            string
		newMovement   domain.Movement
		mockSetup     func(mockMovRepo *MockMovementRepository, mockTxManager *MockTransactionManager)
		expectedError error
	}{
		"should reject updating an internal transfer leg and all next": {
			id: fixture.MovementID.String(),
			newMovement: fixture.MovementMock(
				fixture.AsMovementExpense(50.0),
			),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockTxManager *MockTransactionManager) {
				transferLeg := fixture.MovementMock(
					fixture.WithMovementIsPaid(true),
				)
				transferLeg.TypePayment = domain.TypePaymentInternalTransfer

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(ErrTransferMustUsePairEndpoint)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(transferLeg, nil)
			},
			expectedError: ErrTransferMustUsePairEndpoint,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockMovRepo := new(MockMovementRepository)
			mockTxManager := new(MockTransactionManager)

			tt.mockSetup(mockMovRepo, mockTxManager)

			uc := NewMovement(
				mockMovRepo,
				new(MockRecurrentRepository),
				new(MockWalletRepository),
				new(MockSubCategory),
				new(MockInvoiceRepository),
				new(MockInvoice),
				new(MockCreditCardRepository),
				mockTxManager,
				nil,
			)

			newMovement := tt.newMovement
			newMovement.Date = &updateDate

			_, err := uc.UpdateAllNext(context.Background(), uuid.MustParse(tt.id), newMovement)

			assert.Equal(t, tt.expectedError, err)

			mockMovRepo.AssertExpectations(t)
			mockTxManager.AssertExpectations(t)
		})
	}
}
