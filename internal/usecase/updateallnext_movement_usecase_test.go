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

func TestMovement_UpdateAllNext(t *testing.T) {
	tests := map[string]struct {
		id               uuid.UUID
		newMovement      domain.Movement
		mockSetup        func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager, mockInvoiceRepo *MockInvoiceRepository, mockCreditCardRepo *MockCreditCardRepository)
		expectedMovement domain.Movement
		expectedError    error
	}{
		"(a) should update recurrent movement and all next when movement and recurrent exist": {
			id: fixture.MovementID,
			newMovement: fixture.MovementMock(
				fixture.WithMovementDescription("Assinatura atualizada"),
				fixture.WithMovementAmount(-150.0),
				fixture.WithMovementIsRecurrent(true),
				fixture.WithMovementRecurrentID(),
				fixture.WithMovementDate(time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)),
			),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager, mockInvoiceRepo *MockInvoiceRepository, mockCreditCardRepo *MockCreditCardRepository) {
				existingMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Assinatura mensal"),
					fixture.WithMovementAmount(-100.0),
					fixture.WithMovementIsRecurrent(true),
					fixture.WithMovementRecurrentID(),
					fixture.WithMovementIsPaid(false),
				)

				recurrent := fixture.RecurrentMovementMock(
					fixture.WithRecurrentMovementDescription("Assinatura mensal"),
					fixture.WithRecurrentMovementAmount(-100.0),
				)

				updatedMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Assinatura atualizada"),
					fixture.WithMovementAmount(-150.0),
					fixture.WithMovementIsRecurrent(true),
					fixture.WithMovementRecurrentID(),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(nil)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(existingMovement, nil)
				mockRecRepo.On("FindByID", fixture.RecurrentMovementID).Return(recurrent, nil)
				mockRecRepo.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(recurrent, nil)
				mockRecRepo.On("Add", mock.Anything, mock.Anything).Return(recurrent, nil)
				mockMovRepo.On("Update", mock.Anything, fixture.MovementID, mock.Anything).Return(updatedMovement, nil)
			},
			expectedMovement: fixture.MovementMock(
				fixture.WithMovementDescription("Assinatura atualizada"),
				fixture.WithMovementAmount(-150.0),
				fixture.WithMovementIsRecurrent(true),
				fixture.WithMovementRecurrentID(),
			),
			expectedError: nil,
		},
		"(b) should create movement from virtual and update all next when only recurrent exists": {
			id: fixture.RecurrentMovementID,
			newMovement: fixture.MovementMock(
				fixture.WithMovementDescription("Assinatura atualizada"),
				fixture.WithMovementAmount(-150.0),
				fixture.WithMovementIsRecurrent(true),
				fixture.WithMovementRecurrentID(),
				fixture.WithMovementDate(time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)),
			),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager, mockInvoiceRepo *MockInvoiceRepository, mockCreditCardRepo *MockCreditCardRepository) {
				recurrent := fixture.RecurrentMovementMock(
					fixture.WithRecurrentMovementDescription("Assinatura mensal"),
					fixture.WithRecurrentMovementAmount(-100.0),
				)

				createdMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Assinatura atualizada"),
					fixture.WithMovementAmount(-150.0),
					fixture.WithMovementIsRecurrent(true),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(nil)

				mockMovRepo.On("FindByID", fixture.RecurrentMovementID).Return(domain.Movement{}, repository.ErrMovementNotFound)
				mockRecRepo.On("FindByID", fixture.RecurrentMovementID).Return(recurrent, nil)
				mockRecRepo.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(recurrent, nil)
				mockRecRepo.On("Add", mock.Anything, mock.Anything).Return(recurrent, nil)
				mockMovRepo.On("Add", mock.Anything, mock.Anything).Return(createdMovement, nil)
			},
			expectedMovement: fixture.MovementMock(
				fixture.WithMovementDescription("Assinatura atualizada"),
				fixture.WithMovementAmount(-150.0),
				fixture.WithMovementIsRecurrent(true),
			),
			expectedError: nil,
		},
		"(c) should update non-recurrent movement without touching recurrence": {
			id: fixture.MovementID,
			newMovement: fixture.MovementMock(
				fixture.WithMovementDescription("Compra avulsa atualizada"),
				fixture.WithMovementAmount(-200.0),
				fixture.WithMovementIsRecurrent(false),
				fixture.WithMovementDate(time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)),
			),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager, mockInvoiceRepo *MockInvoiceRepository, mockCreditCardRepo *MockCreditCardRepository) {
				existingMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Compra avulsa"),
					fixture.WithMovementAmount(-100.0),
					fixture.WithMovementIsRecurrent(false),
					fixture.WithMovementIsPaid(false),
				)
				existingMovement.RecurrentID = nil

				updatedMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Compra avulsa atualizada"),
					fixture.WithMovementAmount(-200.0),
					fixture.WithMovementIsRecurrent(false),
				)
				updatedMovement.RecurrentID = nil

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(nil)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(existingMovement, nil)
				mockMovRepo.On("Update", mock.Anything, fixture.MovementID, mock.Anything).Return(updatedMovement, nil)
			},
			expectedMovement: func() domain.Movement {
				m := fixture.MovementMock(
					fixture.WithMovementDescription("Compra avulsa atualizada"),
					fixture.WithMovementAmount(-200.0),
					fixture.WithMovementIsRecurrent(false),
				)
				m.RecurrentID = nil
				return m
			}(),
			expectedError: nil,
		},
		"should return error when date is nil": {
			id: fixture.MovementID,
			newMovement: func() domain.Movement {
				m := fixture.MovementMock(
					fixture.WithMovementDescription("Movimento sem data"),
				)
				m.Date = nil
				return m
			}(),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager, mockInvoiceRepo *MockInvoiceRepository, mockCreditCardRepo *MockCreditCardRepository) {
			},
			expectedMovement: domain.Movement{},
			expectedError:    ErrDateRequired,
		},
		"should return error when movement and recurrent not found": {
			id: fixture.MovementID,
			newMovement: fixture.MovementMock(
				fixture.WithMovementDescription("Movimento inexistente"),
				fixture.WithMovementDate(time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)),
			),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager, mockInvoiceRepo *MockInvoiceRepository, mockCreditCardRepo *MockCreditCardRepository) {
				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(repository.ErrRecurrentMovementNotFound)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(domain.Movement{}, repository.ErrMovementNotFound)
				mockRecRepo.On("FindByID", fixture.MovementID).Return(domain.RecurrentMovement{}, repository.ErrRecurrentMovementNotFound)
			},
			expectedMovement: domain.Movement{},
			expectedError:    repository.ErrRecurrentMovementNotFound,
		},
		"should update wallet balance when updating paid movement amount": {
			id: fixture.MovementID,
			newMovement: fixture.MovementMock(
				fixture.WithMovementDescription("Movimento pago atualizado"),
				fixture.WithMovementAmount(-200.0),
				fixture.WithMovementIsPaid(true),
				fixture.WithMovementIsRecurrent(true),
				fixture.WithMovementRecurrentID(),
				fixture.WithMovementDate(time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)),
			),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager, mockInvoiceRepo *MockInvoiceRepository, mockCreditCardRepo *MockCreditCardRepository) {
				existingMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Movimento pago"),
					fixture.WithMovementAmount(-100.0),
					fixture.WithMovementIsPaid(true),
					fixture.WithMovementIsRecurrent(true),
					fixture.WithMovementRecurrentID(),
				)

				recurrent := fixture.RecurrentMovementMock()

				updatedMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Movimento pago atualizado"),
					fixture.WithMovementAmount(-200.0),
					fixture.WithMovementIsPaid(true),
					fixture.WithMovementIsRecurrent(true),
					fixture.WithMovementRecurrentID(),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(nil)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(existingMovement, nil)
				mockRecRepo.On("FindByID", fixture.RecurrentMovementID).Return(recurrent, nil)
				mockRecRepo.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(recurrent, nil)
				mockRecRepo.On("Add", mock.Anything, mock.Anything).Return(recurrent, nil)

				mockWalletRepo.On("FindByID", existingMovement.WalletID).Return(domain.Wallet{
					ID:      existingMovement.WalletID,
					Balance: 1000.0,
				}, nil)
				mockWalletRepo.On("UpdateAmount", mock.Anything, existingMovement.WalletID, 900.0).Return(nil)

				mockMovRepo.On("Update", mock.Anything, fixture.MovementID, mock.Anything).Return(updatedMovement, nil)
			},
			expectedMovement: fixture.MovementMock(
				fixture.WithMovementDescription("Movimento pago atualizado"),
				fixture.WithMovementAmount(-200.0),
				fixture.WithMovementIsPaid(true),
				fixture.WithMovementIsRecurrent(true),
				fixture.WithMovementRecurrentID(),
			),
			expectedError: nil,
		},
		"should fail when trying to set recurrent with credit card": {
			id: fixture.MovementID,
			newMovement: fixture.MovementMock(
				fixture.WithMovementDescription("Recorrente com cartao"),
				fixture.WithMovementAmount(-100.0),
				fixture.WithMovementIsRecurrent(true),
				fixture.WithMovementRecurrentID(),
				fixture.WithMovementTypePayment(string(domain.TypePaymentCreditCard)),
				fixture.WithMovementDate(time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)),
			),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager, mockInvoiceRepo *MockInvoiceRepository, mockCreditCardRepo *MockCreditCardRepository) {
			},
			expectedMovement: domain.Movement{},
			expectedError:    ErrRecurrentCreditCardNotSupported,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockMovRepo := new(MockMovementRepository)
			mockRecRepo := new(MockRecurrentRepository)
			mockWalletRepo := new(MockWalletRepository)
			mockSubCat := new(MockSubCategory)
			mockTxManager := new(MockTransactionManager)
			mockInvoiceRepo := &MockInvoiceRepository{}
			mockCreditCardRepo := &MockCreditCardRepository{}

			if tt.mockSetup != nil {
				tt.mockSetup(mockMovRepo, mockRecRepo, mockWalletRepo, mockSubCat, mockTxManager, mockInvoiceRepo, mockCreditCardRepo)
			}

			usecase := NewMovement(
				mockMovRepo,
				mockRecRepo,
				mockWalletRepo,
				mockSubCat,
				mockInvoiceRepo,
				new(MockInvoice),
				mockCreditCardRepo,
				mockTxManager,
			)

			result, err := usecase.UpdateAllNext(context.Background(), tt.id, tt.newMovement)

			assert.Equal(t, tt.expectedError, err)
			assert.Equal(t, tt.expectedMovement, result)

			mockMovRepo.AssertExpectations(t)
			mockRecRepo.AssertExpectations(t)
			mockWalletRepo.AssertExpectations(t)
			mockSubCat.AssertExpectations(t)
			mockTxManager.AssertExpectations(t)
		})
	}
}

