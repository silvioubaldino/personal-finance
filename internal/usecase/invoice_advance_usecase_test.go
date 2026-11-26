package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/domain/fixture"
	"personal-finance/internal/infrastructure/repository"
	"personal-finance/pkg/clock"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var advanceMovementID = uuid.MustParse("99999999-9999-9999-9999-999999999999")

func advanceMovement(amount float64, date time.Time) domain.Movement {
	return fixture.MovementMock(
		fixture.WithMovementID(advanceMovementID),
		fixture.WithMovementAmount(amount),
		fixture.WithMovementDate(date),
		fixture.WithMovementType(domain.TypePaymentInvoiceAdvancePayment),
		fixture.WithMovementWalletID(fixture.DefaultWalletID),
		fixture.WithMovementInvoiceID(&fixture.InvoiceID),
		fixture.WithMovementCreditCardID(&fixture.CreditCardID),
		fixture.WithMovementIsPaid(true),
	)
}

func TestInvoice_AdvancePayment(t *testing.T) {
	dueDate := time.Date(2023, 10, 22, 0, 0, 0, 0, time.UTC)
	advanceDate := time.Date(2023, 10, 11, 0, 0, 0, 0, time.UTC)

	type setup struct {
		invoice  domain.Invoice
		existing domain.MovementList
	}

	tests := map[string]struct {
		setup         setup
		input         AdvancePaymentInput
		ctx           context.Context
		expectWrite   bool
		expectedDebit float64
		expectedError error
		expected      func(t *testing.T, invoice domain.Invoice)
	}{
		"should register advance payment debiting wallet and releasing limit": {
			setup: setup{invoice: fixture.InvoiceMock(fixture.WithInvoiceDueDate(dueDate))},
			input: AdvancePaymentInput{Amount: 150, WalletID: &fixture.DefaultWalletID, PaymentDate: &advanceDate},
			ctx:   context.Background(),

			expectWrite:   true,
			expectedDebit: -150,
			expected: func(t *testing.T, invoice domain.Invoice) {
				assert.Equal(t, -1500.0, invoice.Amount)
				assert.False(t, invoice.IsPaid)
			},
		},
		"should normalize negative amount and default wallet to invoice wallet": {
			setup: setup{invoice: fixture.InvoiceMock(fixture.WithInvoiceDueDate(dueDate), func(i *domain.Invoice) { i.WalletID = &fixture.DefaultWalletID })},
			input: AdvancePaymentInput{Amount: -150, PaymentDate: &advanceDate},
			ctx:   context.Background(),

			expectWrite:   true,
			expectedDebit: -150,
		},
		"should default payment date to today": {
			setup: setup{invoice: fixture.InvoiceMock(fixture.WithInvoiceDueDate(dueDate))},
			input: AdvancePaymentInput{Amount: 150, WalletID: &fixture.DefaultWalletID},
			ctx:   clock.WithNow(context.Background(), advanceDate),

			expectWrite:   true,
			expectedDebit: -150,
		},
		"should accept a second advance below the remaining amount": {
			setup: setup{
				invoice:  fixture.InvoiceMock(fixture.WithInvoiceDueDate(dueDate)),
				existing: domain.MovementList{advanceMovement(-1000, advanceDate)},
			},
			input: AdvancePaymentInput{Amount: 499.99, WalletID: &fixture.DefaultWalletID, PaymentDate: &advanceDate},
			ctx:   context.Background(),

			expectWrite:   true,
			expectedDebit: -499.99,
		},
		"should accept advance on the due date": {
			setup: setup{invoice: fixture.InvoiceMock(fixture.WithInvoiceDueDate(dueDate))},
			input: AdvancePaymentInput{Amount: 150, WalletID: &fixture.DefaultWalletID, PaymentDate: &dueDate},
			ctx:   context.Background(),

			expectWrite:   true,
			expectedDebit: -150,
		},
		"should reject paid invoice": {
			setup:         setup{invoice: fixture.InvoiceMock(fixture.WithInvoiceDueDate(dueDate), fixture.WithInvoiceIsPaid(true))},
			input:         AdvancePaymentInput{Amount: 150, WalletID: &fixture.DefaultWalletID, PaymentDate: &advanceDate},
			ctx:           context.Background(),
			expectedError: ErrAdvanceInvoiceAlreadyPaid,
		},
		"should reject zero amount": {
			setup:         setup{invoice: fixture.InvoiceMock(fixture.WithInvoiceDueDate(dueDate))},
			input:         AdvancePaymentInput{Amount: 0, WalletID: &fixture.DefaultWalletID, PaymentDate: &advanceDate},
			ctx:           context.Background(),
			expectedError: ErrInvalidAdvancePaymentAmount,
		},
		"should reject amount equal to the remaining amount": {
			setup:         setup{invoice: fixture.InvoiceMock(fixture.WithInvoiceDueDate(dueDate))},
			input:         AdvancePaymentInput{Amount: 1500, WalletID: &fixture.DefaultWalletID, PaymentDate: &advanceDate},
			ctx:           context.Background(),
			expectedError: ErrAdvancePaymentSettlesInvoice,
		},
		"should reject amount above the remaining amount (no credit balance)": {
			setup:         setup{invoice: fixture.InvoiceMock(fixture.WithInvoiceDueDate(dueDate))},
			input:         AdvancePaymentInput{Amount: 1600, WalletID: &fixture.DefaultWalletID, PaymentDate: &advanceDate},
			ctx:           context.Background(),
			expectedError: ErrAdvancePaymentSettlesInvoice,
		},
		"should compare against the remaining amount after previous advances": {
			setup: setup{
				invoice:  fixture.InvoiceMock(fixture.WithInvoiceDueDate(dueDate)),
				existing: domain.MovementList{advanceMovement(-1000, advanceDate)},
			},
			input:         AdvancePaymentInput{Amount: 500, WalletID: &fixture.DefaultWalletID, PaymentDate: &advanceDate},
			ctx:           context.Background(),
			expectedError: ErrAdvancePaymentSettlesInvoice,
		},
		"should reject payment date after due date": {
			setup:         setup{invoice: fixture.InvoiceMock(fixture.WithInvoiceDueDate(dueDate))},
			input:         AdvancePaymentInput{Amount: 150, WalletID: &fixture.DefaultWalletID, PaymentDate: ptr(dueDate.AddDate(0, 0, 1))},
			ctx:           context.Background(),
			expectedError: ErrAdvancePaymentAfterDueDate,
		},
		"should reject when there is no wallet at all": {
			setup:         setup{invoice: fixture.InvoiceMock(fixture.WithInvoiceDueDate(dueDate))},
			input:         AdvancePaymentInput{Amount: 150, PaymentDate: &advanceDate},
			ctx:           context.Background(),
			expectedError: ErrInvoiceWithoutWallet,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Arrange
			invoiceRepo := &MockInvoiceRepository{}
			creditCardRepo := &MockCreditCardRepository{}
			walletRepo := &MockWalletRepository{}
			txManager := &MockTransactionManager{}
			movementRepo := &MockMovementRepository{}

			invoiceRepo.On("FindByID", fixture.InvoiceID).Return(tc.setup.invoice, nil)
			existing := tc.setup.existing
			if existing == nil {
				existing = domain.MovementList{}
			}
			movementRepo.On("FindAdvancePaymentsByInvoiceIDs", mock.Anything).Return(existing, nil).Maybe()

			if tc.expectWrite {
				wallet := fixture.WalletMock(fixture.WithWalletBalance(5000))
				walletRepo.On("FindByID", &fixture.DefaultWalletID).Return(wallet, nil)
				walletRepo.On("UpdateAmount", mock.Anything, &fixture.DefaultWalletID, 5000+tc.expectedDebit).Return(nil)
				movementRepo.On("Add", mock.Anything, mock.MatchedBy(func(m domain.Movement) bool {
					return m.TypePayment == domain.TypePaymentInvoiceAdvancePayment &&
						m.IsPaid &&
						m.Amount == tc.expectedDebit &&
						m.WalletID != nil && *m.WalletID == fixture.DefaultWalletID &&
						m.CreditCardInfo != nil && *m.CreditCardInfo.InvoiceID == fixture.InvoiceID
				})).Return(domain.Movement{}, nil)
				creditCardRepo.On("UpdateLimitDelta", mock.Anything, fixture.CreditCardID, -tc.expectedDebit).Return(domain.CreditCard{}, nil)
				txManager.On("WithTransaction", mock.Anything).Return(nil)
			}

			useCase := NewInvoice(invoiceRepo, creditCardRepo, walletRepo, movementRepo, txManager)

			// Act
			result, err := useCase.AdvancePayment(tc.ctx, fixture.InvoiceID, tc.input)

			// Assert
			if tc.expectedError != nil {
				assert.ErrorIs(t, err, tc.expectedError)
				walletRepo.AssertNotCalled(t, "UpdateAmount", mock.Anything, mock.Anything, mock.Anything)
				movementRepo.AssertNotCalled(t, "Add", mock.Anything, mock.Anything)
				return
			}
			assert.NoError(t, err)
			if tc.expected != nil {
				tc.expected(t, result)
			}
			walletRepo.AssertExpectations(t)
			movementRepo.AssertExpectations(t)
			creditCardRepo.AssertExpectations(t)
		})
	}
}

