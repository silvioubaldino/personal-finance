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
	deleteDate := time.Date(2023, 3, 15, 0, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		id            string
		date          time.Time
		mockSetup     func(mockMovRepo *MockMovementRepository, mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository, mockRecurrentRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository)
		expectedError error
	}{
		"should delete non-installment credit card movement successfully": {
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
		"should delete installment credit card movements successfully": {
			id:   fixture.MovementID.String(),
			date: deleteDate,
			mockSetup: func(mockMovRepo *MockMovementRepository, mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository, mockRecurrentRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository) {
				groupID := uuid.New()
				installmentNumber := 2
				totalInstallments := 5

				existingMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Compra parcelada 2/5"),
					fixture.AsMovementExpense(100.0),
					fixture.WithMovementTypePayment(string(domain.TypePaymentCreditCard)),
					fixture.WithMovementCreditCardID(&fixture.CreditCardID),
					fixture.WithMovementIsPaid(false),
					fixture.WithMovementInstallmentGroupID(&groupID),
				)
				existingMovement.CreditCardInfo.InvoiceID = &fixture.InvoiceID
				existingMovement.CreditCardInfo.InstallmentNumber = &installmentNumber
				existingMovement.CreditCardInfo.TotalInstallments = &totalInstallments

				installment2 := existingMovement
				installment3ID := uuid.New()
				installment3 := fixture.MovementMock(
					fixture.WithMovementID(installment3ID),
					fixture.AsMovementExpense(100.0),
					fixture.WithMovementTypePayment(string(domain.TypePaymentCreditCard)),
					fixture.WithMovementCreditCardID(&fixture.CreditCardID),
					fixture.WithMovementIsPaid(false),
					fixture.WithMovementInstallmentGroupID(&groupID),
				)
				installment3Number := 3
				installment3.CreditCardInfo.InvoiceID = &fixture.InvoiceID
				installment3.CreditCardInfo.InstallmentNumber = &installment3Number
				installment3.CreditCardInfo.TotalInstallments = &totalInstallments

				installments := domain.MovementList{installment2, installment3}

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
				mockMovRepo.On("FindByInstallmentGroupFromNumber", groupID, installmentNumber).Return(installments, nil)

				mockInvoiceRepo.On("FindByID", fixture.InvoiceID).Return(invoice, nil)
				mockInvoiceRepo.On("UpdateAmount", mock.Anything, fixture.InvoiceID, mock.Anything).Return(invoice, nil)

				creditCard := fixture.CreditCardMock()
				mockCreditCardRepo.On("UpdateLimitDelta", mock.Anything, fixture.CreditCardID, mock.Anything).Return(creditCard, nil)

				mockMovRepo.On("Delete", mock.Anything, mock.Anything).Return(nil)
			},
			expectedError: nil,
		},
		"should delete regular unpaid movement": {
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
		"should truncate recurrent chain and delete movement": {
			id:   fixture.MovementID.String(),
			date: deleteDate,
			mockSetup: func(mockMovRepo *MockMovementRepository, mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository, mockRecurrentRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository) {
				existingMovement := fixture.MovementMock(
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

				// Truncate: update recurrent to end at February 2023 (month before deleted March)
				mockRecurrentRepo.On("Update", mock.Anything, recurrent.ID, mock.MatchedBy(func(r domain.RecurrentMovement) bool {
					return r.EndDate != nil && r.EndDate.Month() == time.February && r.EndDate.Year() == 2023
				})).Return(recurrent, nil)

				mockMovRepo.On("Delete", mock.Anything, fixture.MovementID).Return(nil)
			},
			expectedError: nil,
		},
		"should delete single-month recurrent entirely": {
			id:   fixture.MovementID.String(),
			date: time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository, mockRecurrentRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository) {
				existingMovement := fixture.MovementMock(
					fixture.AsMovementExpense(1500.0),
					fixture.WithMovementTypePayment(string(domain.TypePaymentDebit)),
					fixture.WithMovementIsPaid(false),
					fixture.AsMovementRecurrent(),
					fixture.WithMovementRecurrentID(),
				)

				// Recurrent that starts and ends in January 2023
				recurrent := fixture.RecurrentMovementMock(
					fixture.WithRecurrentMovementInitialDate(time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC)),
					fixture.WithRecurrentMovementEndDate(time.Date(2023, 1, 31, 10, 0, 0, 0, time.UTC)),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(nil)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(existingMovement, nil)
				mockRecurrentRepo.On("FindByID", fixture.RecurrentID).Return(recurrent, nil)

				// Single month: delete physical movement, clean up all recurrent movements, delete recurrent
				mockMovRepo.On("Delete", mock.Anything, fixture.MovementID).Return(nil)
				mockMovRepo.On("DeleteAllByRecurrentID", mock.Anything, *recurrent.ID).Return(nil)
				mockRecurrentRepo.On("Delete", mock.Anything, recurrent.ID).Return(nil)
			},
			expectedError: nil,
		},
		"should truncate virtual recurrent by recurrent ID": {
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
			},
			expectedError: nil,
		},
		"should fail when credit card movement is paid": {
			id:   fixture.MovementID.String(),
			date: deleteDate,
			mockSetup: func(mockMovRepo *MockMovementRepository, mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository, mockRecurrentRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository) {
				existingMovement := fixture.MovementMock(
					fixture.AsMovementExpense(100.0),
					fixture.WithMovementTypePayment(string(domain.TypePaymentCreditCard)),
					fixture.WithMovementCreditCardID(&fixture.CreditCardID),
					fixture.WithMovementIsPaid(true),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(ErrCreditMovementShouldNotBePaid)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(existingMovement, nil)
			},
			expectedError: ErrCreditMovementShouldNotBePaid,
		},
		"should fail when any invoice is paid": {
			id:   fixture.MovementID.String(),
			date: deleteDate,
			mockSetup: func(mockMovRepo *MockMovementRepository, mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository, mockRecurrentRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository) {
				groupID := uuid.New()
				installmentNumber := 2
				totalInstallments := 3

				existingMovement := fixture.MovementMock(
					fixture.AsMovementExpense(100.0),
					fixture.WithMovementTypePayment(string(domain.TypePaymentCreditCard)),
					fixture.WithMovementCreditCardID(&fixture.CreditCardID),
					fixture.WithMovementIsPaid(false),
					fixture.WithMovementInstallmentGroupID(&groupID),
				)
				existingMovement.CreditCardInfo.InvoiceID = &fixture.InvoiceID
				existingMovement.CreditCardInfo.InstallmentNumber = &installmentNumber
				existingMovement.CreditCardInfo.TotalInstallments = &totalInstallments

				installment2 := existingMovement
				installment3ID := uuid.New()
				installment3InvoiceID := uuid.New()
				installment3 := fixture.MovementMock(
					fixture.WithMovementID(installment3ID),
					fixture.AsMovementExpense(100.0),
					fixture.WithMovementTypePayment(string(domain.TypePaymentCreditCard)),
					fixture.WithMovementCreditCardID(&fixture.CreditCardID),
					fixture.WithMovementIsPaid(false),
					fixture.WithMovementInstallmentGroupID(&groupID),
				)
				installment3Number := 3
				installment3.CreditCardInfo.InvoiceID = &installment3InvoiceID
				installment3.CreditCardInfo.InstallmentNumber = &installment3Number
				installment3.CreditCardInfo.TotalInstallments = &totalInstallments

				installments := domain.MovementList{installment2, installment3}

				invoice1 := fixture.InvoiceMock(
					fixture.WithInvoiceAmount(-1000.0),
					fixture.WithInvoiceIsPaid(false),
				)

				invoice2 := fixture.InvoiceMock(
					fixture.WithID(installment3InvoiceID),
					fixture.WithInvoiceAmount(-1000.0),
					fixture.WithInvoiceIsPaid(true),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(ErrInvoiceAlreadyPaid)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(existingMovement, nil)
				mockMovRepo.On("FindByInstallmentGroupFromNumber", groupID, installmentNumber).Return(installments, nil)

				mockInvoiceRepo.On("FindByID", fixture.InvoiceID).Return(invoice1, nil).Once()
				mockInvoiceRepo.On("FindByID", installment3InvoiceID).Return(invoice2, nil).Once()
			},
			expectedError: ErrInvoiceAlreadyPaid,
		},
		"should fail when virtual recurrent has no date": {
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

			tt.mockSetup(mockMovRepo, mockInvoiceRepo, mockTxManager, mockCreditCardRepo, mockRecurrentRepo, mockWalletRepo)

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
			err := uc.DeleteAllNext(context.Background(), id, tt.date)

			assert.Equal(t, tt.expectedError, err)

			mockMovRepo.AssertExpectations(t)
			mockInvoiceRepo.AssertExpectations(t)
			mockTxManager.AssertExpectations(t)
			mockRecurrentRepo.AssertExpectations(t)
		})
	}
}