func TestMovement_UpdateAllNext_CreditCard(t *testing.T) {
	tests := map[string]struct {
		id               uuid.UUID
		newMovement      domain.Movement
		mockSetup        func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager, mockInvoiceRepo *MockInvoiceRepository, mockCreditCardRepo *MockCreditCardRepository)
		expectedMovement domain.Movement
		expectedError    error
	}{
		"(d) should update single credit card movement": {
			id: fixture.MovementID,
			newMovement: func() domain.Movement {
				m := fixture.MovementMock(
					fixture.WithMovementDescription("Compra cartao atualizada"),
					fixture.WithMovementAmount(-150.0),
					fixture.WithMovementTypePayment(string(domain.TypePaymentCreditCard)),
					fixture.WithMovementCreditCardID(&fixture.CreditCardID),
					fixture.WithMovementDate(time.Date(2025, 2, 15, 10, 0, 0, 0, time.UTC)),
					fixture.WithMovementIsPaid(false),
				)
				m.CreditCardInfo.InvoiceID = &fixture.InvoiceID
				return m
			}(),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager, mockInvoiceRepo *MockInvoiceRepository, mockCreditCardRepo *MockCreditCardRepository) {
				existing := fixture.MovementMock(
					fixture.WithMovementDescription("Compra cartao"),
					fixture.WithMovementAmount(-100.0),
					fixture.WithMovementTypePayment(string(domain.TypePaymentCreditCard)),
					fixture.WithMovementCreditCardID(&fixture.CreditCardID),
					fixture.WithMovementIsPaid(false),
				)
				existing.CreditCardInfo.InvoiceID = &fixture.InvoiceID

				updated := existing
				updated.Description = "Compra cartao atualizada"
				updated.Amount = -150.0

				invoice := fixture.InvoiceMock(
					fixture.WithInvoiceAmount(-1000.0),
					fixture.WithInvoiceIsPaid(false),
				)

				creditCard := fixture.CreditCardMock()

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(nil)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(existing, nil)
				mockInvoiceRepo.On("FindByID", *existing.CreditCardInfo.InvoiceID).Return(invoice, nil)
				mockCreditCardRepo.On("FindByID", fixture.CreditCardID).Return(creditCard, nil)
				mockInvoiceRepo.On("UpdateAmount", mock.Anything, mock.Anything, mock.Anything).Return(invoice, nil)
				mockCreditCardRepo.On("UpdateLimitDelta", mock.Anything, mock.Anything, mock.Anything).Return(creditCard, nil)
				mockMovRepo.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(updated, nil)
			},
			expectedMovement: func() domain.Movement {
				m := fixture.MovementMock(
					fixture.WithMovementDescription("Compra cartao atualizada"),
					fixture.WithMovementAmount(-150.0),
					fixture.WithMovementTypePayment(string(domain.TypePaymentCreditCard)),
					fixture.WithMovementCreditCardID(&fixture.CreditCardID),
					fixture.WithMovementIsPaid(false),
				)
				m.CreditCardInfo.InvoiceID = &fixture.InvoiceID
				return m
			}(),
			expectedError: nil,
		},
		"should block update when invoice is paid": {
			id: fixture.MovementID,
			newMovement: func() domain.Movement {
				m := fixture.MovementMock(
					fixture.WithMovementDescription("Compra cartao atualizada"),
					fixture.WithMovementAmount(-150.0),
					fixture.WithMovementTypePayment(string(domain.TypePaymentCreditCard)),
					fixture.WithMovementCreditCardID(&fixture.CreditCardID),
					fixture.WithMovementDate(time.Date(2025, 2, 15, 10, 0, 0, 0, time.UTC)),
					fixture.WithMovementIsPaid(false),
				)
				m.CreditCardInfo.InvoiceID = &fixture.InvoiceID
				return m
			}(),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager, mockInvoiceRepo *MockInvoiceRepository, mockCreditCardRepo *MockCreditCardRepository) {
				existing := fixture.MovementMock(
					fixture.WithMovementDescription("Compra cartao"),
					fixture.WithMovementAmount(-100.0),
					fixture.WithMovementTypePayment(string(domain.TypePaymentCreditCard)),
					fixture.WithMovementCreditCardID(&fixture.CreditCardID),
					fixture.WithMovementIsPaid(false),
				)
				existing.CreditCardInfo.InvoiceID = &fixture.InvoiceID

				invoicePaid := fixture.InvoiceMock(
					fixture.WithInvoiceAmount(-1000.0),
					fixture.WithInvoiceIsPaid(true),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(ErrInvoiceAlreadyPaid)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(existing, nil)
				mockInvoiceRepo.On("FindByID", mock.Anything).Return(invoicePaid, nil)
			},
			expectedMovement: domain.Movement{},
			expectedError:    ErrInvoiceAlreadyPaid,
		},
		"should block update when credit limit is insufficient": {
			id: fixture.MovementID,
			newMovement: func() domain.Movement {
				m := fixture.MovementMock(
					fixture.WithMovementDescription("Compra muito cara"),
					fixture.WithMovementAmount(-10000.0),
					fixture.WithMovementTypePayment(string(domain.TypePaymentCreditCard)),
					fixture.WithMovementCreditCardID(&fixture.CreditCardID),
					fixture.WithMovementDate(time.Date(2025, 2, 15, 10, 0, 0, 0, time.UTC)),
					fixture.WithMovementIsPaid(false),
				)
				m.CreditCardInfo.InvoiceID = &fixture.InvoiceID
				return m
			}(),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager, mockInvoiceRepo *MockInvoiceRepository, mockCreditCardRepo *MockCreditCardRepository) {
				existing := fixture.MovementMock(
					fixture.WithMovementDescription("Compra cartao"),
					fixture.WithMovementAmount(-100.0),
					fixture.WithMovementTypePayment(string(domain.TypePaymentCreditCard)),
					fixture.WithMovementCreditCardID(&fixture.CreditCardID),
					fixture.WithMovementIsPaid(false),
				)
				existing.CreditCardInfo.InvoiceID = &fixture.InvoiceID

				invoice := fixture.InvoiceMock(
					fixture.WithInvoiceAmount(-100.0),
					fixture.WithInvoiceIsPaid(false),
				)

				creditCard := fixture.CreditCardMock(
					fixture.WithCreditCardLimit(1000.0),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(ErrInsufficientCreditLimit)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(existing, nil)
				mockInvoiceRepo.On("FindByID", mock.Anything).Return(invoice, nil)
				mockCreditCardRepo.On("FindByID", fixture.CreditCardID).Return(creditCard, nil)
			},
			expectedMovement: domain.Movement{},
			expectedError:    ErrInsufficientCreditLimit,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockMovRepo := new(MockMovementRepository)
			mockRecRepo := new(MockRecurrentRepository)
			mockWalletRepo := new(MockWalletRepository)
			mockSubCat := new(MockSubCategory)
			mockTxManager := new(MockTransactionManager)
			mockInvoiceRepo := &MockInvoiceRepository{}
			mockCreditCardRepo := &MockCreditCardRepository{}

			if tt.mockSetup != nil {
				tt.mockSetup(mockMovRepo, mockRecRepo, mockWalletRepo, mockSubCat, mockTxManager, mockInvoiceRepo, mockCreditCardRepo)
			}

			usecase := NewMovement(
				mockMovRepo,
				mockRecRepo,
				mockWalletRepo,
				mockSubCat,
				mockInvoiceRepo,
				new(MockInvoice),
				mockCreditCardRepo,
				mockTxManager,
			)

			result, err := usecase.UpdateAllNext(context.Background(), tt.id, tt.newMovement)

			assert.Equal(t, tt.expectedError, err)
			assert.Equal(t, tt.expectedMovement, result)

			mockMovRepo.AssertExpectations(t)
			mockInvoiceRepo.AssertExpectations(t)
			mockCreditCardRepo.AssertExpectations(t)
		})
	}
}
