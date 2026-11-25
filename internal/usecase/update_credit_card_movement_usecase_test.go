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

type updateCreditCardMocks struct {
	movRepo     *MockMovementRepository
	invoiceRepo *MockInvoiceRepository
	invoiceUC   *MockInvoice
	cardRepo    *MockCreditCardRepository
	txManager   *MockTransactionManager
}

func newUpdateCreditCardUsecase() (Movement, updateCreditCardMocks) {
	m := updateCreditCardMocks{
		movRepo:     new(MockMovementRepository),
		invoiceRepo: new(MockInvoiceRepository),
		invoiceUC:   new(MockInvoice),
		cardRepo:    new(MockCreditCardRepository),
		txManager:   new(MockTransactionManager),
	}
	m.txManager.On("WithTransaction", mock.Anything).Return(nil)

	return NewMovement(
		m.movRepo,
		new(MockRecurrentRepository),
		new(MockWalletRepository),
		new(MockSubCategory),
		m.invoiceRepo,
		m.invoiceUC,
		m.cardRepo,
		m.txManager,
		nil,
	), m
}

func creditCardPurchase(id uuid.UUID, amount float64, date time.Time, invoiceID uuid.UUID) domain.Movement {
	movement := fixture.MovementMock(
		fixture.WithMovementTypePayment(string(domain.TypePaymentCreditCard)),
		fixture.WithMovementCreditCardID(&fixture.CreditCardID),
		fixture.WithMovementDate(date),
		fixture.WithMovementIsPaid(false),
	)
	movement.ID = &id
	movement.Amount = -amount
	movement.CreditCardInfo.InvoiceID = &invoiceID

	return movement
}

func invoiceWith(id uuid.UUID, amount float64, paid bool) domain.Invoice {
	invoice := fixture.InvoiceMock(fixture.WithInvoiceAmount(amount), fixture.WithInvoiceIsPaid(paid))
	invoice.ID = &id

	return invoice
}

