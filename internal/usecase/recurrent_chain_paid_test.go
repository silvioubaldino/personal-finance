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

func newMovementForChainTest(movRepo *MockMovementRepository, walletRepo *MockWalletRepository) Movement {
	return NewMovement(
		movRepo,
		new(MockRecurrentRepository),
		walletRepo,
		new(MockSubCategory),
		new(MockInvoiceRepository),
		new(MockInvoice),
		new(MockCreditCardRepository),
		new(MockTransactionManager),
		nil,
	)
}

func TestMovement_removeMaterializedFrom(t *testing.T) {
	recurrentID := uuid.New()
	cutoff := time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC)

	earlierID := uuid.New()
	laterPaidID := uuid.New()
	laterPendingID := uuid.New()

	earlier := fixture.MovementMock(fixture.WithMovementID(earlierID), fixture.AsMovementExpense(100.0),
		fixture.WithMovementDate(time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)), fixture.WithMovementIsPaid(true))
	laterPaid := fixture.MovementMock(fixture.WithMovementID(laterPaidID), fixture.AsMovementExpense(100.0),
		fixture.WithMovementDate(time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)), fixture.WithMovementIsPaid(true))
	laterPending := fixture.MovementMock(fixture.WithMovementID(laterPendingID), fixture.AsMovementExpense(100.0),
		fixture.WithMovementDate(time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)), fixture.WithMovementIsPaid(false))

	// Arrange
	movRepo := new(MockMovementRepository)
	walletRepo := new(MockWalletRepository)
	wallet := fixture.WalletMock(fixture.WithWalletBalance(900))

	movRepo.On("FindAllByRecurrentID", recurrentID).Return(domain.MovementList{earlier, laterPaid, laterPending}, nil)
	walletRepo.On("FindByID", laterPaid.WalletID).Return(wallet, nil)
	walletRepo.On("UpdateAmount", mock.Anything, wallet.ID, 1000.0).Return(nil)
	movRepo.On("Delete", mock.Anything, laterPaidID).Return(nil)
	movRepo.On("Delete", mock.Anything, laterPendingID).Return(nil)

	uc := newMovementForChainTest(movRepo, walletRepo)

	// Act
	err := uc.removeMaterializedFrom(context.Background(), nil, recurrentID, cutoff, &earlierID)

	// Assert
	assert.NoError(t, err)
	movRepo.AssertNotCalled(t, "Delete", mock.Anything, earlierID)
	movRepo.AssertExpectations(t)
	walletRepo.AssertExpectations(t)
}

func TestMovement_movePaidFollowingToChain(t *testing.T) {
	oldRecurrentID := uuid.New()
	newRecurrentID := uuid.New()
	from := time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC)

	paidID := uuid.New()
	pendingID := uuid.New()
	paid := fixture.MovementMock(fixture.WithMovementID(paidID), fixture.AsMovementExpense(100.0),
		fixture.WithMovementDate(time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)), fixture.WithMovementIsPaid(true))
	pending := fixture.MovementMock(fixture.WithMovementID(pendingID), fixture.AsMovementExpense(100.0),
		fixture.WithMovementDate(time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)), fixture.WithMovementIsPaid(false))

	// Arrange
	movRepo := new(MockMovementRepository)
	movRepo.On("FindAllByRecurrentID", oldRecurrentID).Return(domain.MovementList{paid, pending}, nil)
	movRepo.On("Update", mock.Anything, paidID, mock.MatchedBy(func(m domain.Movement) bool {
		return m.RecurrentID != nil && *m.RecurrentID == newRecurrentID && m.Amount == paid.Amount
	})).Return(paid, nil)

	uc := newMovementForChainTest(movRepo, new(MockWalletRepository))

	// Act
	err := uc.movePaidFollowingToChain(context.Background(), nil, oldRecurrentID, nil, newRecurrentID, from)

	// Assert
	assert.NoError(t, err)
	movRepo.AssertNotCalled(t, "Update", mock.Anything, pendingID, mock.Anything)
	movRepo.AssertExpectations(t)
}
