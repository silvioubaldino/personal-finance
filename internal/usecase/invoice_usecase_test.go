package usecase

import (
	"context"
	"errors"
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
			expectedError:   ErrInvoiceNotFound,
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
			result, err := useCase.FindOrCreateInvoiceForMovement(context.Background(), tc.invoiceID, &tc.creditCardID, tc.movementDate)

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

				// new amount = current (-1500) + delta (2000) = 500
				updatedInvoice := fixture.InvoiceMock(fixture.WithInvoiceAmount(500.0))
				mockTxManager.On("WithTransaction", mock.Anything).Return(nil)
				mockInvoiceRepo.On("UpdateAmount", mock.Anything, fixture.InvoiceID, 500.0).Return(updatedInvoice, nil)
			},
			expectedInvoice: fixture.InvoiceMock(fixture.WithInvoiceAmount(500.0)),
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

				// new amount = current (-1500) + delta (2000) = 500
				mockInvoiceRepo.On("UpdateAmount", mock.Anything, fixture.InvoiceID, 500.0).Return(domain.Invoice{}, assert.AnError)

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
		mockSetup       func(mockInvoiceRepo *MockInvoiceRepository, mockWalletRepo *MockWalletRepository, mockMovementRepo *MockMovementRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository)
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
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository, mockWalletRepo *MockWalletRepository, mockMovementRepo *MockMovementRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository) {
				invoice := fixture.InvoiceMock(
					fixture.WithInvoiceIsPaid(false),
				)
				mockInvoiceRepo.On("FindByID", fixture.InvoiceID).Return(invoice, nil)

				wallet := fixture.WalletMock(fixture.WithWalletBalance(2000.0))
				mockWalletRepo.On("FindByID", &fixture.DefaultWalletID).Return(wallet, nil)

				paidInvoice := fixture.InvoiceMock(
					fixture.WithInvoicePayment(time.Date(2023, 10, 22, 0, 0, 0, 0, time.UTC), fixture.DefaultWalletID),
				)

				mockTxManager.On("WithTransaction", mock.Anything).Run(func(args mock.Arguments) {
					fn := args.Get(0).(func(*gorm.DB) error)

					mockWalletRepo.On("UpdateAmount", mock.Anything, &fixture.DefaultWalletID, mock.AnythingOfType("float64")).Return(nil)
					mockInvoiceRepo.On("UpdateStatus", mock.Anything, mock.Anything, fixture.InvoiceID, true, mock.Anything, &fixture.DefaultWalletID).Return(paidInvoice, nil)

					movement := fixture.MovementMock(
						fixture.WithMovementAmount(-1500.0),
						fixture.WithMovementDate(time.Date(2023, 10, 22, 0, 0, 0, 0, time.UTC)),
						fixture.WithMovementType(domain.TypePaymentInvoicePayment),
						fixture.WithMovementDescription("Pagamento da fatura "),
					)
					mockMovementRepo.On("Add", mock.Anything, mock.Anything, mock.Anything).Return(movement, nil)

					creditCard := fixture.CreditCardMock()
					mockCreditCardRepo.On("UpdateLimitDelta", mock.Anything, fixture.CreditCardID, -invoice.Amount).Return(creditCard, nil)

					mockMovementRepo.On("PayByInvoiceID", mock.Anything, fixture.InvoiceID).Return(nil)

					fn(nil)
				}).Return(nil)
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
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository, mockWalletRepo *MockWalletRepository, mockMovementRepo *MockMovementRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository) {
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
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository, mockWalletRepo *MockWalletRepository, mockMovementRepo *MockMovementRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository) {
				invoice := fixture.InvoiceMock(
					fixture.WithInvoiceAmount(-1500.0),
					fixture.WithInvoiceIsPaid(false),
				)
				mockInvoiceRepo.On("FindByID", fixture.InvoiceID).Return(invoice, nil)

				wallet := fixture.WalletMock(fixture.WithWalletBalance(1000.0))
				mockWalletRepo.On("FindByID", &fixture.DefaultWalletID).Return(wallet, nil)
			},
			expectedInvoice: domain.Invoice{},
			expectedError:   domain.ErrWalletInsufficient,
		},
		"should fail when invoice repository FindByID fails": {
			invoiceID:   fixture.InvoiceID,
			walletID:    fixture.DefaultWalletID,
			paymentDate: nil,
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository, mockWalletRepo *MockWalletRepository, mockMovementRepo *MockMovementRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository) {
				mockInvoiceRepo.On("FindByID", fixture.InvoiceID).Return(domain.Invoice{}, errors.New("database connection failed"))
			},
			expectedInvoice: domain.Invoice{},
			expectedError:   errors.New("error finding invoice: database connection failed"),
		},
		"should fail when wallet repository FindByID fails": {
			invoiceID:   fixture.InvoiceID,
			walletID:    fixture.DefaultWalletID,
			paymentDate: nil,
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository, mockWalletRepo *MockWalletRepository, mockMovementRepo *MockMovementRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository) {
				invoice := fixture.InvoiceMock(
					fixture.WithInvoiceIsPaid(false),
				)
				mockInvoiceRepo.On("FindByID", fixture.InvoiceID).Return(invoice, nil)

				mockWalletRepo.On("FindByID", &fixture.DefaultWalletID).Return(domain.Wallet{}, errors.New("wallet not found in database"))
			},
			expectedInvoice: domain.Invoice{},
			expectedError:   errors.New("error finding wallet: wallet not found in database"),
		},
		"should fail when wallet UpdateAmount fails during transaction": {
			invoiceID:   fixture.InvoiceID,
			walletID:    fixture.DefaultWalletID,
			paymentDate: nil,
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository, mockWalletRepo *MockWalletRepository, mockMovementRepo *MockMovementRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository) {
				invoice := fixture.InvoiceMock(
					fixture.WithInvoiceIsPaid(false),
				)
				mockInvoiceRepo.On("FindByID", fixture.InvoiceID).Return(invoice, nil)

				wallet := fixture.WalletMock(fixture.WithWalletBalance(2000.0))
				mockWalletRepo.On("FindByID", &fixture.DefaultWalletID).Return(wallet, nil)

				mockTxManager.On("WithTransaction", mock.Anything).Run(func(args mock.Arguments) {
					fn := args.Get(0).(func(*gorm.DB) error)
					mockWalletRepo.On("UpdateAmount", mock.Anything, &fixture.DefaultWalletID, mock.AnythingOfType("float64")).Return(errors.New("wallet update constraint violation"))
					fn(nil)
				}).Return(errors.New("error updating wallet balance: wallet update constraint violation"))
			},
			expectedInvoice: domain.Invoice{},
			expectedError:   errors.New("error updating wallet balance: wallet update constraint violation"),
		},
		"should fail when invoice UpdateStatus fails during transaction": {
			invoiceID:   fixture.InvoiceID,
			walletID:    fixture.DefaultWalletID,
			paymentDate: nil,
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository, mockWalletRepo *MockWalletRepository, mockMovementRepo *MockMovementRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository) {
				invoice := fixture.InvoiceMock(
					fixture.WithInvoiceIsPaid(false),
				)
				mockInvoiceRepo.On("FindByID", fixture.InvoiceID).Return(invoice, nil)

				wallet := fixture.WalletMock(fixture.WithWalletBalance(2000.0))
				mockWalletRepo.On("FindByID", &fixture.DefaultWalletID).Return(wallet, nil)

				mockTxManager.On("WithTransaction", mock.Anything).Run(func(args mock.Arguments) {
					fn := args.Get(0).(func(*gorm.DB) error)
					mockWalletRepo.On("UpdateAmount", mock.Anything, &fixture.DefaultWalletID, mock.AnythingOfType("float64")).Return(nil)
					mockInvoiceRepo.On("UpdateStatus", mock.Anything, mock.Anything, fixture.InvoiceID, true, mock.Anything, &fixture.DefaultWalletID).Return(domain.Invoice{}, errors.New("invoice status update failed"))
					fn(nil)
				}).Return(errors.New("error marking invoice as paid: invoice status update failed"))
			},
			expectedInvoice: domain.Invoice{},
			expectedError:   errors.New("error marking invoice as paid: invoice status update failed"),
		},
		"should fail when movement Add fails during transaction": {
			invoiceID:   fixture.InvoiceID,
			walletID:    fixture.DefaultWalletID,
			paymentDate: nil,
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository, mockWalletRepo *MockWalletRepository, mockMovementRepo *MockMovementRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository) {
				invoice := fixture.InvoiceMock(
					fixture.WithInvoiceIsPaid(false),
				)
				mockInvoiceRepo.On("FindByID", fixture.InvoiceID).Return(invoice, nil)

				wallet := fixture.WalletMock(fixture.WithWalletBalance(2000.0))
				mockWalletRepo.On("FindByID", &fixture.DefaultWalletID).Return(wallet, nil)

				paidInvoice := fixture.InvoiceMock(
					fixture.WithInvoicePayment(time.Now(), fixture.DefaultWalletID),
				)

				mockTxManager.On("WithTransaction", mock.Anything).Run(func(args mock.Arguments) {
					fn := args.Get(0).(func(*gorm.DB) error)
					mockWalletRepo.On("UpdateAmount", mock.Anything, &fixture.DefaultWalletID, mock.AnythingOfType("float64")).Return(nil)
					mockInvoiceRepo.On("UpdateStatus", mock.Anything, mock.Anything, fixture.InvoiceID, true, mock.Anything, &fixture.DefaultWalletID).Return(paidInvoice, nil)
					mockMovementRepo.On("Add", mock.Anything, mock.Anything, mock.Anything).Return(domain.Movement{}, errors.New("movement constraint violation"))
					fn(nil)
				}).Return(errors.New("error creating movement: movement constraint violation"))
			},
			expectedInvoice: domain.Invoice{},
			expectedError:   errors.New("error creating movement: movement constraint violation"),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mockInvoiceRepo := &MockInvoiceRepository{}
			mockCreditCardRepo := &MockCreditCardRepository{}
			mockWalletRepo := &MockWalletRepository{}
			mockTxManager := &MockTransactionManager{}
			mockMovementRepo := &MockMovementRepository{}

			tc.mockSetup(mockInvoiceRepo, mockWalletRepo, mockMovementRepo, mockTxManager, mockCreditCardRepo)
			useCase := NewInvoice(mockInvoiceRepo, mockCreditCardRepo, mockWalletRepo, mockMovementRepo, mockTxManager)
			result, err := useCase.Pay(context.Background(), tc.invoiceID, tc.walletID, tc.paymentDate, nil)

			if tc.expectedError != nil {
				assert.Error(t, err)
				if tc.expectedError == domain.ErrWalletInsufficient {
					assert.Contains(t, err.Error(), tc.expectedError.Error())
				} else {
					assert.Contains(t, err.Error(), tc.expectedError.Error())
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedInvoice.IsPaid, result.IsPaid)
			}

			mockInvoiceRepo.AssertExpectations(t)
			mockWalletRepo.AssertExpectations(t)
			mockTxManager.AssertExpectations(t)
		})
	}
}

