package usecase

import (
	"context"
	"testing"

	"personal-finance/internal/domain"
	"personal-finance/internal/domain/fixture"
	"personal-finance/internal/infrastructure/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

func TestMovement_DeleteOne(t *testing.T) {
	tests := map[string]struct {
		id            string
		mockSetup     func(mockMovRepo *MockMovementRepository, mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository)
		expectedError error
	}{
		"should delete credit card movement with success": {
			id: fixture.MovementID.String(),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository) {
				existingMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Compra no cartão de crédito"),
					fixture.AsMovementExpense(100.0),
					fixture.WithMovementTypePayment(string(domain.TypePaymentCreditCard)),
					fixture.WithMovementCreditCardID(&fixture.CreditCardID),
					fixture.WithMovementIsPaid(false),
				)
				existingMovement.CreditCardInfo.InvoiceID = &fixture.InvoiceID

				invoice := fixture.InvoiceMock(
					fixture.WithInvoiceAmount(-1000.0),
					fixture.WithInvoiceIsPaid(false),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(nil)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(existingMovement, nil)
				mockInvoiceRepo.On("FindByID", *existingMovement.CreditCardInfo.InvoiceID).Return(invoice, nil)

				// newAmount = -1000 - (-100) = -900
				newAmount := invoice.Amount - existingMovement.Amount
				updatedInvoice := invoice
				updatedInvoice.Amount = newAmount
				mockInvoiceRepo.On("UpdateAmount", mock.Anything, *invoice.ID, newAmount).Return(updatedInvoice, nil)

				creditCard := fixture.CreditCardMock()
				mockCreditCardRepo.On("UpdateLimitDelta", mock.Anything, fixture.CreditCardID, -existingMovement.Amount).Return(creditCard, nil)

				mockMovRepo.On("Delete", mock.Anything, fixture.MovementID).Return(nil)
			},
			expectedError: nil,
		},
		"should fail when movement is paid": {
			id: fixture.MovementID.String(),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository) {
				existingMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Compra no cartão de crédito"),
					fixture.AsMovementExpense(100.0),
					fixture.WithMovementTypePayment(string(domain.TypePaymentCreditCard)),
					fixture.WithMovementCreditCardID(&fixture.CreditCardID),
					fixture.WithMovementIsPaid(true),
				)
				existingMovement.CreditCardInfo.InvoiceID = &fixture.InvoiceID

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(ErrCreditMovementShouldNotBePaid)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(existingMovement, nil)
			},
			expectedError: ErrCreditMovementShouldNotBePaid,
		},
		"should fail when invoice is already paid": {
			id: fixture.MovementID.String(),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository) {
				existingMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Compra no cartão de crédito"),
					fixture.AsMovementExpense(100.0),
					fixture.WithMovementTypePayment(string(domain.TypePaymentCreditCard)),
					fixture.WithMovementCreditCardID(&fixture.CreditCardID),
					fixture.WithMovementIsPaid(false),
				)
				existingMovement.CreditCardInfo.InvoiceID = &fixture.InvoiceID

				invoice := fixture.InvoiceMock(
					fixture.WithInvoiceAmount(-1000.0),
					fixture.WithInvoiceIsPaid(true),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(ErrInvoiceAlreadyPaid)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(existingMovement, nil)
				mockInvoiceRepo.On("FindByID", *existingMovement.CreditCardInfo.InvoiceID).Return(invoice, nil)
			},
			expectedError: ErrInvoiceAlreadyPaid,
		},
		"should fail when movement is not credit card": {
			id: fixture.MovementID.String(),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository) {
				existingMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Compra com débito"),
					fixture.AsMovementExpense(100.0),
					fixture.WithMovementTypePayment(string(domain.TypePaymentDebit)),
					fixture.WithMovementIsPaid(false),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(ErrUnsupportedMovementTypeV2)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(existingMovement, nil)
			},
			expectedError: ErrUnsupportedMovementTypeV2,
		},
		"should fail when movement has no credit card info": {
			id: fixture.MovementID.String(),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository) {
				existingMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Compra sem info de cartão"),
					fixture.AsMovementExpense(100.0),
					fixture.WithMovementTypePayment(string(domain.TypePaymentCreditCard)),
					fixture.WithMovementIsPaid(false),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(ErrUnsupportedMovementTypeV2)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(existingMovement, nil)
			},
			expectedError: ErrUnsupportedMovementTypeV2,
		},
		"should fail when movement not found": {
			id: fixture.MovementID.String(),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository) {
				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(repository.ErrMovementNotFound)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(domain.Movement{}, repository.ErrMovementNotFound)
			},
			expectedError: repository.ErrMovementNotFound,
		},
		"should fail when invoice not found": {
			id: fixture.MovementID.String(),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository) {
				existingMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Compra no cartão de crédito"),
					fixture.AsMovementExpense(100.0),
					fixture.WithMovementTypePayment(string(domain.TypePaymentCreditCard)),
					fixture.WithMovementCreditCardID(&fixture.CreditCardID),
					fixture.WithMovementIsPaid(false),
				)
				existingMovement.CreditCardInfo.InvoiceID = &fixture.InvoiceID

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(repository.ErrInvoiceNotFound)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(existingMovement, nil)
				mockInvoiceRepo.On("FindByID", *existingMovement.CreditCardInfo.InvoiceID).Return(domain.Invoice{}, repository.ErrInvoiceNotFound)
			},
			expectedError: repository.ErrInvoiceNotFound,
		},
		"should fail when invoice update fails": {
			id: fixture.MovementID.String(),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository) {
				existingMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Compra no cartão de crédito"),
					fixture.AsMovementExpense(100.0),
					fixture.WithMovementTypePayment(string(domain.TypePaymentCreditCard)),
					fixture.WithMovementCreditCardID(&fixture.CreditCardID),
					fixture.WithMovementIsPaid(false),
				)
				existingMovement.CreditCardInfo.InvoiceID = &fixture.InvoiceID

				invoice := fixture.InvoiceMock(
					fixture.WithInvoiceAmount(-1000.0),
					fixture.WithInvoiceIsPaid(false),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(assert.AnError)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(existingMovement, nil)
				mockInvoiceRepo.On("FindByID", *existingMovement.CreditCardInfo.InvoiceID).Return(invoice, nil)

				newAmount := invoice.Amount - existingMovement.Amount
				mockInvoiceRepo.On("UpdateAmount", mock.Anything, *invoice.ID, newAmount).Return(domain.Invoice{}, assert.AnError)
			},
			expectedError: assert.AnError,
		},
		"should fail when movement delete fails": {
			id: fixture.MovementID.String(),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository) {
				existingMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Compra no cartão de crédito"),
					fixture.AsMovementExpense(100.0),
					fixture.WithMovementTypePayment(string(domain.TypePaymentCreditCard)),
					fixture.WithMovementCreditCardID(&fixture.CreditCardID),
					fixture.WithMovementIsPaid(false),
				)
				existingMovement.CreditCardInfo.InvoiceID = &fixture.InvoiceID

				invoice := fixture.InvoiceMock(
					fixture.WithInvoiceAmount(-1000.0),
					fixture.WithInvoiceIsPaid(false),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(assert.AnError)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(existingMovement, nil)
				mockInvoiceRepo.On("FindByID", *existingMovement.CreditCardInfo.InvoiceID).Return(invoice, nil)

				newAmount := invoice.Amount - existingMovement.Amount
				updatedInvoice := invoice
				updatedInvoice.Amount = newAmount
				mockInvoiceRepo.On("UpdateAmount", mock.Anything, *invoice.ID, newAmount).Return(updatedInvoice, nil)

				creditCard := fixture.CreditCardMock()
				mockCreditCardRepo.On("UpdateLimitDelta", mock.Anything, fixture.CreditCardID, -existingMovement.Amount).Return(creditCard, nil)

				mockMovRepo.On("Delete", mock.Anything, fixture.MovementID).Return(assert.AnError)
			},
			expectedError: assert.AnError,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockMovRepo := new(MockMovementRepository)
			mockInvoiceRepo := new(MockInvoiceRepository)
			mockTxManager := new(MockTransactionManager)

			mockCreditCardRepo := new(MockCreditCardRepository)

			if tt.mockSetup != nil {
				tt.mockSetup(mockMovRepo, mockInvoiceRepo, mockTxManager, mockCreditCardRepo)
			}

			usecase := NewMovement(
				mockMovRepo,
				new(MockRecurrentRepository),
				new(MockWalletRepository),
				new(MockSubCategory),
				mockInvoiceRepo,
				new(MockInvoice),
				mockCreditCardRepo,
				mockTxManager,
			)

			err := usecase.DeleteOne(context.Background(), fixture.MovementID)

			assert.Equal(t, tt.expectedError, err)

			mockMovRepo.AssertExpectations(t)
			mockInvoiceRepo.AssertExpectations(t)
			mockTxManager.AssertExpectations(t)
		})
	}
}
