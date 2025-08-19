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

func TestInvoice_FindOrCreateInvoiceForMovement(t *testing.T) {
	tests := map[string]struct {
		invoiceID       *uuid.UUID
		creditCardID    uuid.UUID
		movementDate    time.Time
		mockSetup       func(mockInvoiceRepo *MockInvoiceRepository, mockCreditCardRepo *MockCreditCardRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager)
		expectedInvoice domain.Invoice
		expectedError   error
	}{
		"should return existing invoice when found": {
			invoiceID:    nil,
			creditCardID: fixture.CreditCardID,
			movementDate: time.Date(2023, 10, 10, 0, 0, 0, 0, time.UTC),
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository, mockCreditCardRepo *MockCreditCardRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				existingInvoice := fixture.InvoiceMock()
				mockInvoiceRepo.On("FindByMonthAndCreditCard", mock.Anything, fixture.CreditCardID).Return(existingInvoice, nil)
			},
			expectedInvoice: fixture.InvoiceMock(),
			expectedError:   nil,
		},
		"should create new invoice when none found": {
			invoiceID:    nil,
			creditCardID: fixture.CreditCardID,
			movementDate: time.Date(2023, 10, 10, 0, 0, 0, 0, time.UTC),
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository, mockCreditCardRepo *MockCreditCardRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				creditCard := fixture.CreditCardMock()
				mockCreditCardRepo.On("FindByID", fixture.CreditCardID).Return(creditCard, nil)

				mockInvoiceRepo.On("FindByMonthAndCreditCard", mock.Anything, fixture.CreditCardID).Return(domain.Invoice{}, nil)

				newInvoice := fixture.InvoiceMock(
					fixture.WithInvoiceAmount(0),
					fixture.WithInvoiceIsPaid(false),
				)
				mockTxManager.On("WithTransaction", mock.Anything, mock.Anything).Return(nil)
				mockInvoiceRepo.On("Add", mock.Anything, mock.Anything, mock.Anything).Return(newInvoice, nil)
			},
			expectedInvoice: fixture.InvoiceMock(
				fixture.WithInvoiceAmount(0),
				fixture.WithInvoiceIsPaid(false),
			),
			expectedError: nil,
		},
		"should fail when credit card not found": {
			invoiceID:    nil,
			creditCardID: fixture.CreditCardID,
			movementDate: time.Date(2023, 10, 10, 0, 0, 0, 0, time.UTC),
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository, mockCreditCardRepo *MockCreditCardRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				mockInvoiceRepo.On("FindByMonthAndCreditCard", mock.Anything, fixture.CreditCardID).Return(domain.Invoice{}, nil)
				mockCreditCardRepo.On("FindByID", fixture.CreditCardID).Return(domain.CreditCard{}, assert.AnError)
			},
			expectedInvoice: domain.Invoice{},
			expectedError:   assert.AnError,
		},
		"should fail when FindByMonthAndCreditCard returns error": {
			invoiceID:    nil,
			creditCardID: fixture.CreditCardID,
			movementDate: time.Date(2023, 10, 10, 0, 0, 0, 0, time.UTC),
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository, mockCreditCardRepo *MockCreditCardRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				mockInvoiceRepo.On("FindByMonthAndCreditCard", mock.Anything, fixture.CreditCardID).Return(domain.Invoice{}, assert.AnError)
			},
			expectedInvoice: domain.Invoice{},
			expectedError:   assert.AnError,
		},
		"should return empty invoice when invoice ID provided but not found": {
			invoiceID:    &fixture.InvoiceID,
			creditCardID: fixture.CreditCardID,
			movementDate: time.Date(2023, 10, 10, 0, 0, 0, 0, time.UTC),
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository, mockCreditCardRepo *MockCreditCardRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				mockInvoiceRepo.On("FindByID", fixture.InvoiceID).Return(domain.Invoice{}, ErrInvoiceNotFound)
			},
			expectedInvoice: domain.Invoice{},
			expectedError:   nil,
		},
		"should fail when FindByID returns other error": {
			invoiceID:    &fixture.InvoiceID,
			creditCardID: fixture.CreditCardID,
			movementDate: time.Date(2023, 10, 10, 0, 0, 0, 0, time.UTC),
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository, mockCreditCardRepo *MockCreditCardRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				mockInvoiceRepo.On("FindByID", fixture.InvoiceID).Return(domain.Invoice{}, assert.AnError)
			},
			expectedInvoice: domain.Invoice{},
			expectedError:   assert.AnError,
		},
		"should fail when repo.Add returns error during invoice creation": {
			invoiceID:    nil,
			creditCardID: fixture.CreditCardID,
			movementDate: time.Date(2023, 10, 10, 0, 0, 0, 0, time.UTC),
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository, mockCreditCardRepo *MockCreditCardRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				mockInvoiceRepo.On("FindByMonthAndCreditCard", mock.Anything, fixture.CreditCardID).Return(domain.Invoice{}, nil)

				creditCard := fixture.CreditCardMock()
				mockCreditCardRepo.On("FindByID", fixture.CreditCardID).Return(creditCard, nil)

				mockInvoiceRepo.On("Add", mock.Anything, mock.Anything, mock.Anything).Return(domain.Invoice{}, assert.AnError)

				mockTxManager.On("WithTransaction", mock.Anything).Run(func(args mock.Arguments) {
					fn := args.Get(0).(func(*gorm.DB) error)
					fn(nil)
				}).Return(assert.AnError)
			},
			expectedInvoice: domain.Invoice{},
			expectedError:   assert.AnError,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mockInvoiceRepo := &MockInvoiceRepository{}
			mockCreditCardRepo := &MockCreditCardRepository{}
			mockWalletRepo := &MockWalletRepository{}
			mockTxManager := &MockTransactionManager{}
			mockMovementRepo := &MockMovementRepository{}

			tc.mockSetup(mockInvoiceRepo, mockCreditCardRepo, mockWalletRepo, mockTxManager)

			useCase := NewInvoice(mockInvoiceRepo, mockCreditCardRepo, mockWalletRepo, mockMovementRepo, mockTxManager)
			result, err := useCase.FindOrCreateInvoiceForMovement(context.Background(), tc.invoiceID, tc.creditCardID, tc.movementDate)

			if tc.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.Equal(t, tc.expectedError, err)
				assert.Equal(t, tc.expectedInvoice.Amount, result.Amount)
				assert.Equal(t, tc.expectedInvoice.IsPaid, result.IsPaid)
			}

			mockInvoiceRepo.AssertExpectations(t)
			mockCreditCardRepo.AssertExpectations(t)
			mockWalletRepo.AssertExpectations(t)
			mockTxManager.AssertExpectations(t)
		})
	}
}

