package usecase

import (
	"context"
	"errors"
	"fmt"
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

func TestMovement_Add(t *testing.T) {
	tests := map[string]struct {
		movementInput    domain.Movement
		mockSetup        func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager, mockInvoiceUseCase *MockInvoice, mockCreditCardRepo *MockCreditCardRepository, mockInvoiceRepo *MockInvoiceRepository)
		expectedMovement domain.Movement
		expectedError    error
	}{
		"should add regular movement with success": {
			movementInput: fixture.MovementMock(
				fixture.WithMovementDescription("Compra no supermercado"),
				fixture.AsMovementExpense(50.0),
			),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager, mockInvoiceUseCase *MockInvoice, mockCreditCardRepo *MockCreditCardRepository, mockInvoiceRepo *MockInvoiceRepository) {
				movement := fixture.MovementMock(
					fixture.WithMovementDescription("Compra no supermercado"),
					fixture.AsMovementExpense(50.0),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(nil)

				mockMovRepo.On("Add", mock.Anything, movement).Return(movement, nil)

				mockWalletRepo.On("FindByID", movement.WalletID).Return(domain.Wallet{
					ID:      movement.WalletID,
					Balance: 1000.0,
				}, nil)

				updatedWallet := domain.Wallet{
					ID:      movement.WalletID,
					Balance: 950.0,
				}
				mockWalletRepo.On("UpdateAmount", mock.Anything, updatedWallet.ID, updatedWallet.Balance).Return(nil)
			},
			expectedMovement: fixture.MovementMock(
				fixture.WithMovementDescription("Compra no supermercado"),
				fixture.AsMovementExpense(50.0),
			),
			expectedError: nil,
		},
		"should add unpaid movement with success": {
			movementInput: fixture.MovementMock(
				fixture.WithMovementDescription("Compra parcelada"),
				fixture.AsMovementExpense(200.0),
				fixture.AsMovementUnpaid(),
			),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager, mockInvoiceUseCase *MockInvoice, mockCreditCardRepo *MockCreditCardRepository, mockInvoiceRepo *MockInvoiceRepository) {
				movement := fixture.MovementMock(
					fixture.WithMovementDescription("Compra parcelada"),
					fixture.AsMovementExpense(200.0),
					fixture.AsMovementUnpaid(),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(nil)

				mockMovRepo.On("Add", mock.Anything, movement).Return(movement, nil)
			},
			expectedMovement: fixture.MovementMock(
				fixture.WithMovementDescription("Compra parcelada"),
				fixture.AsMovementExpense(200.0),
				fixture.AsMovementUnpaid(),
			),
			expectedError: nil,
		},
		"should add recurrent movement with success": {
			movementInput: fixture.MovementMock(
				fixture.WithMovementDescription("Assinatura mensal"),
				fixture.AsMovementExpense(30.0),
				fixture.AsMovementRecurrent(),
			),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager, mockInvoiceUseCase *MockInvoice, mockCreditCardRepo *MockCreditCardRepository, mockInvoiceRepo *MockInvoiceRepository) {
				movementWithoutRecurrentID := fixture.MovementMock(
					fixture.WithMovementDescription("Assinatura mensal"),
					fixture.AsMovementExpense(30.0),
					fixture.AsMovementRecurrent(),
				)

				movementWithRecurrentID := fixture.MovementMock(
					fixture.WithMovementDescription("Assinatura mensal"),
					fixture.AsMovementExpense(30.0),
					fixture.AsMovementRecurrent(),
					fixture.WithMovementRecurrentID(),
				)

				recurrent := domain.ToRecurrentMovement(movementWithoutRecurrentID)
				createdRecurrent := recurrent
				createdRecurrent.ID = &fixture.RecurrentID

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(nil)

				mockRecRepo.On("Add", mock.Anything, recurrent).Return(createdRecurrent, nil)
				mockMovRepo.On("Add", mock.Anything, movementWithRecurrentID).Return(movementWithRecurrentID, nil)

				mockWalletRepo.On("FindByID", movementWithoutRecurrentID.WalletID).Return(domain.Wallet{
					ID:      movementWithoutRecurrentID.WalletID,
					Balance: 1000.0,
				}, nil)

				updatedWallet := domain.Wallet{
					ID:      movementWithoutRecurrentID.WalletID,
					Balance: 970.0,
				}
				mockWalletRepo.On("UpdateAmount", mock.Anything, updatedWallet.ID, updatedWallet.Balance).Return(nil)
			},
			expectedMovement: fixture.MovementMock(
				fixture.WithMovementDescription("Assinatura mensal"),
				fixture.AsMovementExpense(30.0),
				fixture.AsMovementRecurrent(),
				fixture.WithMovementRecurrentID(),
			),
			expectedError: nil,
		},
		"should return error when subcategory does not belong to category": {
			movementInput: fixture.MovementMock(
				fixture.WithMovementDescription("Movimento com subcategoria inválida"),
				fixture.WithMovementSubCategoryID(fixture.SubCategoryID),
			),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager, mockInvoiceUseCase *MockInvoice, mockCreditCardRepo *MockCreditCardRepository, mockInvoiceRepo *MockInvoiceRepository) {
				movement := fixture.MovementMock(
					fixture.WithMovementDescription("Movimento com subcategoria inválida"),
					fixture.WithMovementSubCategoryID(fixture.SubCategoryID),
				)

				mockSubCat.On("IsSubCategoryBelongsToCategory", fixture.SubCategoryID, *movement.CategoryID).Return(false, nil)
			},
			expectedMovement: domain.Movement{},
			expectedError: fmt.Errorf("validate subcategory: %w: %s",
				fmt.Errorf("invalid input"),
				"subcategory does not belong to the provided category",
			),
		},
		"should return error when fails to add movement": {
			movementInput: fixture.MovementMock(
				fixture.WithMovementDescription("Movimento com erro"),
			),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager, mockInvoiceUseCase *MockInvoice, mockCreditCardRepo *MockCreditCardRepository, mockInvoiceRepo *MockInvoiceRepository) {
				movement := fixture.MovementMock(
					fixture.WithMovementDescription("Movimento com erro"),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(errors.New("error when creating movement"))

				mockMovRepo.On("Add", mock.Anything, movement).Return(domain.Movement{}, errors.New("error when creating movement"))
			},
			expectedMovement: domain.Movement{},
			expectedError:    errors.New("error when creating movement"),
		},
		"should return error when fails to find wallet": {
			movementInput: fixture.MovementMock(
				fixture.WithMovementDescription("Movimento com erro na carteira"),
				fixture.AsMovementExpense(100.0),
			),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager, mockInvoiceUseCase *MockInvoice, mockCreditCardRepo *MockCreditCardRepository, mockInvoiceRepo *MockInvoiceRepository) {
				movement := fixture.MovementMock(
					fixture.WithMovementDescription("Movimento com erro na carteira"),
					fixture.AsMovementExpense(100.0),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(errors.New("error when searching wallet"))

				mockMovRepo.On("Add", mock.Anything, movement).Return(movement, nil)

				mockWalletRepo.On("FindByID", movement.WalletID).Return(domain.Wallet{}, errors.New("error when searching wallet"))
			},
			expectedMovement: domain.Movement{},
			expectedError:    errors.New("error when searching wallet"),
		},
		"should return error when fails to update wallet balance": {
			movementInput: fixture.MovementMock(
				fixture.WithMovementDescription("Movimento com erro na atualização da carteira"),
				fixture.AsMovementExpense(150.0),
			),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager, mockInvoiceUseCase *MockInvoice, mockCreditCardRepo *MockCreditCardRepository, mockInvoiceRepo *MockInvoiceRepository) {
				movement := fixture.MovementMock(
					fixture.WithMovementDescription("Movimento com erro na atualização da carteira"),
					fixture.AsMovementExpense(150.0),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(errors.New("error when updating wallet"))

				mockMovRepo.On("Add", mock.Anything, movement).Return(movement, nil)

				mockWalletRepo.On("FindByID", movement.WalletID).Return(domain.Wallet{
					ID:      movement.WalletID,
					Balance: 1000.0,
				}, nil)

				updatedWallet := domain.Wallet{
					ID:      movement.WalletID,
					Balance: 850.0,
				}
				mockWalletRepo.On("UpdateAmount", mock.Anything, updatedWallet.ID, updatedWallet.Balance).Return(errors.New("error when updating wallet"))
			},
			expectedMovement: domain.Movement{},
			expectedError:    errors.New("error when updating wallet"),
		},
		"should return error when fails to create recurrence": {
			movementInput: fixture.MovementMock(
				fixture.WithMovementDescription("Movimento recorrente com erro"),
				fixture.AsMovementExpense(200.0),
				fixture.AsMovementRecurrent(),
			),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager, mockInvoiceUseCase *MockInvoice, mockCreditCardRepo *MockCreditCardRepository, mockInvoiceRepo *MockInvoiceRepository) {
				movement := fixture.MovementMock(
					fixture.WithMovementDescription("Movimento recorrente com erro"),
					fixture.AsMovementExpense(200.0),
					fixture.AsMovementRecurrent(),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(errors.New("error when creating recurrence"))

				recurrent := domain.ToRecurrentMovement(movement)

				mockRecRepo.On("Add", mock.Anything, recurrent).Return(domain.RecurrentMovement{}, errors.New("error when creating recurrence"))
			},
			expectedMovement: domain.Movement{},
			expectedError:    errors.New("error when creating recurrence"),
		},
		"should add credit card movement with success": {
			movementInput: fixture.MovementMock(
				fixture.WithMovementDescription("Compra no cartão de crédito"),
				fixture.AsMovementExpense(150.0),
				fixture.WithMovementTypePayment(string(domain.TypePaymentCreditCard)),
				fixture.WithMovementCreditCardID(&fixture.CreditCardID),
				fixture.WithMovementDate(time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC)),
			),
			mockSetup: func(
				mockMovRepo *MockMovementRepository,
				mockRecRepo *MockRecurrentRepository,
				mockWalletRepo *MockWalletRepository,
				mockSubCat *MockSubCategory,
				mockTxManager *MockTransactionManager,
				mockInvoiceUseCase *MockInvoice,
				mockCreditCardRepo *MockCreditCardRepository,
				mockInvoiceRepo *MockInvoiceRepository) {
				testDate := time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC)
				movement := fixture.MovementMock(
					fixture.WithMovementDescription("Compra no cartão de crédito"),
					fixture.AsMovementExpense(150.0),
					fixture.WithMovementTypePayment(string(domain.TypePaymentCreditCard)),
					fixture.WithMovementCreditCardID(&fixture.CreditCardID),
					fixture.WithMovementDate(testDate),
					fixture.WithMovementIsPaid(false),
				)

				invoice := fixture.InvoiceMock(
					fixture.WithInvoiceAmount(150.0),
					fixture.WithInvoiceIsPaid(false),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(nil)

				mockInvoiceUseCase.On("FindOrCreateInvoiceForMovement", mock.Anything, mock.Anything, &fixture.CreditCardID, mock.Anything).Return(invoice, nil)

				updatedInvoice := invoice
				updatedInvoice.Amount = invoice.Amount + movement.Amount // 150.0 + (-150.0) = 0.0
				mockInvoiceRepo.On("UpdateAmount", mock.Anything, *invoice.ID, updatedInvoice.Amount).Return(updatedInvoice, nil)

				movementWithInvoice := movement
				movementWithInvoice.CreditCardInfo.InvoiceID = invoice.ID

				mockMovRepo.On("Add", mock.Anything, movementWithInvoice).Return(movementWithInvoice, nil)
			},
			expectedMovement: func() domain.Movement {
				testDate := time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC)
				movement := fixture.MovementMock(
					fixture.WithMovementDescription("Compra no cartão de crédito"),
					fixture.AsMovementExpense(150.0),
					fixture.WithMovementTypePayment(string(domain.TypePaymentCreditCard)),
					fixture.WithMovementCreditCardID(&fixture.CreditCardID),
					fixture.WithMovementDate(testDate),
					fixture.WithMovementIsPaid(false),
				)
				movement.CreditCardInfo.InvoiceID = &fixture.InvoiceID
				return movement
			}(),
			expectedError: nil,
		},
		"should fail to add credit card movement when credit card is empty": {
			movementInput: fixture.MovementMock(
				fixture.WithMovementDescription("Compra no cartão de crédito"),
				fixture.AsMovementExpense(150.0),
				fixture.WithMovementTypePayment(string(domain.TypePaymentCreditCard)),
				fixture.WithMovementCreditCardID(nil),
				fixture.WithMovementDate(time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC)),
			),
			mockSetup: func(
				mockMovRepo *MockMovementRepository,
				mockRecRepo *MockRecurrentRepository,
				mockWalletRepo *MockWalletRepository,
				mockSubCat *MockSubCategory,
				mockTxManager *MockTransactionManager,
				mockInvoiceUseCase *MockInvoice,
				mockCreditCardRepo *MockCreditCardRepository,
				mockInvoiceRepo *MockInvoiceRepository) {

				mockInvoiceUseCase.On("FindOrCreateInvoiceForMovement", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(domain.Invoice{}, errors.New("invalid credit card ID"))

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(errors.New("error finding/creating invoice: invalid credit card ID"))
			},
			expectedMovement: domain.Movement{},
			expectedError:    errors.New("error finding/creating invoice: invalid credit card ID"),
		},
		"should fail to add credit card movement when FindOrCreateInvoiceForMovement fails": {
			movementInput: fixture.MovementMock(
				fixture.WithMovementDescription("Compra no cartão de crédito"),
				fixture.AsMovementExpense(150.0),
				fixture.WithMovementTypePayment(string(domain.TypePaymentCreditCard)),
				fixture.WithMovementCreditCardID(&fixture.CreditCardID),
				fixture.WithMovementDate(time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC)),
			),
			mockSetup: func(
				mockMovRepo *MockMovementRepository,
				mockRecRepo *MockRecurrentRepository,
				mockWalletRepo *MockWalletRepository,
				mockSubCat *MockSubCategory,
				mockTxManager *MockTransactionManager,
				mockInvoiceUseCase *MockInvoice,
				mockCreditCardRepo *MockCreditCardRepository,
				mockInvoiceRepo *MockInvoiceRepository) {

				mockInvoiceUseCase.On("FindOrCreateInvoiceForMovement", mock.Anything, mock.Anything, &fixture.CreditCardID, mock.Anything).
					Return(domain.Invoice{}, errors.New("database connection failed"))

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(errors.New("error finding/creating invoice: database connection failed"))
			},
			expectedMovement: domain.Movement{},
			expectedError:    errors.New("error finding/creating invoice: database connection failed"),
		},
		"should fail to add credit card movement when invoiceRepo.UpdateAmount fails": {
			movementInput: fixture.MovementMock(
				fixture.WithMovementDescription("Compra no cartão de crédito"),
				fixture.AsMovementExpense(150.0),
				fixture.WithMovementTypePayment(string(domain.TypePaymentCreditCard)),
				fixture.WithMovementCreditCardID(&fixture.CreditCardID),
				fixture.WithMovementDate(time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC)),
			),
			mockSetup: func(
				mockMovRepo *MockMovementRepository,
				mockRecRepo *MockRecurrentRepository,
				mockWalletRepo *MockWalletRepository,
				mockSubCat *MockSubCategory,
				mockTxManager *MockTransactionManager,
				mockInvoiceUseCase *MockInvoice,
				mockCreditCardRepo *MockCreditCardRepository,
				mockInvoiceRepo *MockInvoiceRepository) {

				invoice := fixture.InvoiceMock(
					fixture.WithInvoiceAmount(150.0),
					fixture.WithInvoiceIsPaid(false),
				)

				mockInvoiceUseCase.On("FindOrCreateInvoiceForMovement", mock.Anything, mock.Anything, &fixture.CreditCardID, mock.Anything).
					Return(invoice, nil)

				mockInvoiceRepo.On("UpdateAmount", mock.Anything, *invoice.ID, mock.Anything).
					Return(domain.Invoice{}, errors.New("failed to update invoice amount"))

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(errors.New("failed to update invoice amount"))
			},
			expectedMovement: domain.Movement{},
			expectedError:    errors.New("failed to update invoice amount"),
		},
		"should fail to add credit card movement when CreditCardInfo is nil": {
			movementInput: fixture.MovementMock(
				fixture.WithMovementDescription("Compra no cartão sem informações"),
				fixture.AsMovementExpense(150.0),
				fixture.WithMovementTypePayment(string(domain.TypePaymentCreditCard)),
				fixture.WithMovementDate(time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC)),
			),
			mockSetup: func(
				mockMovRepo *MockMovementRepository,
				mockRecRepo *MockRecurrentRepository,
				mockWalletRepo *MockWalletRepository,
				mockSubCat *MockSubCategory,
				mockTxManager *MockTransactionManager,
				mockInvoiceUseCase *MockInvoice,
				mockCreditCardRepo *MockCreditCardRepository,
				mockInvoiceRepo *MockInvoiceRepository) {

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(&gorm.DB{})
					}).Return(errors.New("credit_card_info is required for credit card movements")).Once()
			},
			expectedMovement: domain.Movement{},
			expectedError:    errors.New("credit_card_info is required for credit card movements"),
		},
		"should fail to add credit card movement when repository Add fails": {
			movementInput: fixture.MovementMock(
				fixture.WithMovementDescription("Compra no cartão com erro no repositório"),
				fixture.AsMovementExpense(200.0),
				fixture.WithMovementTypePayment(string(domain.TypePaymentCreditCard)),
				fixture.WithMovementCreditCardID(&fixture.CreditCardID),
				fixture.WithMovementDate(time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC)),
				fixture.WithMovementIsPaid(false),
			),
			mockSetup: func(
				mockMovRepo *MockMovementRepository,
				mockRecRepo *MockRecurrentRepository,
				mockWalletRepo *MockWalletRepository,
				mockSubCat *MockSubCategory,
				mockTxManager *MockTransactionManager,
				mockInvoiceUseCase *MockInvoice,
				mockCreditCardRepo *MockCreditCardRepository,
				mockInvoiceRepo *MockInvoiceRepository) {

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(&gorm.DB{})
					}).Return(errors.New("failed to save movement in database")).Once()

				invoice := fixture.InvoiceMock(
					fixture.WithInvoiceAmount(0.0),
					fixture.WithInvoiceIsPaid(false),
				)

				mockInvoiceUseCase.On("FindOrCreateInvoiceForMovement",
					mock.Anything,
					mock.Anything,
					&fixture.CreditCardID,
					mock.Anything,
				).Return(invoice, nil)

				mockInvoiceRepo.On("UpdateAmount",
					mock.Anything,
					*invoice.ID,
					mock.Anything,
				).Return(invoice, nil)

				mockMovRepo.On("Add", mock.Anything, mock.Anything).Return(
					domain.Movement{},
					errors.New("failed to save movement in database"),
				)
			},
			expectedMovement: domain.Movement{},
			expectedError:    errors.New("failed to save movement in database"),
		},
		"should add credit card installment movement (5 installments) with success": {
			movementInput: fixture.MovementMock(
				fixture.WithMovementDescription("Compra parcelada em 5x"),
				fixture.WithMovementAmount(-100.0),
				fixture.WithMovementTypePayment(string(domain.TypePaymentCreditCard)),
				fixture.WithMovementCreditCardID(&fixture.CreditCardID),
				fixture.WithMovementInstallment(1, 5),
				fixture.WithMovementDate(time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC)),
				fixture.WithMovementIsPaid(false),
			),
			mockSetup: func(
				mockMovRepo *MockMovementRepository,
				mockRecRepo *MockRecurrentRepository,
				mockWalletRepo *MockWalletRepository,
				mockSubCat *MockSubCategory,
				mockTxManager *MockTransactionManager,
				mockInvoiceUseCase *MockInvoice,
				mockCreditCardRepo *MockCreditCardRepository,
				mockInvoiceRepo *MockInvoiceRepository) {

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(&gorm.DB{})
					}).Return(nil).Once()

				baseInvoice := fixture.InvoiceMock(
					fixture.WithInvoiceAmount(0.0),
					fixture.WithInvoiceIsPaid(false),
				)

				baseDate := time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC)
				installmentAmount := -100.0 // -500/5 = -100 per installment

				for i := 1; i <= 5; i++ {
					installmentDate := baseDate.AddDate(0, i-1, 0)

					mockInvoiceUseCase.On("FindOrCreateInvoiceForMovement",
						mock.Anything,
						mock.Anything,
						&fixture.CreditCardID,
						installmentDate,
					).Return(baseInvoice, nil)

					newAmount := baseInvoice.Amount + installmentAmount
					updatedInvoice := baseInvoice
					updatedInvoice.Amount = newAmount
					mockInvoiceRepo.On("UpdateAmount",
						mock.Anything,
						*baseInvoice.ID,
						newAmount,
					).Return(updatedInvoice, nil)

					baseInvoice.Amount = newAmount

					expectedMovement := fixture.MovementMock(
						fixture.WithMovementDescription("Compra parcelada em 5x"),
						fixture.WithMovementAmount(installmentAmount),
						fixture.WithMovementTypePayment(string(domain.TypePaymentCreditCard)),
						fixture.WithMovementCreditCardID(&fixture.CreditCardID),
						fixture.WithMovementInstallment(i, 5),
						fixture.WithMovementDate(installmentDate),
						fixture.WithMovementIsPaid(false),
					)
					expectedMovement.CreditCardInfo.InvoiceID = baseInvoice.ID

					mockMovRepo.On("Add", mock.Anything, mock.Anything).Return(expectedMovement, nil)
				}
			},
			expectedMovement: func() domain.Movement {
				testDate := time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC)
				movement := fixture.MovementMock(
					fixture.WithMovementDescription("Compra parcelada em 5x"),
					fixture.WithMovementAmount(-100.0),
					fixture.WithMovementTypePayment(string(domain.TypePaymentCreditCard)),
					fixture.WithMovementCreditCardID(&fixture.CreditCardID),
					fixture.WithMovementInstallment(1, 5),
					fixture.WithMovementDate(testDate),
					fixture.WithMovementIsPaid(false),
				)
				movement.CreditCardInfo.InvoiceID = &fixture.InvoiceID
				return movement
			}(),
			expectedError: nil,
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
			mockInvoiceUseCase := &MockInvoice{}
			mockCreditCardRepo := &MockCreditCardRepository{}

			if tt.mockSetup != nil {
				tt.mockSetup(mockMovRepo, mockRecRepo, mockWalletRepo, mockSubCat, mockTxManager, mockInvoiceUseCase, mockCreditCardRepo, mockInvoiceRepo)
			}

			usecase := NewMovement(
				mockMovRepo,
				mockRecRepo,
				mockWalletRepo,
				mockSubCat,
				mockInvoiceRepo,
				mockInvoiceUseCase,
				mockCreditCardRepo,
				mockTxManager,
			)

			result, err := usecase.Add(context.Background(), tt.movementInput)

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