func TestInvoice_PayPartial(t *testing.T) {
	tests := map[string]struct {
		invoiceID       uuid.UUID
		walletID        uuid.UUID
		paymentDate     *time.Time
		amount          *float64
		mockSetup       func(mockInvoiceRepo *MockInvoiceRepository, mockWalletRepo *MockWalletRepository, mockMovementRepo *MockMovementRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository)
		expectedInvoice domain.Invoice
		expectedError   error
	}{
		"should pay partial invoice successfully": {
			invoiceID: fixture.InvoiceID,
			walletID:  fixture.DefaultWalletID,
			paymentDate: func() *time.Time {
				t := time.Date(2023, 10, 22, 0, 0, 0, 0, time.UTC)
				return &t
			}(),
			amount: func() *float64 {
				a := -1000.0
				return &a
			}(),
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository, mockWalletRepo *MockWalletRepository, mockMovementRepo *MockMovementRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository) {
				invoice := fixture.InvoiceMock(
					fixture.WithInvoiceIsPaid(false),
					fixture.WithInvoiceAmount(-1500.0),
				)
				mockInvoiceRepo.On("FindByID", fixture.InvoiceID).Return(invoice, nil)

				wallet := fixture.WalletMock(fixture.WithWalletBalance(2000.0))
				mockWalletRepo.On("FindByID", &fixture.DefaultWalletID).Return(wallet, nil)

				paidInvoice := fixture.InvoiceMock(
					fixture.WithInvoicePayment(time.Date(2023, 10, 22, 0, 0, 0, 0, time.UTC), fixture.DefaultWalletID),
					fixture.WithInvoiceAmount(-1500.0),
				)

				nextInvoice := fixture.InvoiceMock(
					fixture.WithInvoiceAmount(0),
					fixture.WithInvoiceIsPaid(false),
				)

				mockTxManager.On("WithTransaction", mock.Anything).Run(func(args mock.Arguments) {
					fn := args.Get(0).(func(*gorm.DB) error)

					mockWalletRepo.On("UpdateAmount", mock.Anything, &fixture.DefaultWalletID, mock.AnythingOfType("float64")).Return(nil)
					mockInvoiceRepo.On("UpdateStatus", mock.Anything, mock.Anything, fixture.InvoiceID, true, mock.Anything, &fixture.DefaultWalletID).Return(paidInvoice, nil)

					paymentMovement := fixture.MovementMock(
						fixture.WithMovementAmount(-1000.0),
						fixture.WithMovementDate(time.Date(2023, 10, 22, 0, 0, 0, 0, time.UTC)),
						fixture.WithMovementType(domain.TypePaymentInvoicePayment),
					)
					mockMovementRepo.On("Add", mock.Anything, mock.MatchedBy(func(m domain.Movement) bool {
						return m.TypePayment == domain.TypePaymentInvoicePayment
					})).Return(paymentMovement, nil)

					mockCreditCardRepo.On("UpdateLimitDelta", mock.Anything, fixture.CreditCardID, 1000.0).Return(domain.CreditCard{}, nil)

					mockInvoiceRepo.On("FindByMonthAndCreditCard", mock.Anything, fixture.CreditCardID).Return(nextInvoice, nil)
					mockInvoiceRepo.On("UpdateAmount", mock.Anything, *nextInvoice.ID, -500.0).Return(nextInvoice, nil)

					remainderMovement := fixture.MovementMock(
						fixture.WithMovementAmount(-500.0),
						fixture.WithMovementType(domain.TypePaymentInvoiceRemainder),
					)
					mockMovementRepo.On("Add", mock.Anything, mock.MatchedBy(func(m domain.Movement) bool {
						return m.TypePayment == domain.TypePaymentInvoiceRemainder
					})).Return(remainderMovement, nil)

					mockMovementRepo.On("PayByInvoiceID", mock.Anything, fixture.InvoiceID).Return(nil)

					fn(nil)
				}).Return(nil)
			},
			expectedInvoice: fixture.InvoiceMock(
				fixture.WithInvoicePayment(time.Date(2023, 10, 22, 0, 0, 0, 0, time.UTC), fixture.DefaultWalletID),
			),
			expectedError: nil,
		},
		"should normalize positive amount to negative": {
			invoiceID: fixture.InvoiceID,
			walletID:  fixture.DefaultWalletID,
			paymentDate: func() *time.Time {
				t := time.Date(2023, 10, 22, 0, 0, 0, 0, time.UTC)
				return &t
			}(),
			amount: func() *float64 {
				a := 1000.0
				return &a
			}(),
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository, mockWalletRepo *MockWalletRepository, mockMovementRepo *MockMovementRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository) {
				invoice := fixture.InvoiceMock(
					fixture.WithInvoiceIsPaid(false),
					fixture.WithInvoiceAmount(-1500.0),
				)
				mockInvoiceRepo.On("FindByID", fixture.InvoiceID).Return(invoice, nil)

				wallet := fixture.WalletMock(fixture.WithWalletBalance(2000.0))
				mockWalletRepo.On("FindByID", &fixture.DefaultWalletID).Return(wallet, nil)

				paidInvoice := fixture.InvoiceMock(
					fixture.WithInvoicePayment(time.Date(2023, 10, 22, 0, 0, 0, 0, time.UTC), fixture.DefaultWalletID),
					fixture.WithInvoiceAmount(-1500.0),
				)

				nextInvoice := fixture.InvoiceMock(
					fixture.WithInvoiceAmount(0),
					fixture.WithInvoiceIsPaid(false),
				)

				mockTxManager.On("WithTransaction", mock.Anything).Run(func(args mock.Arguments) {
					fn := args.Get(0).(func(*gorm.DB) error)

					mockWalletRepo.On("UpdateAmount", mock.Anything, &fixture.DefaultWalletID, mock.AnythingOfType("float64")).Return(nil)
					mockInvoiceRepo.On("UpdateStatus", mock.Anything, mock.Anything, fixture.InvoiceID, true, mock.Anything, &fixture.DefaultWalletID).Return(paidInvoice, nil)
					mockMovementRepo.On("Add", mock.Anything, mock.Anything).Return(domain.Movement{}, nil)
					mockCreditCardRepo.On("UpdateLimitDelta", mock.Anything, fixture.CreditCardID, mock.AnythingOfType("float64")).Return(domain.CreditCard{}, nil)
					mockInvoiceRepo.On("FindByMonthAndCreditCard", mock.Anything, fixture.CreditCardID).Return(nextInvoice, nil)
					mockInvoiceRepo.On("UpdateAmount", mock.Anything, *nextInvoice.ID, mock.AnythingOfType("float64")).Return(nextInvoice, nil)

					mockMovementRepo.On("PayByInvoiceID", mock.Anything, fixture.InvoiceID).Return(nil)

					fn(nil)
				}).Return(nil)
			},
			expectedInvoice: fixture.InvoiceMock(
				fixture.WithInvoicePayment(time.Date(2023, 10, 22, 0, 0, 0, 0, time.UTC), fixture.DefaultWalletID),
			),
			expectedError: nil,
		},
		"should fail when amount is greater than invoice amount": {
			invoiceID: fixture.InvoiceID,
			walletID:  fixture.DefaultWalletID,
			paymentDate: func() *time.Time {
				t := time.Date(2023, 10, 22, 0, 0, 0, 0, time.UTC)
				return &t
			}(),
			amount: func() *float64 {
				a := -2000.0
				return &a
			}(),
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository, mockWalletRepo *MockWalletRepository, mockMovementRepo *MockMovementRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository) {
				invoice := fixture.InvoiceMock(
					fixture.WithInvoiceIsPaid(false),
					fixture.WithInvoiceAmount(-1500.0),
				)
				mockInvoiceRepo.On("FindByID", fixture.InvoiceID).Return(invoice, nil)

				wallet := fixture.WalletMock(fixture.WithWalletBalance(3000.0))
				mockWalletRepo.On("FindByID", &fixture.DefaultWalletID).Return(wallet, nil)
			},
			expectedInvoice: domain.Invoice{},
			expectedError:   ErrInvalidPaymentAmount,
		},
		"should fail when amount is zero": {
			invoiceID: fixture.InvoiceID,
			walletID:  fixture.DefaultWalletID,
			paymentDate: func() *time.Time {
				t := time.Date(2023, 10, 22, 0, 0, 0, 0, time.UTC)
				return &t
			}(),
			amount: func() *float64 {
				a := 0.0
				return &a
			}(),
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository, mockWalletRepo *MockWalletRepository, mockMovementRepo *MockMovementRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository) {
				invoice := fixture.InvoiceMock(
					fixture.WithInvoiceIsPaid(false),
					fixture.WithInvoiceAmount(-1500.0),
				)
				mockInvoiceRepo.On("FindByID", fixture.InvoiceID).Return(invoice, nil)

				wallet := fixture.WalletMock(fixture.WithWalletBalance(2000.0))
				mockWalletRepo.On("FindByID", &fixture.DefaultWalletID).Return(wallet, nil)
			},
			expectedInvoice: domain.Invoice{},
			expectedError:   ErrInvalidPaymentAmount,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mockInvoiceRepo := &MockInvoiceRepository{}
			mockCreditCardRepo := &MockCreditCardRepository{}
			mockWalletRepo := &MockWalletRepository{}
			mockTxManager := &MockTransactionManager{}
			mockMovementRepo := &MockMovementRepository{}

			tc.mockSetup(mockInvoiceRepo, mockWalletRepo, mockMovementRepo, mockTxManager, mockCreditCardRepo)
			useCase := NewInvoice(mockInvoiceRepo, mockCreditCardRepo, mockWalletRepo, mockMovementRepo, mockTxManager)
			result, err := useCase.Pay(context.Background(), tc.invoiceID, tc.walletID, tc.paymentDate, tc.amount)

			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError.Error())
			} else {
				assert.NoError(t, err)
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
				mockInvoiceRepo.On("FindOpenByMonth", date).Return(invoices, nil)
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
				mockInvoiceRepo.On("FindOpenByMonth", date).Return([]domain.Invoice{}, nil)
			},
			expectedInvoices: []domain.Invoice{},
			expectedError:    nil,
		},
		"should fail when repository returns error": {
			date: time.Date(2023, 10, 15, 0, 0, 0, 0, time.UTC),
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository) {
				date := time.Date(2023, 10, 15, 0, 0, 0, 0, time.UTC)
				mockInvoiceRepo.On("FindOpenByMonth", date).Return([]domain.Invoice{}, assert.AnError)
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

func TestInvoice_FindDetailedInvoicesByPeriod(t *testing.T) {
	period := domain.Period{
		From: time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2023, 10, 31, 0, 0, 0, 0, time.UTC),
	}

	tests := map[string]struct {
		period                   domain.Period
		mockSetup                func(mockInvoiceRepo *MockInvoiceRepository, mockMovementRepo *MockMovementRepository)
		expectedDetailedInvoices []domain.DetailedInvoice
		expectedError            error
	}{
		"should find detailed invoices successfully": {
			period: period,
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository, mockMovementRepo *MockMovementRepository) {
				invoice := fixture.InvoiceMock(fixture.WithInvoiceAmount(1500.0))
				invoices := []domain.Invoice{invoice}
				mockInvoiceRepo.On("FindByMonth", period.From).Return(invoices, nil)

				movements := domain.MovementList{
					fixture.MovementMock(fixture.WithMovementAmount(-500.0)),
					fixture.MovementMock(fixture.WithMovementAmount(-300.0)),
				}

				mockMovementRepo.On("FindByInvoiceID", *invoice.ID).Return(movements, nil)
			},
			expectedDetailedInvoices: []domain.DetailedInvoice{
				{
					Invoice: fixture.InvoiceMock(fixture.WithInvoiceAmount(1500.0)),
					Movements: domain.MovementList{
						fixture.MovementMock(fixture.WithMovementAmount(-500.0)),
						fixture.MovementMock(fixture.WithMovementAmount(-300.0)),
					},
				},
			},
			expectedError: nil,
		},
		"should return empty list when no invoices found": {
			period: period,
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository, mockMovementRepo *MockMovementRepository) {
				mockInvoiceRepo.On("FindByMonth", period.From).Return([]domain.Invoice{}, nil)
			},
			expectedDetailedInvoices: []domain.DetailedInvoice{},
			expectedError:            nil,
		},
		"should return invoices with empty movements when no movements found": {
			period: period,
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository, mockMovementRepo *MockMovementRepository) {
				invoices := []domain.Invoice{
					fixture.InvoiceMock(fixture.WithInvoiceAmount(1000.0)),
				}
				mockInvoiceRepo.On("FindByMonth", period.From).Return(invoices, nil)
				mockMovementRepo.On("FindByInvoiceID", *invoices[0].ID).Return(domain.MovementList{}, nil)
			},
			expectedDetailedInvoices: []domain.DetailedInvoice{
				{
					Invoice:   fixture.InvoiceMock(fixture.WithInvoiceAmount(1000.0)),
					Movements: domain.MovementList{},
				},
			},
			expectedError: nil,
		},
		"should fail when repo.FindByMonth returns error": {
			period: period,
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository, mockMovementRepo *MockMovementRepository) {
				mockInvoiceRepo.On("FindByMonth", period.From).Return([]domain.Invoice{}, assert.AnError)
			},
			expectedDetailedInvoices: []domain.DetailedInvoice{},
			expectedError:            assert.AnError,
		},
		"should fail when movementRepo.FindByInvoiceID returns error": {
			period: period,
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository, mockMovementRepo *MockMovementRepository) {
				invoices := []domain.Invoice{
					fixture.InvoiceMock(fixture.WithInvoiceAmount(1500.0)),
				}
				mockInvoiceRepo.On("FindByMonth", period.From).Return(invoices, nil)
				mockMovementRepo.On("FindByInvoiceID", *invoices[0].ID).Return(domain.MovementList{}, assert.AnError)
			},
			expectedDetailedInvoices: []domain.DetailedInvoice{},
			expectedError:            assert.AnError,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mockInvoiceRepo := &MockInvoiceRepository{}
			mockCreditCardRepo := &MockCreditCardRepository{}
			mockWalletRepo := &MockWalletRepository{}
			mockTxManager := &MockTransactionManager{}
			mockMovementRepo := &MockMovementRepository{}

			tc.mockSetup(mockInvoiceRepo, mockMovementRepo)

			useCase := NewInvoice(mockInvoiceRepo, mockCreditCardRepo, mockWalletRepo, mockMovementRepo, mockTxManager)
			result, err := useCase.FindDetailedInvoicesByPeriod(context.Background(), tc.period)

			if tc.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.Equal(t, tc.expectedError, err)
				assert.Equal(t, len(tc.expectedDetailedInvoices), len(result))
				for i, expectedDetailedInvoice := range tc.expectedDetailedInvoices {
					assert.Equal(t, expectedDetailedInvoice.Invoice.Amount, result[i].Invoice.Amount)
					assert.Equal(t, len(expectedDetailedInvoice.Movements), len(result[i].Movements))
					for j, expectedMovement := range expectedDetailedInvoice.Movements {
						assert.Equal(t, expectedMovement.Amount, result[i].Movements[j].Amount)
					}
				}
			}

			mockInvoiceRepo.AssertExpectations(t)
			mockMovementRepo.AssertExpectations(t)
		})
	}
}