func TestInvoice_AdvancePayment_Errors(t *testing.T) {
	dueDate := time.Date(2023, 10, 22, 0, 0, 0, 0, time.UTC)
	date := time.Date(2023, 10, 11, 0, 0, 0, 0, time.UTC)
	input := AdvancePaymentInput{Amount: 150, WalletID: &fixture.DefaultWalletID, PaymentDate: &date}

	tests := map[string]struct {
		mockSetup     func(invoiceRepo *MockInvoiceRepository, walletRepo *MockWalletRepository, movementRepo *MockMovementRepository)
		expectedError error
	}{
		"should propagate invoice not found": {
			mockSetup: func(invoiceRepo *MockInvoiceRepository, walletRepo *MockWalletRepository, movementRepo *MockMovementRepository) {
				invoiceRepo.On("FindByID", fixture.InvoiceID).Return(domain.Invoice{}, repository.ErrInvoiceNotFound)
			},
			expectedError: repository.ErrInvoiceNotFound,
		},
		"should propagate wallet not found": {
			mockSetup: func(invoiceRepo *MockInvoiceRepository, walletRepo *MockWalletRepository, movementRepo *MockMovementRepository) {
				invoiceRepo.On("FindByID", fixture.InvoiceID).Return(fixture.InvoiceMock(fixture.WithInvoiceDueDate(dueDate)), nil)
				movementRepo.On("FindAdvancePaymentsByInvoiceIDs", mock.Anything).Return(domain.MovementList{}, nil)
				walletRepo.On("FindByID", &fixture.DefaultWalletID).Return(domain.Wallet{}, repository.ErrWalletNotFound)
			},
			expectedError: repository.ErrWalletNotFound,
		},
		"should fail when the wallet has insufficient balance": {
			mockSetup: func(invoiceRepo *MockInvoiceRepository, walletRepo *MockWalletRepository, movementRepo *MockMovementRepository) {
				invoiceRepo.On("FindByID", fixture.InvoiceID).Return(fixture.InvoiceMock(fixture.WithInvoiceDueDate(dueDate)), nil)
				movementRepo.On("FindAdvancePaymentsByInvoiceIDs", mock.Anything).Return(domain.MovementList{}, nil)
				walletRepo.On("FindByID", &fixture.DefaultWalletID).Return(fixture.WalletMock(fixture.WithWalletBalance(10)), nil)
			},
			expectedError: domain.ErrWalletInsufficient,
		},
		"should propagate advance lookup failure": {
			mockSetup: func(invoiceRepo *MockInvoiceRepository, walletRepo *MockWalletRepository, movementRepo *MockMovementRepository) {
				invoiceRepo.On("FindByID", fixture.InvoiceID).Return(fixture.InvoiceMock(fixture.WithInvoiceDueDate(dueDate)), nil)
				movementRepo.On("FindAdvancePaymentsByInvoiceIDs", mock.Anything).Return(domain.MovementList{}, errors.New("db down"))
			},
			expectedError: errors.New("db down"),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			invoiceRepo := &MockInvoiceRepository{}
			walletRepo := &MockWalletRepository{}
			movementRepo := &MockMovementRepository{}
			tc.mockSetup(invoiceRepo, walletRepo, movementRepo)
			useCase := NewInvoice(invoiceRepo, &MockCreditCardRepository{}, walletRepo, movementRepo, &MockTransactionManager{})

			_, err := useCase.AdvancePayment(context.Background(), fixture.InvoiceID, input)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectedError.Error())
		})
	}
}

