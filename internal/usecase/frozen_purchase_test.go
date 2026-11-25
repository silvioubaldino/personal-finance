package usecase

import (
	"context"
	"testing"

	"personal-finance/internal/domain"
	"personal-finance/internal/domain/fixture"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestMovement_ensureCreditCardPurchasesNotFrozen(t *testing.T) {
	groupID := uuid.New()
	laterInvoiceID := uuid.New()

	purchase := func(invoiceID uuid.UUID, isPaid bool) domain.Movement {
		m := fixture.MovementMock(
			fixture.AsMovementExpense(100.0),
			fixture.WithMovementTypePayment(string(domain.TypePaymentCreditCard)),
			fixture.WithMovementCreditCardID(&fixture.CreditCardID),
			fixture.WithMovementIsPaid(isPaid),
		)
		m.CreditCardInfo.InvoiceID = &invoiceID

		return m
	}

	installment := func(invoiceID uuid.UUID, number, total int) domain.Movement {
		m := purchase(invoiceID, false)
		m.CreditCardInfo.InstallmentGroupID = &groupID
		m.CreditCardInfo.InstallmentNumber = &number
		m.CreditCardInfo.TotalInstallments = &total

		return m
	}

	tests := map[string]struct {
		existing      domain.Movement
		mockSetup     func(mockMovRepo *MockMovementRepository, mockInvoiceRepo *MockInvoiceRepository)
		expectedError error
	}{
		"should allow when invoice is open": {
			existing: purchase(fixture.InvoiceID, false),
			mockSetup: func(_ *MockMovementRepository, mockInvoiceRepo *MockInvoiceRepository) {
				mockInvoiceRepo.On("FindByID", fixture.InvoiceID).
					Return(fixture.InvoiceMock(fixture.WithInvoiceIsPaid(false)), nil)
			},
		},
		"should reject when invoice is paid": {
			existing: purchase(fixture.InvoiceID, false),
			mockSetup: func(_ *MockMovementRepository, mockInvoiceRepo *MockInvoiceRepository) {
				mockInvoiceRepo.On("FindByID", fixture.InvoiceID).
					Return(fixture.InvoiceMock(fixture.WithInvoiceIsPaid(true)), nil)
			},
			expectedError: domain.ErrConflict,
		},
		"should reject when purchase itself is paid": {
			existing:      purchase(fixture.InvoiceID, true),
			mockSetup:     func(_ *MockMovementRepository, _ *MockInvoiceRepository) {},
			expectedError: domain.ErrConflict,
		},
		"should reject installment when a later invoice is paid": {
			existing: installment(fixture.InvoiceID, 1, 2),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockInvoiceRepo *MockInvoiceRepository) {
				first := installment(fixture.InvoiceID, 1, 2)
				second := installment(laterInvoiceID, 2, 2)
				mockMovRepo.On("FindByInstallmentGroupFromNumber", groupID, 1).
					Return(domain.MovementList{first, second}, nil)
				mockInvoiceRepo.On("FindByID", fixture.InvoiceID).
					Return(fixture.InvoiceMock(fixture.WithInvoiceIsPaid(false)), nil)
				mockInvoiceRepo.On("FindByID", laterInvoiceID).
					Return(fixture.InvoiceMock(fixture.WithInvoiceIsPaid(true)), nil)
			},
			expectedError: domain.ErrConflict,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockMovRepo := new(MockMovementRepository)
			mockInvoiceRepo := new(MockInvoiceRepository)
			tt.mockSetup(mockMovRepo, mockInvoiceRepo)

			uc := &Movement{movementRepo: mockMovRepo, invoiceRepo: mockInvoiceRepo}

			err := uc.ensureCreditCardPurchasesNotFrozen(context.Background(), &tt.existing)

			if tt.expectedError != nil {
				assert.ErrorIs(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
			mockMovRepo.AssertExpectations(t)
			mockInvoiceRepo.AssertExpectations(t)
		})
	}
}
