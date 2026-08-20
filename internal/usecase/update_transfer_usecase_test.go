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
)

func TestUpdateTransfer_Execute(t *testing.T) {
	var (
		pairID     = uuid.MustParse("66666666-6666-6666-6666-666666666666")
		originID   = uuid.MustParse("77777777-7777-7777-7777-777777777777")
		destID     = uuid.MustParse("88888888-8888-8888-8888-888888888888")
		updateDate = time.Date(2023, 7, 1, 10, 0, 0, 0, time.UTC)
	)

	type (
		input struct {
			in UpdateTransferInput
		}
		expected struct {
			err error
		}
	)

	tests := map[string]struct {
		input     input
		mockSetup func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager)
		expected  expected
	}{
		"should update both legs and re-settle balances when transfer is paid": {
			input: input{in: UpdateTransferInput{
				PairID:              pairID,
				OriginWalletID:      originWalletID,
				DestinationWalletID: destinationWalletID,
				Amount:              400.0,
				Date:                updateDate,
			}},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				pair := domain.MovementList{
					{ID: &originID, Amount: -500, WalletID: &originWalletID, IsPaid: true, TypePayment: domain.TypePaymentInternalTransfer, PairID: &pairID},
					{ID: &destID, Amount: 500, WalletID: &destinationWalletID, IsPaid: true, TypePayment: domain.TypePaymentInternalTransfer, PairID: &pairID},
				}
				mockMovRepo.On("FindByPairID", pairID).Return(pair, nil)

				// FindByID is stateless in this mock: it always reports the stored balance,
				// exactly like the real repository, which reads outside the transaction. The
				// usecase must therefore net the revert and the apply in memory and write each
				// wallet once.
				originWallet := fixture.WalletMock(fixture.WithWalletID(originWalletID), fixture.WithWalletBalance(500.0))
				destWallet := fixture.WalletMock(fixture.WithWalletID(destinationWalletID), fixture.WithWalletBalance(1000.0))
				mockWalletRepo.On("FindByID", &originWalletID).Return(originWallet, nil)
				mockWalletRepo.On("FindByID", &destinationWalletID).Return(destWallet, nil)

				// origin: 500 + 500 (revert) - 400 (apply); destination: 1000 - 500 + 400
				mockWalletRepo.On("UpdateAmount", mock.Anything, &originWalletID, 600.0).Return(nil).Once()
				mockWalletRepo.On("UpdateAmount", mock.Anything, &destinationWalletID, 900.0).Return(nil).Once()

				mockMovRepo.On("Update", mock.Anything, originID, mock.MatchedBy(func(m domain.Movement) bool {
					return m.Amount == -400.0
				})).Return(domain.Movement{ID: &originID, Amount: -400.0}, nil)
				mockMovRepo.On("Update", mock.Anything, destID, mock.MatchedBy(func(m domain.Movement) bool {
					return m.Amount == 400.0
				})).Return(domain.Movement{ID: &destID, Amount: 400.0}, nil)

				mockTxManager.On("WithTransaction", mock.Anything).Return(nil)
			},
			expected: expected{err: nil},
		},
		"should return ErrSameWalletTransfer when origin and destination wallets are the same": {
			input: input{in: UpdateTransferInput{
				PairID:              pairID,
				OriginWalletID:      originWalletID,
				DestinationWalletID: originWalletID,
				Amount:              500.0,
				Date:                updateDate,
			}},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
			},
			expected: expected{err: ErrSameWalletTransfer},
		},
		"should return ErrInvalidTransferAmount when amount is not positive": {
			input: input{in: UpdateTransferInput{
				PairID:              pairID,
				OriginWalletID:      originWalletID,
				DestinationWalletID: destinationWalletID,
				Amount:              0,
				Date:                updateDate,
			}},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
			},
			expected: expected{err: ErrInvalidTransferAmount},
		},
		"should return ErrDateRequired when date is zero": {
			input: input{in: UpdateTransferInput{
				PairID:              pairID,
				OriginWalletID:      originWalletID,
				DestinationWalletID: destinationWalletID,
				Amount:              500.0,
				Date:                time.Time{},
			}},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
			},
			expected: expected{err: ErrDateRequired},
		},
		"should return ErrTransferPairNotFound when pair has less than two movements": {
			input: input{in: UpdateTransferInput{
				PairID:              pairID,
				OriginWalletID:      originWalletID,
				DestinationWalletID: destinationWalletID,
				Amount:              500.0,
				Date:                updateDate,
			}},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				mockMovRepo.On("FindByPairID", pairID).Return(domain.MovementList{
					{ID: &originID, Amount: -500, TypePayment: domain.TypePaymentInternalTransfer, PairID: &pairID},
				}, nil)
			},
			expected: expected{err: ErrTransferPairNotFound},
		},
		"should return ErrInsufficientBalance when new amount exceeds origin balance": {
			input: input{in: UpdateTransferInput{
				PairID:              pairID,
				OriginWalletID:      originWalletID,
				DestinationWalletID: destinationWalletID,
				Amount:              5000.0,
				Date:                updateDate,
			}},
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

				// Balance is checked against the reverted balance (500 + 500), and no wallet
				// is written when the check rejects the update.
				mockTxManager.On("WithTransaction", mock.Anything).Return(nil)
			},
			expected: expected{err: ErrInsufficientBalance},
		},
		"should not double-credit the destination when only the amount changes": {
			input: input{in: UpdateTransferInput{
				PairID:              pairID,
				OriginWalletID:      originWalletID,
				DestinationWalletID: destinationWalletID,
				Amount:              5.0,
				Date:                updateDate,
			}},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				// Regression: a paid transfer of 10 is edited down to 5. The destination must
				// settle at 5, not at 15 — the old leg has to be reverted before the new one
				// is applied.
				pair := domain.MovementList{
					{ID: &originID, Amount: -10, WalletID: &originWalletID, IsPaid: true, TypePayment: domain.TypePaymentInternalTransfer, PairID: &pairID},
					{ID: &destID, Amount: 10, WalletID: &destinationWalletID, IsPaid: true, TypePayment: domain.TypePaymentInternalTransfer, PairID: &pairID},
				}
				mockMovRepo.On("FindByPairID", pairID).Return(pair, nil)

				originWallet := fixture.WalletMock(fixture.WithWalletID(originWalletID), fixture.WithWalletBalance(90.0))
				destWallet := fixture.WalletMock(fixture.WithWalletID(destinationWalletID), fixture.WithWalletBalance(10.0))
				mockWalletRepo.On("FindByID", &originWalletID).Return(originWallet, nil)
				mockWalletRepo.On("FindByID", &destinationWalletID).Return(destWallet, nil)

				mockWalletRepo.On("UpdateAmount", mock.Anything, &originWalletID, 95.0).Return(nil).Once()
				mockWalletRepo.On("UpdateAmount", mock.Anything, &destinationWalletID, 5.0).Return(nil).Once()

				mockMovRepo.On("Update", mock.Anything, originID, mock.MatchedBy(func(m domain.Movement) bool {
					return m.Amount == -5.0
				})).Return(domain.Movement{ID: &originID, Amount: -5.0}, nil)
				mockMovRepo.On("Update", mock.Anything, destID, mock.MatchedBy(func(m domain.Movement) bool {
					return m.Amount == 5.0
				})).Return(domain.Movement{ID: &destID, Amount: 5.0}, nil)

				mockTxManager.On("WithTransaction", mock.Anything).Return(nil)
			},
			expected: expected{err: nil},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Arrange
			mockMovRepo := new(MockMovementRepository)
			mockWalletRepo := new(MockWalletRepository)
			mockTxManager := new(MockTransactionManager)
			tc.mockSetup(mockMovRepo, mockWalletRepo, mockTxManager)

			usecase := NewUpdateTransfer(mockMovRepo, mockWalletRepo, mockTxManager)

			// Act
			_, err := usecase.Execute(context.Background(), tc.input.in)

			// Assert
			assert.ErrorIs(t, err, tc.expected.err)
			mockMovRepo.AssertExpectations(t)
			mockWalletRepo.AssertExpectations(t)
			mockTxManager.AssertExpectations(t)
		})
	}
}