func TestInvoice_RevertAdvancePayment(t *testing.T) {
	date := time.Date(2023, 10, 11, 0, 0, 0, 0, time.UTC)
	otherInvoiceID := uuid.MustParse("12121212-1212-1212-1212-121212121212")

	tests := map[string]struct {
		invoice       domain.Invoice
		movement      domain.Movement
		movementErr   error
		expectRevert  bool
		expectedError error
	}{
		"should revert advance payment returning money and consuming limit": {
			invoice:      fixture.InvoiceMock(),
			movement:     advanceMovement(-150, date),
			expectRevert: true,
		},
		"should reject paid invoice": {
			invoice:       fixture.InvoiceMock(fixture.WithInvoiceIsPaid(true)),
			movement:      advanceMovement(-150, date),
			expectedError: ErrAdvanceInvoiceAlreadyPaid,
		},
		"should return not found when movement does not exist": {
			invoice:       fixture.InvoiceMock(),
			movementErr:   repository.ErrMovementNotFound,
			expectedError: ErrAdvancePaymentNotFound,
		},
		"should return not found when movement is not an advance payment": {
			invoice: fixture.InvoiceMock(),
			movement: fixture.MovementMock(
				fixture.WithMovementID(advanceMovementID),
				fixture.WithMovementType(domain.TypePaymentInvoicePayment),
				fixture.WithMovementInvoiceID(&fixture.InvoiceID),
			),
			expectedError: ErrAdvancePaymentNotFound,
		},
		"should return not found when advance payment belongs to another invoice": {
			invoice: fixture.InvoiceMock(),
			movement: fixture.MovementMock(
				fixture.WithMovementID(advanceMovementID),
				fixture.WithMovementType(domain.TypePaymentInvoiceAdvancePayment),
				fixture.WithMovementInvoiceID(&otherInvoiceID),
			),
			expectedError: ErrAdvancePaymentNotFound,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Arrange
			invoiceRepo := &MockInvoiceRepository{}
			creditCardRepo := &MockCreditCardRepository{}
			walletRepo := &MockWalletRepository{}
			txManager := &MockTransactionManager{}
			movementRepo := &MockMovementRepository{}

			invoiceRepo.On("FindByID", fixture.InvoiceID).Return(tc.invoice, nil)
			movementRepo.On("FindByID", advanceMovementID).Return(tc.movement, tc.movementErr).Maybe()
			movementRepo.On("FindAdvancePaymentsByInvoiceIDs", mock.Anything).Return(domain.MovementList{}, nil).Maybe()

			if tc.expectRevert {
				walletRepo.On("FindByID", &fixture.DefaultWalletID).Return(fixture.WalletMock(fixture.WithWalletBalance(1000)), nil)
				walletRepo.On("UpdateAmount", mock.Anything, &fixture.DefaultWalletID, 1150.0).Return(nil)
				movementRepo.On("Delete", mock.Anything, advanceMovementID).Return(nil)
				creditCardRepo.On("UpdateLimitDelta", mock.Anything, fixture.CreditCardID, -150.0).Return(domain.CreditCard{}, nil)
				txManager.On("WithTransaction", mock.Anything).Return(nil)
			}

			useCase := NewInvoice(invoiceRepo, creditCardRepo, walletRepo, movementRepo, txManager)

			// Act
			result, err := useCase.RevertAdvancePayment(context.Background(), fixture.InvoiceID, advanceMovementID)

			// Assert
			if tc.expectedError != nil {
				assert.ErrorIs(t, err, tc.expectedError)
				movementRepo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, 0.0, result.AdvancePaidAmount)
			assert.Equal(t, -1500.0, result.RemainingAmount)
			walletRepo.AssertExpectations(t)
			movementRepo.AssertExpectations(t)
			creditCardRepo.AssertExpectations(t)
		})
	}
}