func TestMovement_UpdateOne_CreditCard_DateMovesInvoice(t *testing.T) {
	march := time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC)
	afterClosing := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	fromID, toID := uuid.New(), uuid.New()

	tests := map[string]struct {
		target        domain.Invoice
		expectedError error
	}{
		"should move the purchase to the invoice of the new date": {
			target: invoiceWith(toID, 0, false),
		},
		"should reject moving the purchase to a paid invoice": {
			target:        invoiceWith(toID, -300, true),
			expectedError: domain.ErrConflict,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Arrange
			usecase, m := newUpdateCreditCardUsecase()
			id := uuid.New()
			existing := creditCardPurchase(id, 200, march, fromID)
			input := creditCardPurchase(id, 200, afterClosing, fromID)
			input.CreditCardInfo = nil

			m.movRepo.On("FindByID", id).Return(existing, nil)
			m.invoiceRepo.On("FindByID", fromID).Return(invoiceWith(fromID, -200, false), nil)
			m.invoiceUC.On("FindOrCreateInvoiceForMovement", mock.Anything, (*uuid.UUID)(nil), &fixture.CreditCardID, afterClosing).
				Return(tt.target, nil)
			m.cardRepo.On("FindByID", fixture.CreditCardID).Return(fixture.CreditCardMock(), nil).Maybe()

			if tt.expectedError == nil {
				m.invoiceRepo.On("UpdateAmount", mock.Anything, fromID, 0.0).Return(domain.Invoice{}, nil).Once()
				m.invoiceRepo.On("UpdateAmount", mock.Anything, toID, -200.0).Return(domain.Invoice{}, nil).Once()
				m.movRepo.On("Update", mock.Anything, id, mock.MatchedBy(func(mv domain.Movement) bool {
					return mv.CreditCardInfo != nil && *mv.CreditCardInfo.InvoiceID == toID
				})).Return(input, nil).Once()
			}

			// Act
			_, err := usecase.UpdateOne(context.Background(), id, input)

			// Assert
			assert.ErrorIs(t, err, tt.expectedError)
			m.invoiceRepo.AssertExpectations(t)
			m.movRepo.AssertExpectations(t)
			m.cardRepo.AssertNotCalled(t, "UpdateLimitDelta", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestMovement_UpdateAllNext_CreditCard(t *testing.T) {
	march := time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC)
	invoiceIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}

	t.Run("should update a standalone purchase like updating only it", func(t *testing.T) {
		// Arrange
		usecase, m := newUpdateCreditCardUsecase()
		id := uuid.New()
		existing := creditCardPurchase(id, 200, march, invoiceIDs[0])
		input := creditCardPurchase(id, 250, march, invoiceIDs[0])
		input.CreditCardInfo = nil

		m.movRepo.On("FindByID", id).Return(existing, nil)
		m.invoiceRepo.On("FindByID", invoiceIDs[0]).Return(invoiceWith(invoiceIDs[0], -200, false), nil)
		m.cardRepo.On("FindByID", fixture.CreditCardID).Return(fixture.CreditCardMock(), nil)
		m.invoiceRepo.On("UpdateAmount", mock.Anything, invoiceIDs[0], -250.0).Return(domain.Invoice{}, nil).Once()
		m.cardRepo.On("UpdateLimitDelta", mock.Anything, fixture.CreditCardID, -50.0).Return(domain.CreditCard{}, nil).Once()
		m.movRepo.On("Update", mock.Anything, id, mock.Anything).Return(input, nil).Once()

		// Act
		result, err := usecase.UpdateAllNext(context.Background(), id, input)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, input.Amount, result.Amount)
		m.invoiceRepo.AssertExpectations(t)
		m.cardRepo.AssertExpectations(t)
		m.movRepo.AssertExpectations(t)
	})

	t.Run("should rewrite the current and next installments, each on its own invoice", func(t *testing.T) {
		// Arrange
		usecase, m := newUpdateCreditCardUsecase()
		groupID := uuid.New()
		ids := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
		installments := make(domain.MovementList, 0, 2)
		for i := 1; i <= 2; i++ {
			number, total := i+1, 3
			purchase := creditCardPurchase(ids[i], 400, march.AddDate(0, i, 0), invoiceIDs[i])
			purchase.CreditCardInfo.InstallmentGroupID = &groupID
			purchase.CreditCardInfo.InstallmentNumber = &number
			purchase.CreditCardInfo.TotalInstallments = &total
			installments = append(installments, purchase)
		}
		input := creditCardPurchase(ids[1], 500, installments[0].Date.UTC(), invoiceIDs[1])
		input.CreditCardInfo = nil

		m.movRepo.On("FindByID", ids[1]).Return(installments[0], nil)
		m.movRepo.On("FindByInstallmentGroupFromNumber", groupID, 2).Return(installments, nil)
		m.cardRepo.On("FindByID", fixture.CreditCardID).Return(fixture.CreditCardMock(), nil)
		m.cardRepo.On("UpdateLimitDelta", mock.Anything, fixture.CreditCardID, -100.0).Return(domain.CreditCard{}, nil).Twice()
		for i := 1; i <= 2; i++ {
			m.invoiceRepo.On("FindByID", invoiceIDs[i]).Return(invoiceWith(invoiceIDs[i], -400, false), nil)
			m.invoiceRepo.On("UpdateAmount", mock.Anything, invoiceIDs[i], -500.0).Return(domain.Invoice{}, nil).Once()
			m.movRepo.On("Update", mock.Anything, ids[i], mock.MatchedBy(func(mv domain.Movement) bool {
				return mv.Amount == -500
			})).Return(installments[i-1], nil).Once()
		}

		// Act
		_, err := usecase.UpdateAllNext(context.Background(), ids[1], input)

		// Assert
		assert.NoError(t, err)
		m.invoiceRepo.AssertExpectations(t)
		m.cardRepo.AssertExpectations(t)
		m.movRepo.AssertExpectations(t)
	})
}