func TestInvoice_UpdateAmount(t *testing.T) {
	tests := map[string]struct {
		invoiceID       uuid.UUID
		amount          float64
		mockSetup       func(mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager)
		expectedInvoice domain.Invoice
		expectedError   error
	}{
		"should update amount successfully": {
			invoiceID: fixture.InvoiceID,
			amount:    2000.0,
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager) {
				invoice := fixture.InvoiceMock(fixture.WithInvoiceIsPaid(false))
				mockInvoiceRepo.On("FindByID", fixture.InvoiceID).Return(invoice, nil)

				updatedInvoice := fixture.InvoiceMock(fixture.WithInvoiceAmount(2000.0))
				mockTxManager.On("WithTransaction", mock.Anything, mock.Anything).Return(nil)
				mockInvoiceRepo.On("UpdateAmount", mock.Anything, fixture.InvoiceID, 2000.0).Return(updatedInvoice, nil)
			},
			expectedInvoice: fixture.InvoiceMock(fixture.WithInvoiceAmount(2000.0)),
			expectedError:   nil,
		},
		"should fail when invoice is already paid": {
			invoiceID: fixture.InvoiceID,
			amount:    2000.0,
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager) {
				invoice := fixture.InvoiceMock(fixture.WithInvoiceIsPaid(true))
				mockInvoiceRepo.On("FindByID", fixture.InvoiceID).Return(invoice, nil)
			},
			expectedInvoice: domain.Invoice{},
			expectedError:   ErrInvoiceCannotModify,
		},
		"should fail when invoice not found": {
			invoiceID: fixture.InvoiceID,
			amount:    2000.0,
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager) {
				mockInvoiceRepo.On("FindByID", fixture.InvoiceID).Return(domain.Invoice{}, assert.AnError)
			},
			expectedInvoice: domain.Invoice{},
			expectedError:   assert.AnError,
		},
		"should fail when repo.UpdateAmount returns error": {
			invoiceID: fixture.InvoiceID,
			amount:    2000.0,
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository, mockTxManager *MockTransactionManager) {
				invoice := fixture.InvoiceMock(fixture.WithInvoiceIsPaid(false))
				mockInvoiceRepo.On("FindByID", fixture.InvoiceID).Return(invoice, nil)

				mockInvoiceRepo.On("UpdateAmount", mock.Anything, fixture.InvoiceID, 2000.0).Return(domain.Invoice{}, assert.AnError)

				mockTxManager.On("WithTransaction", mock.Anything).Run(func(args mock.Arguments) {
					fn := args.Get(0).(func(*gorm.DB) error)
					fn(nil)
				}).Return(assert.AnError)
			},
			expectedInvoice: domain.Invoice{},
			expectedError:   assert.AnError,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mockInvoiceRepo := &MockInvoiceRepository{}
			mockCreditCardRepo := &MockCreditCardRepository{}
			mockWalletRepo := &MockWalletRepository{}
			mockTxManager := &MockTransactionManager{}
			mockMovementRepo := &MockMovementRepository{}

			tc.mockSetup(mockInvoiceRepo, mockTxManager)

			useCase := NewInvoice(mockInvoiceRepo, mockCreditCardRepo, mockWalletRepo, mockMovementRepo, mockTxManager)
			result, err := useCase.UpdateAmount(context.Background(), tc.invoiceID, tc.amount)

			if tc.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.Equal(t, tc.expectedError, err)
				assert.Equal(t, tc.expectedInvoice.Amount, result.Amount)
			}

			mockInvoiceRepo.AssertExpectations(t)
			mockTxManager.AssertExpectations(t)
		})
	}
}

