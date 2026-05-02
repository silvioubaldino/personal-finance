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

func TestMovement_DeleteOne(t *testing.T) {
	deleteDate := time.Date(2023, 3, 15, 0, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		id            string
		date          time.Time
		mockSetup     func(mockMovRepo *MockMovementRepository, mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository, mockRecurrentRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository)
		expectedError error
	}{
		"should delete credit card movement with success": {
			id:   fixture.MovementID.String(),
			date: deleteDate,
			mockSetup: func(mockMovRepo *MockMovementRepository, mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository, mockRecurrentRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository) {
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
		"should delete regular unpaid movement with success": {
			id:   fixture.MovementID.String(),
			date: deleteDate,
			mockSetup: func(mockMovRepo *MockMovementRepository, mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository, mockRecurrentRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository) {
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
					}).Return(nil)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(existingMovement, nil)
				mockMovRepo.On("Delete", mock.Anything, fixture.MovementID).Return(nil)
			},
			expectedError: nil,
		},
		"should delete regular paid movement and revert wallet": {
			id:   fixture.MovementID.String(),
			date: deleteDate,
			mockSetup: func(mockMovRepo *MockMovementRepository, mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository, mockRecurrentRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository) {
				existingMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Pagamento com débito"),
					fixture.AsMovementExpense(100.0),
					fixture.WithMovementTypePayment(string(domain.TypePaymentDebit)),
					fixture.WithMovementIsPaid(true),
				)

				wallet := fixture.WalletMock(
					fixture.WithWalletID(fixture.WalletID),
					fixture.WithWalletBalance(900.0),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(nil)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(existingMovement, nil)
				mockWalletRepo.On("FindByID", existingMovement.WalletID).Return(wallet, nil)
				mockWalletRepo.On("UpdateAmount", mock.Anything, wallet.ID, wallet.Balance+existingMovement.ReverseAmount()).Return(nil)
				mockMovRepo.On("Delete", mock.Anything, fixture.MovementID).Return(nil)
			},
			expectedError: nil,
		},
		"should delete recurrent movement and split chain": {
			id:   fixture.MovementID.String(),
			date: deleteDate,
			mockSetup: func(mockMovRepo *MockMovementRepository, mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository, mockRecurrentRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository) {
				existingMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Aluguel"),
					fixture.AsMovementExpense(1500.0),
					fixture.WithMovementTypePayment(string(domain.TypePaymentDebit)),
					fixture.WithMovementIsPaid(false),
					fixture.AsMovementRecurrent(),
					fixture.WithMovementRecurrentID(),
				)

				recurrent := fixture.RecurrentMovementMock(
					fixture.WithRecurrentMovementInitialDate(time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC)),
					fixture.WithRecurrentMovementEndDate(time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(nil)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(existingMovement, nil)
				mockRecurrentRepo.On("FindByID", fixture.RecurrentID).Return(recurrent, nil)

				// Split: update old recurrent to end at February 2023 (month before deleted March)
				mockRecurrentRepo.On("Update", mock.Anything, recurrent.ID, mock.MatchedBy(func(r domain.RecurrentMovement) bool {
					return r.EndDate != nil && r.EndDate.Month() == time.February && r.EndDate.Year() == 2023
				})).Return(recurrent, nil)

				// Create continuation starting April 2023 (month after deleted March)
				mockRecurrentRepo.On("Add", mock.Anything, mock.MatchedBy(func(r domain.RecurrentMovement) bool {
					return r.ID == nil && r.InitialDate != nil && r.InitialDate.Month() == time.April && r.InitialDate.Year() == 2023
				})).Return(domain.RecurrentMovement{}, nil)

				mockMovRepo.On("Delete", mock.Anything, fixture.MovementID).Return(nil)
			},
			expectedError: nil,
		},
		"should delete paid recurrent movement, split chain and revert wallet": {
			id:   fixture.MovementID.String(),
			date: deleteDate,
			mockSetup: func(mockMovRepo *MockMovementRepository, mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository, mockRecurrentRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository) {
				existingMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Aluguel pago"),
					fixture.AsMovementExpense(1500.0),
					fixture.WithMovementTypePayment(string(domain.TypePaymentDebit)),
					fixture.WithMovementIsPaid(true),
					fixture.AsMovementRecurrent(),
					fixture.WithMovementRecurrentID(),
				)

				recurrent := fixture.RecurrentMovementMock(
					fixture.WithRecurrentMovementInitialDate(time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC)),
					fixture.WithRecurrentMovementEndDate(time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)),
				)

				wallet := fixture.WalletMock(
					fixture.WithWalletID(fixture.WalletID),
					fixture.WithWalletBalance(500.0),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(nil)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(existingMovement, nil)
				mockRecurrentRepo.On("FindByID", fixture.RecurrentID).Return(recurrent, nil)
				mockRecurrentRepo.On("Update", mock.Anything, recurrent.ID, mock.Anything).Return(recurrent, nil)
				mockRecurrentRepo.On("Add", mock.Anything, mock.Anything).Return(domain.RecurrentMovement{}, nil)
				mockWalletRepo.On("FindByID", existingMovement.WalletID).Return(wallet, nil)
				mockWalletRepo.On("UpdateAmount", mock.Anything, wallet.ID, wallet.Balance+existingMovement.ReverseAmount()).Return(nil)
				mockMovRepo.On("Delete", mock.Anything, fixture.MovementID).Return(nil)
			},
			expectedError: nil,
		},
		"should delete virtual recurrent (no physical movement) by recurrent ID": {
			id:   fixture.RecurrentMovementID.String(),
			date: deleteDate,
			mockSetup: func(mockMovRepo *MockMovementRepository, mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository, mockRecurrentRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository) {
				recurrent := fixture.RecurrentMovementMock(
					fixture.WithRecurrentMovementInitialDate(time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC)),
					fixture.WithRecurrentMovementEndDate(time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(nil)

				mockMovRepo.On("FindByID", fixture.RecurrentMovementID).Return(domain.Movement{}, repository.ErrMovementNotFound)
				mockRecurrentRepo.On("FindByID", fixture.RecurrentMovementID).Return(recurrent, nil)
				mockRecurrentRepo.On("Update", mock.Anything, recurrent.ID, mock.Anything).Return(recurrent, nil)
				mockRecurrentRepo.On("Add", mock.Anything, mock.Anything).Return(domain.RecurrentMovement{}, nil)
			},
			expectedError: nil,
		},
		"should fail when credit card movement is paid": {
			id:   fixture.MovementID.String(),
			date: deleteDate,
			mockSetup: func(mockMovRepo *MockMovementRepository, mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository, mockRecurrentRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository) {
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
			id:   fixture.MovementID.String(),
			date: deleteDate,
			mockSetup: func(mockMovRepo *MockMovementRepository, mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository, mockRecurrentRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository) {
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
		"should fail when movement not found and recurrent not found": {
			id:   fixture.MovementID.String(),
			date: deleteDate,
			mockSetup: func(mockMovRepo *MockMovementRepository, mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository, mockRecurrentRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository) {
				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(repository.ErrRecurrentMovementNotFound)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(domain.Movement{}, repository.ErrMovementNotFound)
				mockRecurrentRepo.On("FindByID", fixture.MovementID).Return(domain.RecurrentMovement{}, repository.ErrRecurrentMovementNotFound)
			},
			expectedError: repository.ErrRecurrentMovementNotFound,
		},
		"should fail when virtual recurrent delete has no date": {
			id: fixture.RecurrentMovementID.String(),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository, mockRecurrentRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository) {
				recurrent := fixture.RecurrentMovementMock()

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(ErrDateRequired)

				mockMovRepo.On("FindByID", fixture.RecurrentMovementID).Return(domain.Movement{}, repository.ErrMovementNotFound)
				mockRecurrentRepo.On("FindByID", fixture.RecurrentMovementID).Return(recurrent, nil)
			},
			expectedError: ErrDateRequired,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockMovRepo := new(MockMovementRepository)
			mockInvoiceRepo := new(MockInvoiceRepository)
			mockTxManager := new(MockTransactionManager)
			mockCreditCardRepo := new(MockCreditCardRepository)
			mockRecurrentRepo := new(MockRecurrentRepository)
			mockWalletRepo := new(MockWalletRepository)

			if tt.mockSetup != nil {
				tt.mockSetup(mockMovRepo, mockInvoiceRepo, mockTxManager, mockCreditCardRepo, mockRecurrentRepo, mockWalletRepo)
			}

			uc := NewMovement(
				mockMovRepo,
				mockRecurrentRepo,
				mockWalletRepo,
				new(MockSubCategory),
				mockInvoiceRepo,
				new(MockInvoice),
				mockCreditCardRepo,
				mockTxManager,
				nil,
			)

			id := uuid.MustParse(tt.id)
			err := uc.DeleteOne(context.Background(), id, tt.date)

			assert.Equal(t, tt.expectedError, err)

			mockMovRepo.AssertExpectations(t)
			mockInvoiceRepo.AssertExpectations(t)
			mockTxManager.AssertExpectations(t)
			mockRecurrentRepo.AssertExpectations(t)
		})
	}
}