func TestInvoice_FindByID_AdvanceDerivedFields(t *testing.T) {
	date := time.Date(2023, 10, 11, 0, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		invoice           domain.Invoice
		advances          domain.MovementList
		expectedAdvance   float64
		expectedRemaining float64
		expectedCount     int
	}{
		"should expose zero advance when there is none": {
			invoice:           fixture.InvoiceMock(),
			advances:          domain.MovementList{},
			expectedAdvance:   0,
			expectedRemaining: -1500,
		},
		"should derive advance and remaining from advance payments": {
			invoice: fixture.InvoiceMock(fixture.WithInvoiceAmount(-1503.90)),
			advances: domain.MovementList{
				advanceMovement(-150, date),
				advanceMovement(-49, date.AddDate(0, 0, 5)),
			},
			expectedAdvance:   -199,
			expectedRemaining: -1304.90,
			expectedCount:     2,
		},
		"should keep remaining as the covered value after the invoice is paid": {
			invoice:           fixture.InvoiceMock(fixture.WithInvoiceAmount(-1503.90), fixture.WithInvoiceIsPaid(true)),
			advances:          domain.MovementList{advanceMovement(-199, date)},
			expectedAdvance:   -199,
			expectedRemaining: -1304.90,
			expectedCount:     1,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			invoiceRepo := &MockInvoiceRepository{}
			movementRepo := &MockMovementRepository{}
			invoiceRepo.On("FindByID", fixture.InvoiceID).Return(tc.invoice, nil)
			movementRepo.On("FindAdvancePaymentsByInvoiceIDs", []uuid.UUID{fixture.InvoiceID}).Return(tc.advances, nil)
			useCase := NewInvoice(invoiceRepo, &MockCreditCardRepository{}, &MockWalletRepository{}, movementRepo, &MockTransactionManager{})

			result, err := useCase.FindByID(context.Background(), fixture.InvoiceID)

			assert.NoError(t, err)
			assert.Equal(t, tc.invoice.Amount, result.Amount)
			assert.Equal(t, tc.expectedAdvance, result.AdvancePaidAmount)
			assert.Equal(t, tc.expectedRemaining, result.RemainingAmount)
			assert.Len(t, result.AdvancePayments, tc.expectedCount)
		})
	}
}