func TestInvoice_Pay(t *testing.T) {
	tests := map[string]struct {
		invoiceID       uuid.UUID
		walletID        uuid.UUID
		paymentDate     *time.Time
		mockSetup       func(mockInvoiceRepo *MockInvoiceRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager)
		expectedInvoice domain.Invoice
		expectedError   error
	}{
		"should pay invoice successfully": {
			invoiceID: fixture.InvoiceID,
			walletID:  fixture.DefaultWalletID,
			paymentDate: func() *time.Time {
				t := time.Date(2023, 10, 22, 0, 0, 0, 0, time.UTC)
				return &t
			}(),
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				invoice := fixture.InvoiceMock(
					fixture.WithInvoiceAmount(1500.0),
					fixture.WithInvoiceIsPaid(false),
				)
				mockInvoiceRepo.On("FindByID", fixture.InvoiceID).Return(invoice, nil)

				wallet := fixture.WalletMock(fixture.WithWalletBalance(2000.0))
				mockWalletRepo.On("FindByID", &fixture.DefaultWalletID).Return(wallet, nil)

				mockTxManager.On("WithTransaction", mock.Anything, mock.Anything).Return(nil)
				mockWalletRepo.On("UpdateAmount", mock.Anything, &fixture.DefaultWalletID, mock.AnythingOfType("float64")).Return(nil)

				paidInvoice := fixture.InvoiceMock(
					fixture.WithInvoicePayment(time.Date(2023, 10, 22, 0, 0, 0, 0, time.UTC), fixture.DefaultWalletID),
				)
				mockInvoiceRepo.On("UpdateStatus", mock.Anything, fixture.InvoiceID, true, mock.Anything, &fixture.DefaultWalletID).Return(paidInvoice, nil)
			},
			expectedInvoice: fixture.InvoiceMock(
				fixture.WithInvoicePayment(time.Date(2023, 10, 22, 0, 0, 0, 0, time.UTC), fixture.DefaultWalletID),
			),
			expectedError: nil,
		},
		"should fail when invoice already paid": {
			invoiceID:   fixture.InvoiceID,
			walletID:    fixture.DefaultWalletID,
			paymentDate: nil,
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				invoice := fixture.InvoiceMock(fixture.WithInvoiceIsPaid(true))
				mockInvoiceRepo.On("FindByID", fixture.InvoiceID).Return(invoice, nil)
			},
			expectedInvoice: domain.Invoice{},
			expectedError:   ErrInvoiceAlreadyPaid,
		},
		"should fail when insufficient balance": {
			invoiceID:   fixture.InvoiceID,
			walletID:    fixture.DefaultWalletID,
			paymentDate: nil,
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				invoice := fixture.InvoiceMock(
					fixture.WithInvoiceAmount(1500.0),
					fixture.WithInvoiceIsPaid(false),
				)
				mockInvoiceRepo.On("FindByID", fixture.InvoiceID).Return(invoice, nil)

				wallet := fixture.WalletMock(fixture.WithWalletBalance(1000.0))
				mockWalletRepo.On("FindByID", &fixture.DefaultWalletID).Return(wallet, nil)

				mockTxManager.On("WithTransaction", mock.Anything, mock.Anything).Return(ErrInsufficientBalance)
			},
			expectedInvoice: domain.Invoice{},
			expectedError:   ErrInsufficientBalance,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mockInvoiceRepo := &MockInvoiceRepository{}
			mockCreditCardRepo := &MockCreditCardRepository{}
			mockWalletRepo := &MockWalletRepository{}
			mockTxManager := &MockTransactionManager{}
			mockMovementRepo := &MockMovementRepository{}

			tc.mockSetup(mockInvoiceRepo, mockWalletRepo, mockTxManager)
			useCase := NewInvoice(mockInvoiceRepo, mockCreditCardRepo, mockWalletRepo, mockMovementRepo, mockTxManager)
			result, err := useCase.Pay(context.Background(), tc.invoiceID, tc.walletID, tc.paymentDate)

			assert.Equal(t, tc.expectedError, err)
			if tc.expectedError == nil {
				assert.Equal(t, tc.expectedInvoice.IsPaid, result.IsPaid)
			}

			mockInvoiceRepo.AssertExpectations(t)
			mockWalletRepo.AssertExpectations(t)
			mockTxManager.AssertExpectations(t)
		})
	}
}

