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

func TestPayTransfer_Settle(t *testing.T) {
	var (
		pairID   = uuid.MustParse("99999999-9999-9999-9999-999999999999")
		originID = uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
		destID   = uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")

		pay = func(u *PayTransfer, ctx context.Context, id uuid.UUID) (TransferOutput, error) {
			return u.Execute(ctx, id)
		}
		revert = func(u *PayTransfer, ctx context.Context, id uuid.UUID) (TransferOutput, error) {
			return u.Revert(ctx, id)
		}
	)

	transferPair := func(isPaid bool) domain.MovementList {
		return domain.MovementList{
			{ID: &originID, Amount: -500, WalletID: &originWalletID, IsPaid: isPaid, TypePayment: domain.TypePaymentInternalTransfer, PairID: &pairID},
			{ID: &destID, Amount: 500, WalletID: &destinationWalletID, IsPaid: isPaid, TypePayment: domain.TypePaymentInternalTransfer, PairID: &pairID},
		}
	}

	type expected struct {
		err error
	}

	tests := map[string]struct {
		action    func(u *PayTransfer, ctx context.Context, id uuid.UUID) (TransferOutput, error)
		mockSetup func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager)
		expected  expected
	}{
		"should settle both legs and move both balances when transfer is pending": {
			action: pay,
			mockSetup: func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				mockMovRepo.On("FindByPairID", pairID).Return(transferPair(false), nil)

				originWallet := fixture.WalletMock(fixture.WithWalletID(originWalletID), fixture.WithWalletBalance(1000.0))
				destWallet := fixture.WalletMock(fixture.WithWalletID(destinationWalletID), fixture.WithWalletBalance(500.0))
				mockWalletRepo.On("FindByID", &originWalletID).Return(originWallet, nil)
				mockWalletRepo.On("FindByID", &destinationWalletID).Return(destWallet, nil)

				mockWalletRepo.On("UpdateAmount", mock.Anything, &originWalletID, 500.0).Return(nil).Once()
				mockWalletRepo.On("UpdateAmount", mock.Anything, &destinationWalletID, 1000.0).Return(nil).Once()

				mockMovRepo.On("UpdateIsPaid", mock.Anything, originID, mock.MatchedBy(func(m domain.Movement) bool {
					return m.IsPaid
				})).Return(domain.Movement{ID: &originID, IsPaid: true}, nil).Once()
				mockMovRepo.On("UpdateIsPaid", mock.Anything, destID, mock.MatchedBy(func(m domain.Movement) bool {
					return m.IsPaid
				})).Return(domain.Movement{ID: &destID, IsPaid: true}, nil).Once()

				mockTxManager.On("WithTransaction", mock.Anything).Return(nil)
			},
			expected: expected{err: nil},
		},
		"should revert both legs and undo both balances when transfer is paid": {
			action: revert,
			mockSetup: func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				mockMovRepo.On("FindByPairID", pairID).Return(transferPair(true), nil)

				originWallet := fixture.WalletMock(fixture.WithWalletID(originWalletID), fixture.WithWalletBalance(500.0))
				destWallet := fixture.WalletMock(fixture.WithWalletID(destinationWalletID), fixture.WithWalletBalance(1000.0))
				mockWalletRepo.On("FindByID", &originWalletID).Return(originWallet, nil)
				mockWalletRepo.On("FindByID", &destinationWalletID).Return(destWallet, nil)

				mockWalletRepo.On("UpdateAmount", mock.Anything, &originWalletID, 1000.0).Return(nil).Once()
				mockWalletRepo.On("UpdateAmount", mock.Anything, &destinationWalletID, 500.0).Return(nil).Once()

				mockMovRepo.On("UpdateIsPaid", mock.Anything, originID, mock.MatchedBy(func(m domain.Movement) bool {
					return !m.IsPaid
				})).Return(domain.Movement{ID: &originID}, nil).Once()
				mockMovRepo.On("UpdateIsPaid", mock.Anything, destID, mock.MatchedBy(func(m domain.Movement) bool {
					return !m.IsPaid
				})).Return(domain.Movement{ID: &destID}, nil).Once()

				mockTxManager.On("WithTransaction", mock.Anything).Return(nil)
			},
			expected: expected{err: nil},
		},
		"should return ErrMovementAlreadyPaid when paying a settled transfer": {
			action: pay,
			mockSetup: func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				mockMovRepo.On("FindByPairID", pairID).Return(transferPair(true), nil)
			},
			expected: expected{err: ErrMovementAlreadyPaid},
		},
		"should return ErrMovementNotPaid when reverting a pending transfer": {
			action: revert,
			mockSetup: func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				mockMovRepo.On("FindByPairID", pairID).Return(transferPair(false), nil)
			},
			expected: expected{err: ErrMovementNotPaid},
		},
		"should return ErrTransferPairInconsistentPayment when the legs disagree on is_paid": {
			action: pay,
			mockSetup: func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				pair := domain.MovementList{
					{ID: &originID, Amount: -500, WalletID: &originWalletID, IsPaid: true, TypePayment: domain.TypePaymentInternalTransfer, PairID: &pairID},
					{ID: &destID, Amount: 500, WalletID: &destinationWalletID, IsPaid: false, TypePayment: domain.TypePaymentInternalTransfer, PairID: &pairID},
				}
				mockMovRepo.On("FindByPairID", pairID).Return(pair, nil)
			},
			expected: expected{err: ErrTransferPairInconsistentPayment},
		},
		"should return ErrInsufficientBalance when origin cannot cover the transfer": {
			action: pay,
			mockSetup: func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				mockMovRepo.On("FindByPairID", pairID).Return(transferPair(false), nil)

				originWallet := fixture.WalletMock(fixture.WithWalletID(originWalletID), fixture.WithWalletBalance(100.0))
				destWallet := fixture.WalletMock(fixture.WithWalletID(destinationWalletID), fixture.WithWalletBalance(500.0))
				mockWalletRepo.On("FindByID", &originWalletID).Return(originWallet, nil)
				mockWalletRepo.On("FindByID", &destinationWalletID).Return(destWallet, nil)

				mockTxManager.On("WithTransaction", mock.Anything).Return(nil)
			},
			expected: expected{err: ErrInsufficientBalance},
		},
		"should return ErrInsufficientBalance when destination no longer holds the transferred amount": {
			action: revert,
			mockSetup: func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				mockMovRepo.On("FindByPairID", pairID).Return(transferPair(true), nil)

				originWallet := fixture.WalletMock(fixture.WithWalletID(originWalletID), fixture.WithWalletBalance(500.0))
				destWallet := fixture.WalletMock(fixture.WithWalletID(destinationWalletID), fixture.WithWalletBalance(200.0))
				mockWalletRepo.On("FindByID", &originWalletID).Return(originWallet, nil)
				mockWalletRepo.On("FindByID", &destinationWalletID).Return(destWallet, nil)

				mockTxManager.On("WithTransaction", mock.Anything).Return(nil)
			},
			expected: expected{err: ErrInsufficientBalance},
		},
		"should return ErrTransferPairNotFound when pair has less than two movements": {
			action: pay,
			mockSetup: func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				mockMovRepo.On("FindByPairID", pairID).Return(domain.MovementList{
					{ID: &originID, Amount: -500, TypePayment: domain.TypePaymentInternalTransfer, PairID: &pairID},
				}, nil)
			},
			expected: expected{err: ErrTransferPairNotFound},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Arrange
			mockMovRepo := new(MockMovementRepository)
			mockWalletRepo := new(MockWalletRepository)
			mockTxManager := new(MockTransactionManager)
			tc.mockSetup(mockMovRepo, mockWalletRepo, mockTxManager)

			usecase := NewPayTransfer(mockMovRepo, mockWalletRepo, mockTxManager)

			// Act
			_, err := tc.action(&usecase, context.Background(), pairID)

			// Assert
			assert.ErrorIs(t, err, tc.expected.err)
			mockMovRepo.AssertExpectations(t)
			mockWalletRepo.AssertExpectations(t)
			mockTxManager.AssertExpectations(t)
		})
	}
}