func TestInvoice_Pay_OperatesOnRemainingAmount(t *testing.T) {
	date := time.Date(2023, 10, 11, 0, 0, 0, 0, time.UTC)
	payDate := time.Date(2023, 10, 22, 0, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		invoiceAmount  float64
		advances       domain.MovementList
		amount         *float64
		expectedPaid   float64
		expectedRemain float64 // sobra lançada na fatura seguinte (0 = nenhuma)
		expectedError  error
	}{
		"should pay the remaining amount by default without leftover (Mercado Pago case)": {
			invoiceAmount: -1503.90,
			advances:      domain.MovementList{advanceMovement(-150, date), advanceMovement(-49, date)},
			expectedPaid:  -1304.90,
		},
		"should send to next invoice only what is left of the remaining amount": {
			invoiceAmount:  -1500,
			advances:       domain.MovementList{advanceMovement(-500, date)},
			amount:         ptr(-600.0),
			expectedPaid:   -600,
			expectedRemain: -400,
		},
		"should reject amount greater than the remaining amount": {
			invoiceAmount: -1500,
			advances:      domain.MovementList{advanceMovement(-500, date)},
			amount:        ptr(-1200.0),
			expectedError: ErrInvalidPaymentAmount,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Arrange
			invoiceRepo := &MockInvoiceRepository{}
			creditCardRepo := &MockCreditCardRepository{}
			walletRepo := &MockWalletRepository{}
			txManager := &MockTransactionManager{}
			movementRepo := &MockMovementRepository{}

			invoice := fixture.InvoiceMock(fixture.WithInvoiceAmount(tc.invoiceAmount))
			invoiceRepo.On("FindByID", fixture.InvoiceID).Return(invoice, nil)
			movementRepo.On("FindAdvancePaymentsByInvoiceIDs", mock.Anything).Return(tc.advances, nil)
			walletRepo.On("FindByID", &fixture.DefaultWalletID).Return(fixture.WalletMock(fixture.WithWalletBalance(5000)), nil)

			if tc.expectedError == nil {
				paid := fixture.InvoiceMock(fixture.WithInvoiceAmount(tc.invoiceAmount), fixture.WithInvoicePayment(payDate, fixture.DefaultWalletID))
				walletRepo.On("UpdateAmount", mock.Anything, &fixture.DefaultWalletID, 5000+tc.expectedPaid).Return(nil)
				invoiceRepo.On("UpdateStatus", mock.Anything, mock.Anything, fixture.InvoiceID, true, mock.Anything, &fixture.DefaultWalletID).Return(paid, nil)
				movementRepo.On("Add", mock.Anything, mock.MatchedBy(func(m domain.Movement) bool {
					return m.TypePayment == domain.TypePaymentInvoicePayment && m.Amount == tc.expectedPaid
				})).Return(domain.Movement{}, nil)
				creditCardRepo.On("UpdateLimitDelta", mock.Anything, fixture.CreditCardID, -tc.expectedPaid).Return(domain.CreditCard{}, nil)
				movementRepo.On("PayByInvoiceID", mock.Anything, fixture.InvoiceID).Return(nil)
				txManager.On("WithTransaction", mock.Anything).Return(nil)

				if tc.expectedRemain != 0 {
					next := fixture.InvoiceMock(fixture.WithInvoiceAmount(0))
					invoiceRepo.On("FindByMonthAndCreditCard", mock.Anything, fixture.CreditCardID).Return(next, nil)
					invoiceRepo.On("UpdateAmount", mock.Anything, *next.ID, tc.expectedRemain).Return(next, nil)
					movementRepo.On("Add", mock.Anything, mock.MatchedBy(func(m domain.Movement) bool {
						return m.TypePayment == domain.TypePaymentInvoiceRemainder && m.Amount == tc.expectedRemain
					})).Return(domain.Movement{}, nil)
				}
			}

			useCase := NewInvoice(invoiceRepo, creditCardRepo, walletRepo, movementRepo, txManager)

			// Act
			_, err := useCase.Pay(context.Background(), fixture.InvoiceID, fixture.DefaultWalletID, &payDate, tc.amount)

			// Assert
			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError.Error())
				return
			}
			assert.NoError(t, err)
			movementRepo.AssertExpectations(t)
			invoiceRepo.AssertExpectations(t)
			if tc.expectedRemain == 0 {
				movementRepo.AssertNotCalled(t, "Add", mock.Anything, mock.MatchedBy(func(m domain.Movement) bool {
					return m.TypePayment == domain.TypePaymentInvoiceRemainder
				}))
			}
		})
	}
}