func TestMovement_FindByPeriod(t *testing.T) {
	tests := map[string]struct {
		periodInput       domain.Period
		mockSetup         func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockInvoiceRepo *MockInvoiceRepository, mockCreditCardRepo *MockCreditCardRepository)
		expectedMovements domain.MovementList
		expectedError     error
	}{
		"should find only non-recurrent movement with success": {
			periodInput: domain.Period{
				From: time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2025, 5, 31, 23, 59, 59, 0, time.UTC),
			},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockInvoiceRepo *MockInvoiceRepository, mockCreditCardRepo *MockCreditCardRepository) {
				mockMovRepo.On("FindByPeriod", mock.Anything, mock.Anything).Return(domain.MovementList{
					fixture.MovementMock(fixture.WithMovementDescription("Compra no supermercado")),
				}, nil)

				mockRecRepo.On("FindByMonth", mock.Anything, mock.Anything).Return([]domain.RecurrentMovement{}, nil)

				mockInvoiceRepo.On("FindOpenByMonth", mock.Anything, mock.Anything).Return([]domain.Invoice{}, nil)
			},
			expectedMovements: domain.MovementList{
				fixture.MovementMock(fixture.WithMovementDescription("Compra no supermercado")),
			},
			expectedError: nil,
		},
		"should find movement with recurrence and ignore recurrent movement": {
			periodInput: domain.Period{
				From: time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2025, 5, 31, 23, 59, 59, 0, time.UTC),
			},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockInvoiceRepo *MockInvoiceRepository, mockCreditCardRepo *MockCreditCardRepository) {
				movement := fixture.MovementMock(
					fixture.WithMovementDescription("Assinatura mensal"),
					fixture.AsMovementRecurrent(),
					fixture.WithMovementRecurrentID(),
				)

				recurrent := fixture.RecurrentMovementMock(
					fixture.WithRecurrentMovementDescription("Assinatura mensal"),
				)

				mockMovRepo.On("FindByPeriod", mock.Anything, mock.Anything).Return(domain.MovementList{movement}, nil)
				mockRecRepo.On("FindByMonth", mock.Anything, mock.Anything).Return([]domain.RecurrentMovement{recurrent}, nil)

				mockInvoiceRepo.On("FindOpenByMonth", mock.Anything, mock.Anything).Return([]domain.Invoice{}, nil)
			},
			expectedMovements: []domain.Movement{
				fixture.MovementMock(
					fixture.WithMovementDescription("Assinatura mensal"),
					fixture.AsMovementRecurrent(),
					fixture.WithMovementRecurrentID(),
				),
			},
			expectedError: nil,
		},
		"should find movement and different recurrent movement with success": {
			periodInput: domain.Period{
				From: time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2025, 5, 31, 23, 59, 59, 0, time.UTC),
			},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockInvoiceRepo *MockInvoiceRepository, mockCreditCardRepo *MockCreditCardRepository) {
				movement := fixture.MovementMock(
					fixture.WithMovementDescription("Compra no supermercado"),
				)

				recurrent := fixture.RecurrentMovementMock(
					fixture.WithRecurrentMovementDescription("Assinatura mensal"),
				)

				mockMovRepo.On("FindByPeriod", mock.Anything, mock.Anything).Return(domain.MovementList{movement}, nil)
				mockRecRepo.On("FindByMonth", mock.Anything, mock.Anything).Return([]domain.RecurrentMovement{recurrent}, nil)

				mockInvoiceRepo.On("FindOpenByMonth", mock.Anything, mock.Anything).Return([]domain.Invoice{}, nil)
			},
			expectedMovements: func() []domain.Movement {
				fromRecurrent := domain.FromRecurrentMovement(
					fixture.RecurrentMovementMock(
						fixture.WithRecurrentMovementDescription("Assinatura mensal"),
					),
					time.Date(2025, 5, 31, 23, 59, 59, 0, time.UTC),
				)
				fromRecurrent.ID = &fixture.RecurrentMovementID

				return []domain.Movement{
					fixture.MovementMock(fixture.WithMovementDescription("Compra no supermercado")),
					fromRecurrent,
				}
			}(),
			expectedError: nil,
		},
		"should return error when fails to find movements": {
			periodInput: domain.Period{
				From: time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2025, 5, 31, 23, 59, 59, 0, time.UTC),
			},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockInvoiceRepo *MockInvoiceRepository, mockCreditCardRepo *MockCreditCardRepository) {
				mockMovRepo.On("FindByPeriod", mock.Anything, mock.Anything).
					Return(domain.MovementList{}, errors.New("error to find transactions"))
			},
			expectedMovements: domain.MovementList{},
			expectedError:     errors.New("error to find transactions"),
		},
		"should return error when fails to find recurrent movements": {
			periodInput: domain.Period{
				From: time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2025, 5, 31, 23, 59, 59, 0, time.UTC),
			},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockInvoiceRepo *MockInvoiceRepository, mockCreditCardRepo *MockCreditCardRepository) {
				mockMovRepo.On("FindByPeriod", mock.Anything, mock.Anything).Return(domain.MovementList{
					fixture.MovementMock(fixture.WithMovementDescription("Compra no supermercado")),
				}, nil)

				mockInvoiceRepo.On("FindOpenByMonth", mock.Anything, mock.Anything).Return([]domain.Invoice{}, nil).Maybe()

				mockRecRepo.On("FindByMonth", mock.Anything, mock.Anything).Return([]domain.RecurrentMovement{}, errors.New("error to find recurrents"))
			},
			expectedMovements: domain.MovementList{},
			expectedError:     errors.New("error to find recurrents"),
		},
		"should find movements and invoices and convert invoices to virtual movements": {
			periodInput: domain.Period{
				From: time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2025, 5, 31, 23, 59, 59, 0, time.UTC),
			},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockInvoiceRepo *MockInvoiceRepository, mockCreditCardRepo *MockCreditCardRepository) {
				movement := fixture.MovementMock(
					fixture.WithMovementDescription("Compra no supermercado"),
				)

				invoice1 := fixture.InvoiceMock(
					fixture.WithInvoiceAmount(500.0),
					fixture.WithInvoiceIsPaid(false),
				)
				dueDate1 := time.Date(2025, 5, 15, 0, 0, 0, 0, time.UTC)
				invoice1.DueDate = dueDate1
				invoice1.UserID = "user-test-id"
				invoice1.WalletID = &fixture.DefaultWalletID

				invoice2 := fixture.InvoiceMock(
					fixture.WithInvoiceAmount(300.0),
					fixture.WithInvoiceIsPaid(false),
				)
				dueDate2 := time.Date(2025, 5, 20, 0, 0, 0, 0, time.UTC)
				invoice2.DueDate = dueDate2
				invoice2.UserID = "user-test-id"
				invoice2.WalletID = &fixture.DefaultWalletID

				mockMovRepo.On("FindByPeriod", mock.Anything, mock.Anything).Return(domain.MovementList{movement}, nil)
				mockRecRepo.On("FindByMonth", mock.Anything, mock.Anything).Return([]domain.RecurrentMovement{}, nil)
				mockInvoiceRepo.On("FindOpenByMonth", mock.Anything, mock.Anything).Return([]domain.Invoice{invoice1, invoice2}, nil)

				mockCreditCardRepo.On("FindNameByID", fixture.CreditCardID).Return("Test Credit Card", nil)
			},
			expectedMovements: func() []domain.Movement {
				regularMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Compra no supermercado"),
				)

				dueDate1 := time.Date(2025, 5, 15, 0, 0, 0, 0, time.UTC)
				virtualMovement1 := domain.Movement{
					ID:          &fixture.InvoiceID,
					Description: "Pagamento da fatura Test Credit Card",
					Amount:      500.0,
					Date:        &dueDate1,
					UserID:      "user-test-id",
					CreditCardInfo: &domain.CreditCardMovement{
						InvoiceID:    &fixture.InvoiceID,
						CreditCardID: &fixture.CreditCardID,
					},
					WalletID:    &fixture.DefaultWalletID,
					TypePayment: "invoice_payment",
				}

				dueDate2 := time.Date(2025, 5, 20, 0, 0, 0, 0, time.UTC)
				virtualMovement2 := domain.Movement{
					ID:          &fixture.InvoiceID,
					Description: "Pagamento da fatura Test Credit Card",
					Amount:      300.0,
					Date:        &dueDate2,
					UserID:      "user-test-id",
					CreditCardInfo: &domain.CreditCardMovement{
						InvoiceID:    &fixture.InvoiceID,
						CreditCardID: &fixture.CreditCardID,
					},
					WalletID:    &fixture.DefaultWalletID,
					TypePayment: "invoice_payment",
				}

				return []domain.Movement{regularMovement, virtualMovement1, virtualMovement2}
			}(),
			expectedError: nil,
		},
		"should return error when creditCardRepo.FindNameByID fails": {
			periodInput: domain.Period{
				From: time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2025, 5, 31, 23, 59, 59, 0, time.UTC),
			},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockInvoiceRepo *MockInvoiceRepository, mockCreditCardRepo *MockCreditCardRepository) {
				movement := fixture.MovementMock(
					fixture.WithMovementDescription("Compra no supermercado"),
				)

				invoice := fixture.InvoiceMock(
					fixture.WithInvoiceAmount(500.0),
					fixture.WithInvoiceIsPaid(false),
				)
				dueDate := time.Date(2025, 5, 15, 0, 0, 0, 0, time.UTC)
				invoice.DueDate = dueDate
				invoice.UserID = "user-test-id"
				invoice.WalletID = &fixture.DefaultWalletID

				mockMovRepo.On("FindByPeriod", mock.Anything, mock.Anything).Return(domain.MovementList{movement}, nil)
				mockRecRepo.On("FindByMonth", mock.Anything, mock.Anything).Return([]domain.RecurrentMovement{}, nil)
				mockInvoiceRepo.On("FindOpenByMonth", mock.Anything, mock.Anything).Return([]domain.Invoice{invoice}, nil)

				mockCreditCardRepo.On("FindNameByID", fixture.CreditCardID).Return("", errors.New("credit card not found"))
			},
			expectedMovements: domain.MovementList{},
			expectedError:     errors.New("credit card not found"),
		},
		"should return error when invoiceRepo.FindByMonth fails": {
			periodInput: domain.Period{
				From: time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2025, 5, 31, 23, 59, 59, 0, time.UTC),
			},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockInvoiceRepo *MockInvoiceRepository, mockCreditCardRepo *MockCreditCardRepository) {
				movement := fixture.MovementMock(
					fixture.WithMovementDescription("Compra no supermercado"),
				)

				mockMovRepo.On("FindByPeriod", mock.Anything, mock.Anything).Return(domain.MovementList{movement}, nil)
				mockRecRepo.On("FindByMonth", mock.Anything, mock.Anything).Return([]domain.RecurrentMovement{}, nil)

				mockInvoiceRepo.On("FindOpenByMonth", mock.Anything, mock.Anything).Return([]domain.Invoice{}, errors.New("database connection failed"))
			},
			expectedMovements: domain.MovementList{},
			expectedError:     errors.New("database connection failed"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockMovRepo := new(MockMovementRepository)
			mockRecRepo := new(MockRecurrentRepository)
			mockInvoiceRepo := &MockInvoiceRepository{}
			mockCreditCardRepo := &MockCreditCardRepository{}

			if tt.mockSetup != nil {
				tt.mockSetup(mockMovRepo, mockRecRepo, mockInvoiceRepo, mockCreditCardRepo)
			}

			usecase := NewMovement(
				mockMovRepo,
				mockRecRepo,
				new(MockWalletRepository),
				new(MockSubCategory),
				mockInvoiceRepo,
				new(MockInvoice),
				mockCreditCardRepo,
				new(MockTransactionManager),
			)

			result, err := usecase.FindByPeriod(context.Background(), tt.periodInput)

			if tt.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.Equal(t, tt.expectedError, err)
			}
			assert.Equal(t, tt.expectedMovements, result)

			mockMovRepo.AssertExpectations(t)
			mockRecRepo.AssertExpectations(t)
			mockInvoiceRepo.AssertExpectations(t)
			mockCreditCardRepo.AssertExpectations(t)
		})
	}
}