func TestInvoice_FindByMonth(t *testing.T) {
	tests := map[string]struct {
		date             time.Time
		mockSetup        func(mockInvoiceRepo *MockInvoiceRepository)
		expectedInvoices []domain.Invoice
		expectedError    error
	}{
		"should find invoices by month successfully": {
			date: time.Date(2023, 10, 15, 0, 0, 0, 0, time.UTC),
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository) {
				invoices := []domain.Invoice{
					fixture.InvoiceMock(fixture.WithInvoiceAmount(1500.0)),
					fixture.InvoiceMock(fixture.WithInvoiceAmount(800.0)),
				}
				date := time.Date(2023, 10, 15, 0, 0, 0, 0, time.UTC)
				mockInvoiceRepo.On("FindByMonth", date).Return(invoices, nil)
			},
			expectedInvoices: []domain.Invoice{
				fixture.InvoiceMock(fixture.WithInvoiceAmount(1500.0)),
				fixture.InvoiceMock(fixture.WithInvoiceAmount(800.0)),
			},
			expectedError: nil,
		},
		"should return empty list when no invoices found": {
			date: time.Date(2023, 11, 15, 0, 0, 0, 0, time.UTC),
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository) {
				date := time.Date(2023, 11, 15, 0, 0, 0, 0, time.UTC)
				mockInvoiceRepo.On("FindByMonth", date).Return([]domain.Invoice{}, nil)
			},
			expectedInvoices: []domain.Invoice{},
			expectedError:    nil,
		},
		"should fail when repository returns error": {
			date: time.Date(2023, 10, 15, 0, 0, 0, 0, time.UTC),
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository) {
				date := time.Date(2023, 10, 15, 0, 0, 0, 0, time.UTC)
				mockInvoiceRepo.On("FindByMonth", date).Return([]domain.Invoice{}, assert.AnError)
			},
			expectedInvoices: []domain.Invoice{},
			expectedError:    assert.AnError,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mockInvoiceRepo := &MockInvoiceRepository{}
			mockCreditCardRepo := &MockCreditCardRepository{}
			mockWalletRepo := &MockWalletRepository{}
			mockTxManager := &MockTransactionManager{}
			mockMovementRepo := &MockMovementRepository{}

			tc.mockSetup(mockInvoiceRepo)
			useCase := NewInvoice(mockInvoiceRepo, mockCreditCardRepo, mockWalletRepo, mockMovementRepo, mockTxManager)
			result, err := useCase.FindByMonth(context.Background(), tc.date)

			if tc.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.Equal(t, tc.expectedError, err)
				assert.Equal(t, len(tc.expectedInvoices), len(result))
				for i, expectedInvoice := range tc.expectedInvoices {
					assert.Equal(t, expectedInvoice.Amount, result[i].Amount)
				}
			}

			mockInvoiceRepo.AssertExpectations(t)
		})
	}
}