func TestInvoice_RevertPayment_KeepsAdvancePayments(t *testing.T) {
	date := time.Date(2023, 10, 11, 0, 0, 0, 0, time.UTC)

	// Fatura de -1500 com adiantamento de -500 e quitação de -1000: sem sobra, e o
	// revert-pay não apaga o adiantamento (só DeleteByInvoiceID, que mira a quitação).
	invoiceRepo := &MockInvoiceRepository{}
	creditCardRepo := &MockCreditCardRepository{}
	walletRepo := &MockWalletRepository{}
	txManager := &MockTransactionManager{}
	movementRepo := &MockMovementRepository{}

	paymentDate := time.Date(2023, 10, 22, 0, 0, 0, 0, time.UTC)
	invoice := fixture.InvoiceMock(fixture.WithInvoicePayment(paymentDate, fixture.WalletID))
	invoiceRepo.On("FindByID", fixture.InvoiceID).Return(invoice, nil)
	invoiceRepo.On("UpdateStatus", mock.Anything, mock.Anything, fixture.InvoiceID, false, (*time.Time)(nil), (*uuid.UUID)(nil)).Return(invoice, nil)
	movementRepo.On("FindInvoicePaymentByInvoiceID", fixture.InvoiceID).Return(fixture.MovementMock(
		fixture.WithMovementAmount(-1000), fixture.WithMovementType(domain.TypePaymentInvoicePayment)), nil)
	movementRepo.On("FindAdvancePaymentsByInvoiceIDs", mock.Anything).Return(domain.MovementList{advanceMovement(-500, date)}, nil)
	walletRepo.On("FindByID", &fixture.WalletID).Return(fixture.WalletMock(fixture.WithWalletBalance(2000)), nil)
	walletRepo.On("UpdateAmount", mock.Anything, &fixture.WalletID, 3000.0).Return(nil)
	movementRepo.On("DeleteByInvoiceID", mock.Anything, fixture.InvoiceID).Return(nil)
	creditCardRepo.On("UpdateLimitDelta", mock.Anything, fixture.CreditCardID, -1000.0).Return(domain.CreditCard{}, nil)
	movementRepo.On("RevertPayByInvoiceID", mock.Anything, fixture.InvoiceID).Return(nil)
	txManager.On("WithTransaction", mock.Anything).Return(nil)

	useCase := NewInvoice(invoiceRepo, creditCardRepo, walletRepo, movementRepo, txManager)

	result, err := useCase.RevertPayment(context.Background(), fixture.InvoiceID)

	assert.NoError(t, err)
	assert.Equal(t, -500.0, result.AdvancePaidAmount)
	assert.Equal(t, -1000.0, result.RemainingAmount)
	movementRepo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
	movementRepo.AssertExpectations(t)
	creditCardRepo.AssertExpectations(t)
}

func ptr[T any](v T) *T { return &v }
