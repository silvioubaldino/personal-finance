package usecase

import (
	"context"
	"testing"

	"personal-finance/internal/domain"
	"personal-finance/internal/domain/fixture"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDeleteTransfer_Execute(t *testing.T) {
	var (
		pairID   = uuid.MustParse("33333333-3333-3333-3333-333333333333")
		originID = uuid.MustParse("44444444-4444-4444-4444-444444444444")
		destID   = uuid.MustParse("55555555-5555-5555-5555-555555555555")
	)

	type (
		expected struct {
			err error
		}
	)

	tests := map[string]struct {
		mockSetup func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager)
		expected  expected
	}{
		"should delete both legs and revert balances when transfer is paid": {
			mockSetup: func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				pair := domain.MovementList{
					{ID: &originID, Amount: -500, WalletID: &originWalletID, IsPaid: true, TypePayment: domain.TypePaymentInternalTransfer, PairID: &pairID},
					{ID: &destID, Amount: 500, WalletID: &destinationWalletID, IsPaid: true, TypePayment: domain.TypePaymentInternalTransfer, PairID: &pairID},
				}
				mockMovRepo.On("FindByPairID", pairID).Return(pair, nil)

				originWallet := fixture.WalletMock(fixture.WithWalletID(originWalletID), fixture.WithWalletBalance(500.0))
				destWallet := fixture.WalletMock(fixture.WithWalletID(destinationWalletID), fixture.WithWalletBalance(1000.0))
				mockWalletRepo.On("FindByID", &originWalletID).Return(originWallet, nil)
				mockWalletRepo.On("FindByID", &destinationWalletID).Return(destWallet, nil)
				mockWalletRepo.On("UpdateAmount", mock.Anything, &originWalletID, 1000.0).Return(nil)
				mockWalletRepo.On("UpdateAmount", mock.Anything, &destinationWalletID, 500.0).Return(nil)

				mockMovRepo.On("Delete", mock.Anything, originID).Return(nil)
				mockMovRepo.On("Delete", mock.Anything, destID).Return(nil)

				mockTxManager.On("WithTransaction", mock.Anything).Return(nil)
			},
			expected: expected{err: nil},
		},
		"should delete both legs without touching balances when transfer is not paid": {
			mockSetup: func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				pair := domain.MovementList{
					{ID: &originID, Amount: -500, WalletID: &originWalletID, IsPaid: false, TypePayment: domain.TypePaymentInternalTransfer, PairID: &pairID},
					{ID: &destID, Amount: 500, WalletID: &destinationWalletID, IsPaid: false, TypePayment: domain.TypePaymentInternalTransfer, PairID: &pairID},
				}
				mockMovRepo.On("FindByPairID", pairID).Return(pair, nil)

				mockMovRepo.On("Delete", mock.Anything, originID).Return(nil)
				mockMovRepo.On("Delete", mock.Anything, destID).Return(nil)

				mockTxManager.On("WithTransaction", mock.Anything).Return(nil)
			},
			expected: expected{err: nil},
		},
		"should return ErrTransferPairNotFound when pair has less than two movements": {
			mockSetup: func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				mockMovRepo.On("FindByPairID", pairID).Return(domain.MovementList{
					{ID: &originID, Amount: -500, TypePayment: domain.TypePaymentInternalTransfer, PairID: &pairID},
				}, nil)
			},
			expected: expected{err: ErrTransferPairNotFound},
		},
		"should return ErrMovementNotInternalTransfer when a leg is not an internal transfer": {
			mockSetup: func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				pair := domain.MovementList{
					{ID: &originID, Amount: -500, TypePayment: domain.TypePaymentMoney, PairID: &pairID},
					{ID: &destID, Amount: 500, TypePayment: domain.TypePaymentInternalTransfer, PairID: &pairID},
				}
				mockMovRepo.On("FindByPairID", pairID).Return(pair, nil)
			},
			expected: expected{err: ErrMovementNotInternalTransfer},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Arrange
			mockMovRepo := new(MockMovementRepository)
			mockWalletRepo := new(MockWalletRepository)
			mockTxManager := new(MockTransactionManager)
			tc.mockSetup(mockMovRepo, mockWalletRepo, mockTxManager)

			usecase := NewDeleteTransfer(mockMovRepo, mockWalletRepo, mockTxManager)

			// Act
			err := usecase.Execute(context.Background(), pairID)

			// Assert
			assert.ErrorIs(t, err, tc.expected.err)
			mockMovRepo.AssertExpectations(t)
			mockWalletRepo.AssertExpectations(t)
			mockTxManager.AssertExpectations(t)
		})
	}
}