func TestMovement_Pay(t *testing.T) {
	id := uuid.New()

	tests := map[string]struct {
		id               uuid.UUID
		date             time.Time
		mockSetup        func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager)
		expectedMovement domain.Movement
		expectedError    error
	}{
		"should pay existing movement": {
			id:   fixture.MovementID,
			date: time.Now(),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				movement := fixture.MovementMock()
				movement.IsPaid = false

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(nil)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(movement, nil)
				movementPaid := movement
				movementPaid.IsPaid = true
				mockMovRepo.On("UpdateIsPaid", mock.Anything, fixture.MovementID, movementPaid).Return(movementPaid, nil)
				mockWalletRepo.On("FindByID", movement.WalletID).Return(domain.Wallet{ID: movement.WalletID, Balance: 1000.0}, nil)
				mockWalletRepo.On("UpdateAmount", mock.Anything, movement.WalletID, 1000.0+movement.Amount).Return(nil)
			},
			expectedMovement: func() domain.Movement {
				m := fixture.MovementMock()
				m.IsPaid = true
				return m
			}(),
			expectedError: nil,
		},
		"should pay recurrent movement": {
			id:   fixture.RecurrentMovementID,
			date: time.Now(),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				recurrent := fixture.RecurrentMovementMock()
				recurrent.Wallet.ID = &id
				recurrent.WalletID = &id
				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(nil)
				mockMovRepo.On("FindByID", fixture.RecurrentMovementID).Return(domain.Movement{}, domain.ErrNotFound)
				mockRecRepo.On("FindByID", fixture.RecurrentMovementID).Return(recurrent, nil)
				mov := domain.FromRecurrentMovement(recurrent, time.Now())
				mov.IsPaid = true

				mockMovRepo.On("Add", mock.Anything, mov).Return(mov, nil)
				mockWalletRepo.On("FindByID", mov.WalletID).Return(domain.Wallet{ID: mov.WalletID, Balance: 1000.0}, nil)
				mockWalletRepo.On("UpdateAmount", mock.Anything, mov.WalletID, 1000.0+mov.Amount).Return(nil)
			},
			expectedMovement: func() domain.Movement {
				mov := domain.FromRecurrentMovement(fixture.RecurrentMovementMock(), time.Now())
				mov.IsPaid = true
				mov.WalletID = &id
				mov.Wallet.ID = &id
				return mov
			}(),
			expectedError: nil,
		},
		"should return error when wallet has insufficient balance": {
			id:   fixture.RecurrentMovementID,
			date: time.Now(),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				recurrent := fixture.RecurrentMovementMock()
				recurrent.Wallet.ID = &id
				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(fmt.Errorf("error updating wallet: %w", ErrInsufficientBalance))
				mockMovRepo.On("FindByID", fixture.RecurrentMovementID).Return(domain.Movement{}, domain.ErrNotFound)
				mockRecRepo.On("FindByID", fixture.RecurrentMovementID).Return(recurrent, nil)
				mov := domain.FromRecurrentMovement(recurrent, time.Now())
				mov.IsPaid = true
				mockMovRepo.On("Add", mock.Anything, mov).Return(mov, nil)
				mockWalletRepo.On("FindByID", mov.WalletID).Return(domain.Wallet{ID: mov.WalletID, Balance: 10.0}, nil)
			},
			expectedMovement: domain.Movement{},
			expectedError:    fmt.Errorf("error updating wallet: %w", ErrInsufficientBalance),
		},
		"should return error if date is zero for recurrent": {
			id:   fixture.RecurrentMovementID,
			date: time.Time{},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(fmt.Errorf("error paying movement with id: %s: %w", fixture.RecurrentMovementID, ErrDateRequired))
				mockMovRepo.On("FindByID", fixture.RecurrentMovementID).Return(domain.Movement{}, domain.ErrNotFound)
				mockRecRepo.On("FindByID", fixture.RecurrentMovementID).Return(fixture.RecurrentMovementMock(), nil)
			},
			expectedMovement: domain.Movement{},
			expectedError:    fmt.Errorf("error paying movement with id: %s: %w", fixture.RecurrentMovementID, ErrDateRequired),
		},
		"should return error if already paid": {
			id:   fixture.MovementID,
			date: time.Now(),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				movement := fixture.MovementMock(
					fixture.WithMovementIsPaid(true),
				)
				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(fmt.Errorf("error paying movement with id: %s: %w", fixture.MovementID, ErrMovementAlreadyPaid))
				mockMovRepo.On("FindByID", fixture.MovementID).Return(movement, nil)
			},
			expectedMovement: domain.Movement{},
			expectedError:    fmt.Errorf("error paying movement with id: %s: %w", fixture.MovementID, ErrMovementAlreadyPaid),
		},
		"should return error when find movement fail": {
			id:   fixture.RecurrentMovementID,
			date: time.Now(),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				recurrent := fixture.RecurrentMovementMock()
				recurrent.Wallet.ID = &id
				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(assert.AnError)
				mockMovRepo.On("FindByID", fixture.RecurrentMovementID).Return(domain.Movement{}, assert.AnError)
			},
			expectedMovement: domain.Movement{},
			expectedError:    assert.AnError,
		},
		"should return error when find recurrent movement fail": {
			id:   fixture.RecurrentMovementID,
			date: time.Now(),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				recurrent := fixture.RecurrentMovementMock()
				recurrent.Wallet.ID = &id
				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(assert.AnError)
				mockMovRepo.On("FindByID", fixture.RecurrentMovementID).Return(domain.Movement{}, domain.ErrNotFound)
				mockRecRepo.On("FindByID", fixture.RecurrentMovementID).
					Return(domain.RecurrentMovement{}, assert.AnError)
			},
			expectedMovement: domain.Movement{},
			expectedError:    assert.AnError,
		},
		"should return error when add movement fail": {
			id:   fixture.RecurrentMovementID,
			date: time.Now(),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				recurrent := fixture.RecurrentMovementMock()
				recurrent.Wallet.ID = &id
				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(assert.AnError)
				mockMovRepo.On("FindByID", fixture.RecurrentMovementID).Return(domain.Movement{}, domain.ErrNotFound)
				mockRecRepo.On("FindByID", fixture.RecurrentMovementID).Return(recurrent, nil)
				mov := domain.FromRecurrentMovement(recurrent, time.Now())
				mov.IsPaid = true
				mockMovRepo.On("Add", mock.Anything, mov).Return(domain.Movement{}, assert.AnError)
			},
			expectedMovement: domain.Movement{},
			expectedError:    assert.AnError,
		},
		"should return error when update existing movement fail": {
			id:   fixture.MovementID,
			date: time.Now(),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				movement := fixture.MovementMock()
				movement.IsPaid = false

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(assert.AnError)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(movement, nil)
				movementPaid := movement
				movementPaid.IsPaid = true
				mockMovRepo.On("UpdateIsPaid", mock.Anything, fixture.MovementID, movementPaid).
					Return(domain.Movement{}, assert.AnError)
			},
			expectedMovement: domain.Movement{},
			expectedError:    assert.AnError,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockMovRepo := new(MockMovementRepository)
			mockRecRepo := new(MockRecurrentRepository)
			mockWalletRepo := new(MockWalletRepository)
			mockTxManager := new(MockTransactionManager)

			if tt.mockSetup != nil {
				tt.mockSetup(mockMovRepo, mockRecRepo, mockWalletRepo, mockTxManager)
			}

			mockInvoiceRepo := &MockInvoiceRepository{}
			usecase := NewMovement(
				mockMovRepo,
				mockRecRepo,
				mockWalletRepo,
				new(MockSubCategory),
				mockInvoiceRepo,
				new(MockInvoice),
				new(MockCreditCardRepository),
				mockTxManager,
			)

			result, err := usecase.Pay(context.Background(), tt.id, tt.date)

			assert.Equal(t, tt.expectedError, err)
			assert.Equal(t, tt.expectedMovement, result)

			mockMovRepo.AssertExpectations(t)
			mockRecRepo.AssertExpectations(t)
			mockWalletRepo.AssertExpectations(t)
			mockTxManager.AssertExpectations(t)
		})
	}
}