func TestInvoice_RevertPayment(t *testing.T) {
	tests := map[string]struct {
		invoiceID     uuid.UUID
		mockSetup     func(mockInvoiceRepo *MockInvoiceRepository, mockWalletRepo *MockWalletRepository, mockMovementRepo *MockMovementRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository)
		expectedError error
	}{
		"should revert payment successfully": {
			invoiceID: fixture.InvoiceID,
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository, mockWalletRepo *MockWalletRepository, mockMovementRepo *MockMovementRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository) {
				paymentDate := time.Date(2023, 10, 22, 0, 0, 0, 0, time.UTC)
				invoice := fixture.InvoiceMock(fixture.WithInvoiceIsPaid(true), fixture.WithInvoicePayment(paymentDate, fixture.WalletID))
				mockInvoiceRepo.On("FindByID", fixture.InvoiceID).Return(invoice, nil)

				paymentMovement := fixture.MovementMock(
					fixture.WithMovementAmount(-1500.0),
					fixture.WithMovementType(domain.TypePaymentInvoicePayment),
				)
				mockMovementRepo.On("FindByInvoiceID", *invoice.ID).Return(domain.MovementList{paymentMovement}, nil)

				wallet := fixture.WalletMock(fixture.WithWalletBalance(2000.0))
				mockWalletRepo.On("FindByID", &fixture.WalletID).Return(wallet, nil)

				mockWalletRepo.On("UpdateAmount", mock.Anything, &fixture.WalletID, 3500.0).Return(nil)
				mockInvoiceRepo.On("UpdateStatus", mock.Anything, mock.Anything, fixture.InvoiceID, false, (*time.Time)(nil), (*uuid.UUID)(nil)).Return(invoice, nil)
				mockMovementRepo.On("DeleteByInvoiceID", mock.Anything, fixture.InvoiceID).Return(nil)

				creditCard := fixture.CreditCardMock()
				mockCreditCardRepo.On("UpdateLimitDelta", mock.Anything, fixture.CreditCardID, -1500.0).Return(creditCard, nil)

				mockMovementRepo.On("RevertPayByInvoiceID", mock.Anything, fixture.InvoiceID).Return(nil)

				mockTxManager.On("WithTransaction", mock.Anything).Return(nil)
			},
			expectedError: nil,
		},
		"should fail when invoice not found": {
			invoiceID: fixture.InvoiceID,
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository, mockWalletRepo *MockWalletRepository, mockMovementRepo *MockMovementRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository) {
				mockInvoiceRepo.On("FindByID", fixture.InvoiceID).Return(domain.Invoice{}, ErrInvoiceNotFound)
			},
			expectedError: ErrInvoiceNotFound,
		},
		"should fail when invoice is not paid": {
			invoiceID: fixture.InvoiceID,
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository, mockWalletRepo *MockWalletRepository, mockMovementRepo *MockMovementRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository) {
				invoice := fixture.InvoiceMock(fixture.WithInvoiceIsPaid(false))
				mockInvoiceRepo.On("FindByID", fixture.InvoiceID).Return(invoice, nil)
			},
			expectedError: ErrInvoiceNotPaid,
		},
		"should fail when wallet not found": {
			invoiceID: fixture.InvoiceID,
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository, mockWalletRepo *MockWalletRepository, mockMovementRepo *MockMovementRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository) {
				paymentDate := time.Date(2023, 10, 22, 0, 0, 0, 0, time.UTC)
				invoice := fixture.InvoiceMock(fixture.WithInvoiceIsPaid(true), fixture.WithInvoicePayment(paymentDate, fixture.WalletID))
				mockInvoiceRepo.On("FindByID", fixture.InvoiceID).Return(invoice, nil)

				paymentMovement := fixture.MovementMock(
					fixture.WithMovementAmount(-1500.0),
					fixture.WithMovementType(domain.TypePaymentInvoicePayment),
				)
				mockMovementRepo.On("FindByInvoiceID", *invoice.ID).Return(domain.MovementList{paymentMovement}, nil)

				mockWalletRepo.On("FindByID", &fixture.WalletID).Return(domain.Wallet{}, errors.New("wallet not found"))
			},
			expectedError: errors.New("wallet not found"),
		},
		"should fail when wallet UpdateAmount fails": {
			invoiceID: fixture.InvoiceID,
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository, mockWalletRepo *MockWalletRepository, mockMovementRepo *MockMovementRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository) {
				paymentDate := time.Date(2023, 10, 22, 0, 0, 0, 0, time.UTC)
				invoice := fixture.InvoiceMock(fixture.WithInvoiceIsPaid(true), fixture.WithInvoicePayment(paymentDate, fixture.WalletID))
				mockInvoiceRepo.On("FindByID", fixture.InvoiceID).Return(invoice, nil)

				paymentMovement := fixture.MovementMock(
					fixture.WithMovementAmount(-1500.0),
					fixture.WithMovementType(domain.TypePaymentInvoicePayment),
				)
				mockMovementRepo.On("FindByInvoiceID", *invoice.ID).Return(domain.MovementList{paymentMovement}, nil)

				wallet := fixture.WalletMock(fixture.WithWalletBalance(2000.0))
				mockWalletRepo.On("FindByID", &fixture.WalletID).Return(wallet, nil)

				mockTxManager.On("WithTransaction", mock.Anything).Run(func(args mock.Arguments) {
					fn := args.Get(0).(func(tx *gorm.DB) error)
					mockWalletRepo.On("UpdateAmount", mock.Anything, &fixture.WalletID, 3500.0).Return(errors.New("wallet update failed"))
					_ = fn(nil)
				}).Return(errors.New("error updating wallet balance: wallet update failed"))
			},
			expectedError: errors.New("error updating wallet balance: wallet update failed"),
		},
		"should fail when invoice UpdateStatus fails": {
			invoiceID: fixture.InvoiceID,
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository, mockWalletRepo *MockWalletRepository, mockMovementRepo *MockMovementRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository) {
				paymentDate := time.Date(2023, 10, 22, 0, 0, 0, 0, time.UTC)
				invoice := fixture.InvoiceMock(fixture.WithInvoiceIsPaid(true), fixture.WithInvoicePayment(paymentDate, fixture.WalletID))
				mockInvoiceRepo.On("FindByID", fixture.InvoiceID).Return(invoice, nil)

				paymentMovement := fixture.MovementMock(
					fixture.WithMovementAmount(-1500.0),
					fixture.WithMovementType(domain.TypePaymentInvoicePayment),
				)
				mockMovementRepo.On("FindByInvoiceID", *invoice.ID).Return(domain.MovementList{paymentMovement}, nil)

				wallet := fixture.WalletMock(fixture.WithWalletBalance(2000.0))
				mockWalletRepo.On("FindByID", &fixture.WalletID).Return(wallet, nil)

				mockTxManager.On("WithTransaction", mock.Anything).Run(func(args mock.Arguments) {
					fn := args.Get(0).(func(tx *gorm.DB) error)
					mockWalletRepo.On("UpdateAmount", mock.Anything, &fixture.WalletID, 3500.0).Return(nil)
					mockInvoiceRepo.On("UpdateStatus", mock.Anything, mock.Anything, fixture.InvoiceID, false, (*time.Time)(nil), (*uuid.UUID)(nil)).Return(domain.Invoice{}, errors.New("invoice update failed"))
					_ = fn(nil)
				}).Return(errors.New("error reverting invoice payment status: invoice update failed"))
			},
			expectedError: errors.New("error reverting invoice payment status: invoice update failed"),
		},
		"should fail when movement DeleteByInvoiceID fails": {
			invoiceID: fixture.InvoiceID,
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository, mockWalletRepo *MockWalletRepository, mockMovementRepo *MockMovementRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository) {
				paymentDate := time.Date(2023, 10, 22, 0, 0, 0, 0, time.UTC)
				invoice := fixture.InvoiceMock(fixture.WithInvoiceIsPaid(true), fixture.WithInvoicePayment(paymentDate, fixture.WalletID))
				mockInvoiceRepo.On("FindByID", fixture.InvoiceID).Return(invoice, nil)

				paymentMovement := fixture.MovementMock(
					fixture.WithMovementAmount(-1500.0),
					fixture.WithMovementType(domain.TypePaymentInvoicePayment),
				)
				mockMovementRepo.On("FindByInvoiceID", *invoice.ID).Return(domain.MovementList{paymentMovement}, nil)

				wallet := fixture.WalletMock(fixture.WithWalletBalance(2000.0))
				mockWalletRepo.On("FindByID", &fixture.WalletID).Return(wallet, nil)

				mockTxManager.On("WithTransaction", mock.Anything).Run(func(args mock.Arguments) {
					fn := args.Get(0).(func(tx *gorm.DB) error)
					mockWalletRepo.On("UpdateAmount", mock.Anything, &fixture.WalletID, 3500.0).Return(nil)
					mockInvoiceRepo.On("UpdateStatus", mock.Anything, mock.Anything, fixture.InvoiceID, false, (*time.Time)(nil), (*uuid.UUID)(nil)).Return(invoice, nil)
					mockMovementRepo.On("DeleteByInvoiceID", mock.Anything, fixture.InvoiceID).Return(errors.New("movement delete failed"))
					_ = fn(nil)
				}).Return(errors.New("error deleting invoice movement: movement delete failed"))
			},
			expectedError: errors.New("error deleting invoice movement: movement delete failed"),
		},
		"should fail when has insufficient balance fails": {
			invoiceID: fixture.InvoiceID,
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository, mockWalletRepo *MockWalletRepository, mockMovementRepo *MockMovementRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository) {
				paymentDate := time.Date(2023, 10, 22, 0, 0, 0, 0, time.UTC)
				invoice := fixture.InvoiceMock(fixture.WithInvoiceIsPaid(true), fixture.WithInvoicePayment(paymentDate, fixture.WalletID))
				mockInvoiceRepo.On("FindByID", fixture.InvoiceID).Return(invoice, nil)

				paymentMovement := fixture.MovementMock(
					fixture.WithMovementAmount(-1500.0),
					fixture.WithMovementType(domain.TypePaymentInvoicePayment),
				)
				mockMovementRepo.On("FindByInvoiceID", *invoice.ID).Return(domain.MovementList{paymentMovement}, nil)

				wallet := fixture.WalletMock(fixture.WithWalletBalance(200.0))
				mockWalletRepo.On("FindByID", &fixture.WalletID).Return(wallet, nil)

				mockTxManager.On("WithTransaction", mock.Anything).Run(func(args mock.Arguments) {
					fn := args.Get(0).(func(tx *gorm.DB) error)
					_ = fn(nil)
				}).Return(errors.New("error deleting invoice movement: movement delete failed"))
			},
			expectedError: errors.New("error deleting invoice movement: movement delete failed"),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mockInvoiceRepo := &MockInvoiceRepository{}
			mockCreditCardRepo := &MockCreditCardRepository{}
			mockWalletRepo := &MockWalletRepository{}
			mockTxManager := &MockTransactionManager{}
			mockMovementRepo := &MockMovementRepository{}

			tc.mockSetup(mockInvoiceRepo, mockWalletRepo, mockMovementRepo, mockTxManager, mockCreditCardRepo)
			useCase := NewInvoice(mockInvoiceRepo, mockCreditCardRepo, mockWalletRepo, mockMovementRepo, mockTxManager)
			result, err := useCase.RevertPayment(context.Background(), tc.invoiceID)

			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError.Error())
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}

			mockInvoiceRepo.AssertExpectations(t)
			mockWalletRepo.AssertExpectations(t)
			mockMovementRepo.AssertExpectations(t)
			mockTxManager.AssertExpectations(t)
		})
	}
}

