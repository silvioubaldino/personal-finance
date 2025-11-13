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
	"gorm.io/gorm"
)

func TestMovement_DeleteAllNext(t *testing.T) {
	tests := map[string]struct {
		id            string
		mockSetup     func(mockMovRepo *MockMovementRepository, mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager)
		expectedError error
	}{
		"should delete non-installment credit card movement successfully": {
			id: fixture.MovementID.String(),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager) {
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

				mockMovRepo.On("Delete", mock.Anything, fixture.MovementID).Return(nil)
			},
			expectedError: nil,
		},
		"should delete installment credit card movements successfully": {
			id: fixture.MovementID.String(),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager) {
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

				// Parcelas 2, 3, 4, 5
				installment2 := existingMovement
				installment3ID := uuid.New()
				installment3 := fixture.MovementMock(
					fixture.WithMovementID(installment3ID),
					fixture.WithMovementDescription("Compra parcelada 3/5"),
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

				installment4ID := uuid.New()
				installment4 := fixture.MovementMock(
					fixture.WithMovementID(installment4ID),
					fixture.WithMovementDescription("Compra parcelada 4/5"),
					fixture.AsMovementExpense(100.0),
					fixture.WithMovementTypePayment(string(domain.TypePaymentCreditCard)),
					fixture.WithMovementCreditCardID(&fixture.CreditCardID),
					fixture.WithMovementIsPaid(false),
					fixture.WithMovementInstallmentGroupID(&groupID),
				)
				installment4Number := 4
				installment4.CreditCardInfo.InvoiceID = &fixture.InvoiceID
				installment4.CreditCardInfo.InstallmentNumber = &installment4Number
				installment4.CreditCardInfo.TotalInstallments = &totalInstallments

				installment5ID := uuid.New()
				installment5 := fixture.MovementMock(
					fixture.WithMovementID(installment5ID),
					fixture.WithMovementDescription("Compra parcelada 5/5"),
					fixture.AsMovementExpense(100.0),
					fixture.WithMovementTypePayment(string(domain.TypePaymentCreditCard)),
					fixture.WithMovementCreditCardID(&fixture.CreditCardID),
					fixture.WithMovementIsPaid(false),
					fixture.WithMovementInstallmentGroupID(&groupID),
				)
				installment5Number := 5
				installment5.CreditCardInfo.InvoiceID = &fixture.InvoiceID
				installment5.CreditCardInfo.InstallmentNumber = &installment5Number
				installment5.CreditCardInfo.TotalInstallments = &totalInstallments

				installments := domain.MovementList{installment2, installment3, installment4, installment5}

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

				// Validação de faturas pagas e deleção das parcelas
				mockInvoiceRepo.On("FindByID", fixture.InvoiceID).Return(invoice, nil)
				mockInvoiceRepo.On("UpdateAmount", mock.Anything, fixture.InvoiceID, mock.Anything).Return(invoice, nil)
				mockMovRepo.On("Delete", mock.Anything, mock.Anything).Return(nil)
			},
			expectedError: nil,
		},
		"should return error when movement is not credit card": {
			id: fixture.MovementID.String(),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager) {
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
		"should return error when movement is paid": {
			id: fixture.MovementID.String(),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager) {
				existingMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Compra no cartão de crédito"),
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
		"should return error when any invoice is paid": {
			id: fixture.MovementID.String(),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager) {
				groupID := uuid.New()
				installmentNumber := 2
				totalInstallments := 3

				existingMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Compra parcelada 2/3"),
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
					fixture.WithMovementDescription("Compra parcelada 3/3"),
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
					fixture.WithInvoiceIsPaid(true), // Fatura paga!
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
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockMovRepo := new(MockMovementRepository)
			mockInvoiceRepo := new(MockInvoiceRepository)
			mockTxManager := new(MockTransactionManager)

			tt.mockSetup(mockMovRepo, mockInvoiceRepo, mockTxManager)

			usecase := &Movement{
				movementRepo: mockMovRepo,
				invoiceRepo:  mockInvoiceRepo,
				txManager:    mockTxManager,
			}

			id, _ := uuid.Parse(tt.id)
			err := usecase.DeleteAllNext(context.Background(), id, time.Time{})

			if tt.expectedError != nil {
				assert.ErrorIs(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			mockMovRepo.AssertExpectations(t)
			mockInvoiceRepo.AssertExpectations(t)
			mockTxManager.AssertExpectations(t)
		})
	}
}

