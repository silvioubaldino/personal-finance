package usecase

import (
	"context"
	"testing"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/domain/fixture"
	"personal-finance/internal/infrastructure/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

func TestMovement_DeleteAllNext(t *testing.T) {
	targetDate := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		id            string
		date          *time.Time
		mockSetup     func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository)
		expectedError error
	}{
		"(a) should delete non-recurrent unpaid movement with success": {
			id:   fixture.MovementID.String(),
			date: nil,
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository) {
				existingMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Compra avulsa"),
					fixture.AsMovementExpense(100.0),
					fixture.WithMovementIsPaid(false),
					fixture.WithMovementIsRecurrent(false),
				)
				existingMovement.RecurrentID = nil

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(nil)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(existingMovement, nil)
				mockMovRepo.On("Delete", mock.Anything, fixture.MovementID).Return(nil)
			},
			expectedError: nil,
		},
		"(a) should delete non-recurrent paid movement and revert wallet": {
			id:   fixture.MovementID.String(),
			date: nil,
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository) {
				existingMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Compra paga"),
					fixture.AsMovementExpense(100.0),
					fixture.WithMovementIsPaid(true),
					fixture.WithMovementIsRecurrent(false),
				)
				existingMovement.RecurrentID = nil

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(nil)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(existingMovement, nil)

				mockWalletRepo.On("FindByID", existingMovement.WalletID).Return(domain.Wallet{
					ID:      existingMovement.WalletID,
					Balance: 1000.0,
				}, nil)
				mockWalletRepo.On("UpdateAmount", mock.Anything, existingMovement.WalletID, 1100.0).Return(nil)

				mockMovRepo.On("Delete", mock.Anything, fixture.MovementID).Return(nil)
			},
			expectedError: nil,
		},
		"(b) should delete recurrent movement and end recurrence (no new recurrence)": {
			id:   fixture.MovementID.String(),
			date: nil,
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository) {
				existingMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Assinatura mensal"),
					fixture.AsMovementExpense(100.0),
					fixture.WithMovementIsPaid(false),
					fixture.WithMovementIsRecurrent(true),
					fixture.WithMovementRecurrentID(),
				)

				recurrent := fixture.RecurrentMovementMock()

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(nil)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(existingMovement, nil)
				mockRecRepo.On("FindByID", fixture.RecurrentMovementID).Return(recurrent, nil)
				mockRecRepo.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(recurrent, nil)
				mockMovRepo.On("Delete", mock.Anything, fixture.MovementID).Return(nil)
			},
			expectedError: nil,
		},
		"(b) should delete recurrent paid movement and revert wallet": {
			id:   fixture.MovementID.String(),
			date: nil,
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository) {
				existingMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Assinatura mensal paga"),
					fixture.AsMovementExpense(100.0),
					fixture.WithMovementIsPaid(true),
					fixture.WithMovementIsRecurrent(true),
					fixture.WithMovementRecurrentID(),
				)

				recurrent := fixture.RecurrentMovementMock()

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(nil)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(existingMovement, nil)
				mockRecRepo.On("FindByID", fixture.RecurrentMovementID).Return(recurrent, nil)
				mockRecRepo.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(recurrent, nil)

				mockWalletRepo.On("FindByID", existingMovement.WalletID).Return(domain.Wallet{
					ID:      existingMovement.WalletID,
					Balance: 1000.0,
				}, nil)
				mockWalletRepo.On("UpdateAmount", mock.Anything, existingMovement.WalletID, 1100.0).Return(nil)

				mockMovRepo.On("Delete", mock.Anything, fixture.MovementID).Return(nil)
			},
			expectedError: nil,
		},
		"(c) should end recurrence for virtual movement with date parameter": {
			id:   fixture.RecurrentMovementID.String(),
			date: &targetDate,
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository) {
				recurrent := fixture.RecurrentMovementMock()

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(nil)

				mockMovRepo.On("FindByID", fixture.RecurrentMovementID).Return(domain.Movement{}, repository.ErrMovementNotFound)
				mockRecRepo.On("FindByID", fixture.RecurrentMovementID).Return(recurrent, nil)
				mockRecRepo.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(recurrent, nil)
			},
			expectedError: nil,
		},
		"(c) should fail when deleting virtual movement without date": {
			id:   fixture.RecurrentMovementID.String(),
			date: nil,
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository) {
				recurrent := fixture.RecurrentMovementMock()

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(ErrDateRequired)

				mockMovRepo.On("FindByID", fixture.RecurrentMovementID).Return(domain.Movement{}, repository.ErrMovementNotFound)
				mockRecRepo.On("FindByID", fixture.RecurrentMovementID).Return(recurrent, nil)
			},
			expectedError: ErrDateRequired,
		},
		"(d) should delete credit card single movement with success": {
			id:   fixture.MovementID.String(),
			date: nil,
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository) {
				existingMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Compra no cartão de crédito"),
					fixture.AsMovementExpense(100.0),
					fixture.WithMovementTypePayment(string(domain.TypePaymentCreditCard)),
					fixture.WithMovementCreditCardID(&fixture.CreditCardID),
					fixture.WithMovementIsPaid(false),
				)
				existingMovement.CreditCardInfo.InvoiceID = &fixture.InvoiceID
				existingMovement.CreditCardInfo.InstallmentGroupID = nil
				existingMovement.CreditCardInfo.InstallmentNumber = nil

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
		// TODO: Test case for installments requires more complex mock setup
		// "d) should delete credit card installments from current onwards" - skipped for now
		"(d) should fail when credit card movement is paid": {
			id:   fixture.MovementID.String(),
			date: nil,
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository) {
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
		"(d) should fail when invoice is already paid": {
			id:   fixture.MovementID.String(),
			date: nil,
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository) {
				existingMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Compra no cartão de crédito"),
					fixture.AsMovementExpense(100.0),
					fixture.WithMovementTypePayment(string(domain.TypePaymentCreditCard)),
					fixture.WithMovementCreditCardID(&fixture.CreditCardID),
					fixture.WithMovementIsPaid(false),
				)
				existingMovement.CreditCardInfo.InvoiceID = &fixture.InvoiceID
				existingMovement.CreditCardInfo.InstallmentGroupID = nil
				existingMovement.CreditCardInfo.InstallmentNumber = nil

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
		// TODO: Test case for installments with paid invoice requires more complex mock setup
		// "d) should fail when any installment invoice is paid" - skipped for now
		"should fail when movement and recurrent not found": {
			id:   fixture.MovementID.String(),
			date: &targetDate,
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository) {
				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(repository.ErrRecurrentMovementNotFound)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(domain.Movement{}, repository.ErrMovementNotFound)
				mockRecRepo.On("FindByID", fixture.MovementID).Return(domain.RecurrentMovement{}, repository.ErrRecurrentMovementNotFound)
			},
			expectedError: repository.ErrRecurrentMovementNotFound,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockMovRepo := new(MockMovementRepository)
			mockRecRepo := new(MockRecurrentRepository)
			mockWalletRepo := new(MockWalletRepository)
			mockInvoiceRepo := new(MockInvoiceRepository)
			mockTxManager := new(MockTransactionManager)
			mockCreditCardRepo := new(MockCreditCardRepository)

			if tt.mockSetup != nil {
				tt.mockSetup(mockMovRepo, mockRecRepo, mockWalletRepo, mockInvoiceRepo, mockTxManager, mockCreditCardRepo)
			}

			usecase := NewMovement(
				mockMovRepo,
				mockRecRepo,
				mockWalletRepo,
				new(MockSubCategory),
				mockInvoiceRepo,
				new(MockInvoice),
				mockCreditCardRepo,
				mockTxManager,
			)

			id, _ := uuid.Parse(tt.id)
			err := usecase.DeleteAllNext(context.Background(), id, tt.date)

			assert.Equal(t, tt.expectedError, err)

			mockMovRepo.AssertExpectations(t)
			mockRecRepo.AssertExpectations(t)
			mockWalletRepo.AssertExpectations(t)
			mockInvoiceRepo.AssertExpectations(t)
			mockTxManager.AssertExpectations(t)
		})
	}
}