func TestInvoice_RevertPartialPayment(t *testing.T) {
	tests := map[string]struct {
		invoiceID     uuid.UUID
		mockSetup     func(mockInvoiceRepo *MockInvoiceRepository, mockWalletRepo *MockWalletRepository, mockMovementRepo *MockMovementRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository)
		expectedError error
	}{
		"should revert partial payment successfully": {
			invoiceID: fixture.InvoiceID,
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository, mockWalletRepo *MockWalletRepository, mockMovementRepo *MockMovementRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository) {
				paymentDate := time.Date(2023, 10, 22, 0, 0, 0, 0, time.UTC)
				invoice := fixture.InvoiceMock(
					fixture.WithInvoiceIsPaid(true),
					fixture.WithInvoicePayment(paymentDate, fixture.WalletID),
					fixture.WithInvoiceAmount(-1500.0),
				)
				mockInvoiceRepo.On("FindByID", fixture.InvoiceID).Return(invoice, nil)

				paymentMovement := fixture.MovementMock(
					fixture.WithMovementAmount(-1000.0),
					fixture.WithMovementType(domain.TypePaymentInvoicePayment),
				)
				mockMovementRepo.On("FindByInvoiceID", *invoice.ID).Return(domain.MovementList{paymentMovement}, nil)

				wallet := fixture.WalletMock(fixture.WithWalletBalance(3000.0))
				mockWalletRepo.On("FindByID", &fixture.WalletID).Return(wallet, nil)

				nextInvoiceID := uuid.New()
				nextInvoice := fixture.InvoiceMock(
					fixture.WithID(nextInvoiceID),
					fixture.WithInvoiceAmount(-500.0),
					fixture.WithInvoiceIsPaid(false),
				)

				remainderMovementID := uuid.New()
				remainderMovement := fixture.MovementMock(
					fixture.WithMovementID(remainderMovementID),
					fixture.WithMovementAmount(-500.0),
					fixture.WithMovementType(domain.TypePaymentInvoiceRemainder),
					fixture.WithMovementDate(invoice.DueDate.AddDate(0, 0, 1)),
				)

				mockTxManager.On("WithTransaction", mock.Anything).Run(func(args mock.Arguments) {
					fn := args.Get(0).(func(*gorm.DB) error)

					mockWalletRepo.On("UpdateAmount", mock.Anything, &fixture.WalletID, mock.AnythingOfType("float64")).Return(nil)
					mockInvoiceRepo.On("UpdateStatus", mock.Anything, mock.Anything, fixture.InvoiceID, false, (*time.Time)(nil), (*uuid.UUID)(nil)).Return(invoice, nil)
					mockMovementRepo.On("DeleteByInvoiceID", mock.Anything, fixture.InvoiceID).Return(nil)

					mockCreditCardRepo.On("UpdateLimitDelta", mock.Anything, fixture.CreditCardID, -1000.0).Return(domain.CreditCard{}, nil)

					mockMovementRepo.On("RevertPayByInvoiceID", mock.Anything, fixture.InvoiceID).Return(nil)

					mockInvoiceRepo.On("FindByMonthAndCreditCard", mock.Anything, fixture.CreditCardID).Return(nextInvoice, nil)
					mockMovementRepo.On("FindByInvoiceID", nextInvoiceID).Return(domain.MovementList{remainderMovement}, nil)
					mockInvoiceRepo.On("UpdateAmount", mock.Anything, nextInvoiceID, 0.0).Return(nextInvoice, nil)
					mockMovementRepo.On("Delete", mock.Anything, remainderMovementID).Return(nil)

					fn(nil)
				}).Return(nil)
			},
			expectedError: nil,
		},
		"should revert full payment when no remainder exists": {
			invoiceID: fixture.InvoiceID,
			mockSetup: func(mockInvoiceRepo *MockInvoiceRepository, mockWalletRepo *MockWalletRepository, mockMovementRepo *MockMovementRepository, mockTxManager *MockTransactionManager, mockCreditCardRepo *MockCreditCardRepository) {
				paymentDate := time.Date(2023, 10, 22, 0, 0, 0, 0, time.UTC)
				invoice := fixture.InvoiceMock(
					fixture.WithInvoiceIsPaid(true),
					fixture.WithInvoicePayment(paymentDate, fixture.WalletID),
					fixture.WithInvoiceAmount(-1500.0),
				)
				mockInvoiceRepo.On("FindByID", fixture.InvoiceID).Return(invoice, nil)

				paymentMovement := fixture.MovementMock(
					fixture.WithMovementAmount(-1500.0),
					fixture.WithMovementType(domain.TypePaymentInvoicePayment),
				)
				mockMovementRepo.On("FindByInvoiceID", *invoice.ID).Return(domain.MovementList{paymentMovement}, nil)

				wallet := fixture.WalletMock(fixture.WithWalletBalance(3500.0))
				mockWalletRepo.On("FindByID", &fixture.WalletID).Return(wallet, nil)

				mockTxManager.On("WithTransaction", mock.Anything).Run(func(args mock.Arguments) {
					fn := args.Get(0).(func(*gorm.DB) error)

					mockWalletRepo.On("UpdateAmount", mock.Anything, &fixture.WalletID, mock.AnythingOfType("float64")).Return(nil)
					mockInvoiceRepo.On("UpdateStatus", mock.Anything, mock.Anything, fixture.InvoiceID, false, (*time.Time)(nil), (*uuid.UUID)(nil)).Return(invoice, nil)
					mockMovementRepo.On("DeleteByInvoiceID", mock.Anything, fixture.InvoiceID).Return(nil)

					mockCreditCardRepo.On("UpdateLimitDelta", mock.Anything, fixture.CreditCardID, -1500.0).Return(domain.CreditCard{}, nil)

					mockMovementRepo.On("RevertPayByInvoiceID", mock.Anything, fixture.InvoiceID).Return(nil)

					fn(nil)
				}).Return(nil)
			},
			expectedError: nil,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mockInvoiceRepo := &MockInvoiceRepository{}
			mockCreditCardRepo := &MockCreditCardRepository{}
			mockWalletRepo := &MockWalletRepository{}
			mockTxManager := &MockTransactionManager{}
			mockMovementRepo := &MockMovementRepository{}

			tc.mockSetup(mockInvoiceRepo, mockWalletRepo, mockMovementRepo, mockTxManager, mockCreditCardRepo)
			useCase := NewInvoice(mockInvoiceRepo, mockCreditCardRepo, mockWalletRepo, mockMovementRepo, mockTxManager)
			result, err := useCase.RevertPayment(context.Background(), tc.invoiceID)

			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError.Error())
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}

			mockInvoiceRepo.AssertExpectations(t)
			mockWalletRepo.AssertExpectations(t)
			mockMovementRepo.AssertExpectations(t)
			mockTxManager.AssertExpectations(t)
		})
	}
}