func TestInvoice_FindByID(t *testing.T) {
	tests := map[string]struct {
		invoiceID       uuid.UUID
		mockSetup       func(mockInvoiceRepo *MockInvoiceRepository)
		expectedInvoice domain.Invoice
		expectedError   error
	}{
		"should find invoice by ID successfully": {
			invoiceID: fixture.InvoiceID,
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository) {
				invoice := fixture.InvoiceMock(
					fixture.WithInvoiceAmount(1200.0),
					fixture.WithInvoiceIsPaid(false),
				)
				mockInvoiceRepo.On("FindByID", fixture.InvoiceID).Return(invoice, nil)
			},
			expectedInvoice: fixture.InvoiceMock(
				fixture.WithInvoiceAmount(1200.0),
				fixture.WithInvoiceIsPaid(false),
			),
			expectedError: nil,
		},
		"should fail when invoice not found": {
			invoiceID: fixture.InvoiceID,
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository) {
				mockInvoiceRepo.On("FindByID", fixture.InvoiceID).Return(domain.Invoice{}, ErrInvoiceNotFound)
			},
			expectedInvoice: domain.Invoice{},
			expectedError:   ErrInvoiceNotFound,
		},
		"should fail when database connection fails": {
			invoiceID: fixture.InvoiceID,
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository) {
				mockInvoiceRepo.On("FindByID", fixture.InvoiceID).Return(domain.Invoice{}, assert.AnError)
			},
			expectedInvoice: domain.Invoice{},
			expectedError:   assert.AnError,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mockInvoiceRepo := &MockInvoiceRepository{}
			mockCreditCardRepo := &MockCreditCardRepository{}
			mockWalletRepo := &MockWalletRepository{}
			mockTxManager := &MockTransactionManager{}
			mockMovementRepo := &MockMovementRepository{}

			tc.mockSetup(mockInvoiceRepo)
			useCase := NewInvoice(mockInvoiceRepo, mockCreditCardRepo, mockWalletRepo, mockMovementRepo, mockTxManager)
			result, err := useCase.FindByID(context.Background(), tc.invoiceID)

			if tc.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.Equal(t, tc.expectedError, err)
				assert.Equal(t, tc.expectedInvoice.Amount, result.Amount)
				assert.Equal(t, tc.expectedInvoice.IsPaid, result.IsPaid)
			}

			mockInvoiceRepo.AssertExpectations(t)
		})
	}
}