func TestMovement_RevertPay(t *testing.T) {
	tests := map[string]struct {
		id               uuid.UUID
		mockSetup        func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager)
		expectedMovement domain.Movement
		expectedError    error
	}{
		"should revert pay existing movement": {
			id: fixture.MovementID,
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				movement := fixture.MovementMock(
					fixture.WithMovementIsPaid(true),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(nil)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(movement, nil)
				movementUnpaid := movement
				movementUnpaid.IsPaid = false
				mockMovRepo.On("UpdateIsPaid", mock.Anything, fixture.MovementID, movementUnpaid).Return(movementUnpaid, nil)
				mockWalletRepo.On("FindByID", movement.WalletID).Return(domain.Wallet{ID: movement.WalletID, Balance: 1000.0}, nil)
				mockWalletRepo.On("UpdateAmount", mock.Anything, movement.WalletID, float64(1100)).Return(nil)
			},
			expectedMovement: func() domain.Movement {
				m := fixture.MovementMock(
					fixture.WithMovementIsPaid(false),
				)
				return m
			}(),
			expectedError: nil,
		},
		"should return error when movement is not paid": {
			id: fixture.MovementID,
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				movement := fixture.MovementMock(
					fixture.WithMovementIsPaid(false),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(fmt.Errorf("error finding movement with id: %s: %w", fixture.MovementID, ErrMovementNotPaid))

				mockMovRepo.On("FindByID", fixture.MovementID).Return(movement, nil)
			},
			expectedMovement: domain.Movement{},
			expectedError:    fmt.Errorf("error finding movement with id: %s: %w", fixture.MovementID, ErrMovementNotPaid),
		},
		"should return error when movement not found": {
			id: fixture.MovementID,
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(fmt.Errorf("error finding movement with id: %s: %w", fixture.MovementID, domain.ErrNotFound))

				mockMovRepo.On("FindByID", fixture.MovementID).Return(domain.Movement{}, domain.ErrNotFound)
			},
			expectedMovement: domain.Movement{},
			expectedError:    fmt.Errorf("error finding movement with id: %s: %w", fixture.MovementID, domain.ErrNotFound),
		},
		"should return error when fails to update movement": {
			id: fixture.MovementID,
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				movement := fixture.MovementMock(
					fixture.WithMovementIsPaid(true),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(fmt.Errorf("error updating movement: %w", assert.AnError))

				mockMovRepo.On("FindByID", fixture.MovementID).Return(movement, nil)
				movementUnpaid := movement
				movementUnpaid.IsPaid = false
				mockMovRepo.On("UpdateIsPaid", mock.Anything, fixture.MovementID, movementUnpaid).Return(domain.Movement{}, assert.AnError)
			},
			expectedMovement: domain.Movement{},
			expectedError:    fmt.Errorf("error updating movement: %w", assert.AnError),
		},
		"should return error when fails to update wallet balance": {
			id: fixture.MovementID,
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				movement := fixture.MovementMock(
					fixture.WithMovementIsPaid(true),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(fmt.Errorf("error updating wallet: %w", assert.AnError))

				mockMovRepo.On("FindByID", fixture.MovementID).Return(movement, nil)
				movementUnpaid := movement
				movementUnpaid.IsPaid = false
				mockMovRepo.On("UpdateIsPaid", mock.Anything, fixture.MovementID, movementUnpaid).Return(movementUnpaid, nil)
				mockWalletRepo.On("FindByID", movement.WalletID).Return(domain.Wallet{ID: movement.WalletID, Balance: 1000.0}, nil)
				mockWalletRepo.On("UpdateAmount", mock.Anything, movement.WalletID, float64(1100)).Return(assert.AnError)
			},
			expectedMovement: domain.Movement{},
			expectedError:    fmt.Errorf("error updating wallet: %w", assert.AnError),
		},
		"should return error when wallet has insufficient balance": {
			id: fixture.MovementID,
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				movement := fixture.MovementMock(
					fixture.WithMovementIsPaid(true),
					fixture.AsMovementIncome(5000.0),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(fmt.Errorf("error updating wallet: %w", ErrInsufficientBalance))

				mockMovRepo.On("FindByID", fixture.MovementID).Return(movement, nil)
				movementUnpaid := movement
				movementUnpaid.IsPaid = false
				mockMovRepo.On("UpdateIsPaid", mock.Anything, fixture.MovementID, movementUnpaid).Return(movementUnpaid, nil)
				mockWalletRepo.On("FindByID", movement.WalletID).Return(domain.Wallet{ID: movement.WalletID, Balance: 1000.0}, nil)
			},
			expectedMovement: domain.Movement{},
			expectedError:    fmt.Errorf("error updating wallet: %w", ErrInsufficientBalance),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockMovRepo := new(MockMovementRepository)
			mockRecRepo := new(MockRecurrentRepository)
			mockWalletRepo := new(MockWalletRepository)
			mockTxManager := new(MockTransactionManager)

			if tt.mockSetup != nil {
				tt.mockSetup(mockMovRepo, mockRecRepo, mockWalletRepo, mockTxManager)
			}

			mockInvoiceRepo := &MockInvoiceRepository{}
			usecase := NewMovement(
				mockMovRepo,
				mockRecRepo,
				mockWalletRepo,
				new(MockSubCategory),
				mockInvoiceRepo,
				new(MockInvoice),
				new(MockCreditCardRepository),
				mockTxManager,
			)

			result, err := usecase.RevertPay(context.Background(), tt.id)

			assert.Equal(t, tt.expectedError, err)
			assert.Equal(t, tt.expectedMovement, result)

			mockMovRepo.AssertExpectations(t)
			mockRecRepo.AssertExpectations(t)
			mockWalletRepo.AssertExpectations(t)
			mockTxManager.AssertExpectations(t)
		})
	}
}

func TestMovement_UpdateOne(t *testing.T) {
	tests := map[string]struct {
		id               uuid.UUID
		newMovement      domain.Movement
		mockSetup        func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager)
		expectedMovement domain.Movement
		expectedError    error
	}{
		"should update unpaid movement with success": {
			id: fixture.MovementID,
			newMovement: fixture.MovementMock(
				fixture.WithMovementDescription("Movimento atualizado"),
				fixture.WithMovementAmount(-200.0),
				fixture.WithMovementIsPaid(false),
			),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager) {
				existingMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Movimento original"),
					fixture.WithMovementAmount(-100.0),
					fixture.WithMovementIsPaid(false),
				)

				updatedMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Movimento atualizado"),
					fixture.WithMovementAmount(-200.0),
					fixture.WithMovementIsPaid(false),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(nil)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(existingMovement, nil)
				mockMovRepo.On("UpdateOne", mock.Anything, fixture.MovementID, mock.Anything).Return(updatedMovement, nil)
			},
			expectedMovement: fixture.MovementMock(
				fixture.WithMovementDescription("Movimento atualizado"),
				fixture.WithMovementAmount(-200.0),
				fixture.WithMovementIsPaid(false),
			),
			expectedError: nil,
		},
		"should update paid movement changing amount with success": {
			id: fixture.MovementID,
			newMovement: fixture.MovementMock(
				fixture.WithMovementDescription("Movimento atualizado"),
				fixture.WithMovementAmount(-200.0),
				fixture.WithMovementIsPaid(true),
			),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager) {
				existingMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Movimento original"),
					fixture.WithMovementAmount(-100.0),
					fixture.WithMovementIsPaid(true),
				)

				updatedMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Movimento atualizado"),
					fixture.WithMovementAmount(-200.0),
					fixture.WithMovementIsPaid(true),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(nil)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(existingMovement, nil)
				mockMovRepo.On("UpdateOne", mock.Anything, fixture.MovementID, mock.Anything).Return(updatedMovement, nil)

				// Chamada para updateWalletBalance com diferença: -200 - (-100) = -100
				mockWalletRepo.On("FindByID", existingMovement.WalletID).Return(domain.Wallet{
					ID:      existingMovement.WalletID,
					Balance: 1000.0,
				}, nil)
				mockWalletRepo.On("UpdateAmount", mock.Anything, existingMovement.WalletID, 900.0).Return(nil)
			},
			expectedMovement: fixture.MovementMock(
				fixture.WithMovementDescription("Movimento atualizado"),
				fixture.WithMovementAmount(-200.0),
				fixture.WithMovementIsPaid(true),
			),
			expectedError: nil,
		},
		"should update paid movement changing wallet with success": {
			id: fixture.MovementID,
			newMovement: fixture.MovementMock(
				fixture.WithMovementDescription("Movimento transferido"),
				fixture.WithMovementAmount(-100.0),
				fixture.WithMovementIsPaid(true),
				fixture.WithMovementWalletID(fixture.FixtureWalletID),
			),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager) {
				existingMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Movimento original"),
					fixture.WithMovementAmount(-100.0),
					fixture.WithMovementIsPaid(true),
				)

				updatedMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Movimento transferido"),
					fixture.WithMovementAmount(-100.0),
					fixture.WithMovementIsPaid(true),
					fixture.WithMovementWalletID(fixture.FixtureWalletID),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(nil)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(existingMovement, nil)
				mockMovRepo.On("UpdateOne", mock.Anything, fixture.MovementID, mock.Anything).Return(updatedMovement, nil)

				// Remover da wallet original: ReverseAmount() = 100.0 (positivo)
				mockWalletRepo.On("FindByID", existingMovement.WalletID).Return(domain.Wallet{
					ID:      existingMovement.WalletID,
					Balance: 1000.0,
				}, nil)
				mockWalletRepo.On("UpdateAmount", mock.Anything, existingMovement.WalletID, 1100.0).Return(nil)

				// Adicionar na nova wallet: amount = -100.0
				mockWalletRepo.On("FindByID", &fixture.FixtureWalletID).Return(domain.Wallet{
					ID:      &fixture.FixtureWalletID,
					Balance: 1000.0,
				}, nil)
				mockWalletRepo.On("UpdateAmount", mock.Anything, &fixture.FixtureWalletID, 900.0).Return(nil)
			},
			expectedMovement: fixture.MovementMock(
				fixture.WithMovementDescription("Movimento transferido"),
				fixture.WithMovementAmount(-100.0),
				fixture.WithMovementIsPaid(true),
				fixture.WithMovementWalletID(fixture.FixtureWalletID),
			),
			expectedError: nil,
		},
		"should return error when subcategory does not belong to category": {
			id: fixture.MovementID,
			newMovement: fixture.MovementMock(
				fixture.WithMovementDescription("Movimento com erro"),
				fixture.WithMovementSubCategoryID(fixture.SubCategoryID),
			),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager) {
				newMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Movimento com erro"),
					fixture.WithMovementSubCategoryID(fixture.SubCategoryID),
				)

				mockSubCat.On("IsSubCategoryBelongsToCategory", fixture.SubCategoryID, *newMovement.CategoryID).Return(false, nil)
			},
			expectedMovement: domain.Movement{},
			expectedError: fmt.Errorf("validate subcategory: %w: %s",
				fmt.Errorf("invalid input"),
				"subcategory does not belong to the provided category",
			),
		},
		"should return error when fails to find movement and recurrent": {
			id: fixture.MovementID,
			newMovement: fixture.MovementMock(
				fixture.WithMovementDescription("Movimento com erro"),
			),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager) {
				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(assert.AnError)

				// Simula erro diferente de "not found"
				mockMovRepo.On("FindByID", fixture.MovementID).Return(domain.Movement{}, assert.AnError)
			},
			expectedMovement: domain.Movement{},
			expectedError:    assert.AnError,
		},
		"should return error when fails to update movement": {
			id: fixture.MovementID,
			newMovement: fixture.MovementMock(
				fixture.WithMovementDescription("Movimento com erro"),
			),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager) {
				existingMovement := fixture.MovementMock()

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(assert.AnError)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(existingMovement, nil)
				mockMovRepo.On("UpdateOne", mock.Anything, fixture.MovementID, mock.Anything).Return(domain.Movement{}, assert.AnError)
			},
			expectedMovement: domain.Movement{},
			expectedError:    assert.AnError,
		},
		"should return error when fails to update wallet balance by amount difference": {
			id: fixture.MovementID,
			newMovement: fixture.MovementMock(
				fixture.WithMovementDescription("Movimento com erro"),
				fixture.WithMovementAmount(-200.0),
				fixture.WithMovementIsPaid(true),
			),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager) {
				existingMovement := fixture.MovementMock(
					fixture.WithMovementAmount(-100.0),
					fixture.WithMovementIsPaid(true),
				)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(existingMovement, nil)

				// Falha na validação de saldo suficiente - saldo insuficiente (10 + (-100) = -90 < 0)
				mockWalletRepo.On("FindByID", existingMovement.WalletID).Return(domain.Wallet{
					ID:      existingMovement.WalletID,
					Balance: 10.0,
				}, nil)

				mockTxManager.On("WithTransaction", mock.Anything).Return(nil)
			},
			expectedMovement: domain.Movement{},
			expectedError:    ErrInsufficientBalance,
		},
		"should create movement from recurrent when movement not found": {
			id: fixture.MovementID,
			newMovement: fixture.MovementMock(
				fixture.WithMovementDescription("Movimento de recorrente"),
				fixture.WithMovementAmount(-150.0),
				fixture.WithMovementIsPaid(false),
				fixture.WithMovementIsRecurrent(true),
				fixture.WithMovementRecurrentID(),
				fixture.WithMovementSubCategoryID(fixture.SubCategoryID),
				fixture.WithMovementDate(time.Date(2023, 9, 15, 10, 0, 0, 0, time.UTC)),
			),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager) {
				createdMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Movimento de recorrente"),
					fixture.WithMovementAmount(-150.0),
					fixture.WithMovementIsPaid(false),
				)

				recurrentMovement := fixture.RecurrentMovementMock()

				mockSubCat.On("IsSubCategoryBelongsToCategory", fixture.SubCategoryID, fixture.CategoryID).Return(true, nil)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(nil)

				// Simulando repository.ErrMovementNotFound
				mockMovRepo.On("FindByID", fixture.MovementID).Return(domain.Movement{}, repository.ErrMovementNotFound)
				// Esta primeira chamada é para o ID passado pelo parâmetro (movimento não existe, busca recorrente)
				mockRecRepo.On("FindByID", fixture.MovementID).Return(recurrentMovement, nil)
				// Esta segunda chamada é para handleRecurrent com RecurrentID
				mockRecRepo.On("FindByID", fixture.RecurrentMovementID).Return(recurrentMovement, nil)
				// Mocks necessários para handleRecurrent
				mockRecRepo.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(recurrentMovement, nil)
				mockRecRepo.On("Add", mock.Anything, mock.Anything, mock.Anything).Return(recurrentMovement, nil)
				mockMovRepo.On("Add", mock.Anything, mock.Anything, mock.Anything).Return(createdMovement, nil)
			},
			expectedMovement: fixture.MovementMock(
				fixture.WithMovementDescription("Movimento de recorrente"),
				fixture.WithMovementAmount(-150.0),
				fixture.WithMovementIsPaid(false),
			),
			expectedError: nil,
		},
		"should handle recurrent movement and update existing movement": {
			id: fixture.MovementID,
			newMovement: fixture.MovementMock(
				fixture.WithMovementDescription("Movimento recorrente atualizado"),
				fixture.WithMovementAmount(-200.0),
				fixture.WithMovementIsRecurrent(true),
				fixture.WithMovementRecurrentID(),
			),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager) {
				existingMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Movimento original"),
					fixture.WithMovementAmount(-100.0),
					fixture.WithMovementIsPaid(false),
					fixture.WithMovementIsRecurrent(true),
				)

				updatedMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Movimento recorrente atualizado"),
					fixture.WithMovementAmount(-200.0),
					fixture.WithMovementIsRecurrent(true),
				)

				recurrentMovement := fixture.RecurrentMovementMock()

				// Mock removido porque este newMovement não tem SubCategoryID

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(nil)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(existingMovement, nil)

				// Mocks para handleRecurrent
				mockRecRepo.On("FindByID", fixture.RecurrentMovementID).Return(recurrentMovement, nil)
				mockRecRepo.On("Update", mock.Anything, &fixture.RecurrentMovementID, mock.Anything).Return(recurrentMovement, nil)
				mockRecRepo.On("Add", mock.Anything, mock.Anything, mock.Anything).Return(recurrentMovement, nil)

				mockMovRepo.On("UpdateOne", mock.Anything, fixture.MovementID, mock.Anything).Return(updatedMovement, nil)
			},
			expectedMovement: fixture.MovementMock(
				fixture.WithMovementDescription("Movimento recorrente atualizado"),
				fixture.WithMovementAmount(-200.0),
				fixture.WithMovementIsRecurrent(true),
			),
			expectedError: nil,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockMovRepo := new(MockMovementRepository)
			mockRecRepo := new(MockRecurrentRepository)
			mockWalletRepo := new(MockWalletRepository)
			mockSubCat := new(MockSubCategory)
			mockTxManager := new(MockTransactionManager)

			if tt.mockSetup != nil {
				tt.mockSetup(mockMovRepo, mockRecRepo, mockWalletRepo, mockSubCat, mockTxManager)
			}

			usecase := NewMovement(
				mockMovRepo,
				mockRecRepo,
				mockWalletRepo,
				mockSubCat,
				new(MockInvoiceRepository),
				new(MockInvoice),
				new(MockCreditCardRepository),
				mockTxManager,
			)

			result, err := usecase.UpdateOne(context.Background(), tt.id, tt.newMovement)

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
