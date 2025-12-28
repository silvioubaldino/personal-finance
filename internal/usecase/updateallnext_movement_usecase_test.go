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
		mockSetup        func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager)
		expectedMovement domain.Movement
		expectedError    error
	}{
		"should update all next from existing movement with success": {
			id: fixture.MovementID,
			newMovement: fixture.MovementMock(
				fixture.WithMovementDescription("Assinatura reajustada"),
				fixture.WithMovementAmount(-150.0),
				fixture.WithMovementIsPaid(false),
				fixture.WithMovementIsRecurrent(true),
				fixture.WithMovementRecurrentID(),
				fixture.WithMovementDate(time.Date(2023, 9, 15, 10, 0, 0, 0, time.UTC)),
			),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager) {
				existingMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Assinatura mensal"),
					fixture.WithMovementAmount(-100.0),
					fixture.WithMovementIsPaid(false),
					fixture.WithMovementIsRecurrent(true),
					fixture.WithMovementRecurrentID(),
				)

				updatedMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Assinatura reajustada"),
					fixture.WithMovementAmount(-150.0),
					fixture.WithMovementIsPaid(false),
					fixture.WithMovementIsRecurrent(true),
				)

				recurrentMovement := fixture.RecurrentMovementMock()

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(nil)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(existingMovement, nil)

				mockRecRepo.On("FindByID", fixture.RecurrentMovementID).Return(recurrentMovement, nil)
				mockRecRepo.On("Update", mock.Anything, &fixture.RecurrentMovementID, mock.Anything).Return(recurrentMovement, nil)
				mockRecRepo.On("Add", mock.Anything, mock.Anything, mock.Anything).Return(recurrentMovement, nil)

				mockMovRepo.On("Update", mock.Anything, fixture.MovementID, mock.Anything).Return(updatedMovement, nil)
			},
			expectedMovement: fixture.MovementMock(
				fixture.WithMovementDescription("Assinatura reajustada"),
				fixture.WithMovementAmount(-150.0),
				fixture.WithMovementIsPaid(false),
				fixture.WithMovementIsRecurrent(true),
			),
			expectedError: nil,
		},
		"should create movement from virtual recurrent and update all next": {
			id: fixture.RecurrentMovementID,
			newMovement: fixture.MovementMock(
				fixture.WithMovementDescription("Assinatura nova"),
				fixture.WithMovementAmount(-200.0),
				fixture.WithMovementIsPaid(false),
				fixture.WithMovementIsRecurrent(true),
				fixture.WithMovementRecurrentID(),
				fixture.WithMovementDate(time.Date(2023, 9, 15, 10, 0, 0, 0, time.UTC)),
			),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager) {
				recurrentMovement := fixture.RecurrentMovementMock()

				createdMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Assinatura nova"),
					fixture.WithMovementAmount(-200.0),
					fixture.WithMovementIsPaid(false),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(nil)

				mockMovRepo.On("FindByID", fixture.RecurrentMovementID).Return(domain.Movement{}, repository.ErrMovementNotFound)
				mockRecRepo.On("FindByID", fixture.RecurrentMovementID).Return(recurrentMovement, nil)

				mockRecRepo.On("Update", mock.Anything, &fixture.RecurrentMovementID, mock.Anything).Return(recurrentMovement, nil)
				mockRecRepo.On("Add", mock.Anything, mock.Anything, mock.Anything).Return(recurrentMovement, nil)

				mockMovRepo.On("Add", mock.Anything, mock.Anything, mock.Anything).Return(createdMovement, nil)
			},
			expectedMovement: fixture.MovementMock(
				fixture.WithMovementDescription("Assinatura nova"),
				fixture.WithMovementAmount(-200.0),
				fixture.WithMovementIsPaid(false),
			),
			expectedError: nil,
		},
		"should update paid movement and adjust wallet balance": {
			id: fixture.MovementID,
			newMovement: fixture.MovementMock(
				fixture.WithMovementDescription("Assinatura reajustada"),
				fixture.WithMovementAmount(-200.0),
				fixture.WithMovementIsPaid(true),
				fixture.WithMovementIsRecurrent(true),
				fixture.WithMovementRecurrentID(),
				fixture.WithMovementDate(time.Date(2023, 9, 15, 10, 0, 0, 0, time.UTC)),
			),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager) {
				existingMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Assinatura mensal"),
					fixture.WithMovementAmount(-100.0),
					fixture.WithMovementIsPaid(true),
					fixture.WithMovementIsRecurrent(true),
				)

				updatedMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Assinatura reajustada"),
					fixture.WithMovementAmount(-200.0),
					fixture.WithMovementIsPaid(true),
				)

				recurrentMovement := fixture.RecurrentMovementMock()

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(nil)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(existingMovement, nil)

				mockRecRepo.On("FindByID", fixture.RecurrentMovementID).Return(recurrentMovement, nil)
				mockRecRepo.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(recurrentMovement, nil)
				mockRecRepo.On("Add", mock.Anything, mock.Anything, mock.Anything).Return(recurrentMovement, nil)

				mockWalletRepo.On("FindByID", existingMovement.WalletID).Return(domain.Wallet{
					ID:      existingMovement.WalletID,
					Balance: 1000.0,
				}, nil)
				mockWalletRepo.On("UpdateAmount", mock.Anything, existingMovement.WalletID, 900.0).Return(nil)

				mockMovRepo.On("Update", mock.Anything, fixture.MovementID, mock.Anything).Return(updatedMovement, nil)
			},
			expectedMovement: fixture.MovementMock(
				fixture.WithMovementDescription("Assinatura reajustada"),
				fixture.WithMovementAmount(-200.0),
				fixture.WithMovementIsPaid(true),
			),
			expectedError: nil,
		},
		"should return error when recurrent_id is missing": {
			id: fixture.MovementID,
			newMovement: fixture.MovementMock(
				fixture.WithMovementDescription("Movimento sem recurrent_id"),
				fixture.WithMovementDate(time.Date(2023, 9, 15, 10, 0, 0, 0, time.UTC)),
			),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager) {
			},
			expectedMovement: domain.Movement{},
			expectedError:    assert.AnError,
		},
		"should return error when date is missing": {
			id: fixture.MovementID,
			newMovement: func() domain.Movement {
				m := fixture.MovementMock(
					fixture.WithMovementDescription("Movimento sem data"),
					fixture.WithMovementRecurrentID(),
				)
				m.Date = nil
				return m
			}(),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager) {
			},
			expectedMovement: domain.Movement{},
			expectedError:    assert.AnError,
		},
		"should return error when movement and recurrent not found": {
			id: fixture.MovementID,
			newMovement: fixture.MovementMock(
				fixture.WithMovementDescription("Movimento inexistente"),
				fixture.WithMovementRecurrentID(),
				fixture.WithMovementDate(time.Date(2023, 9, 15, 10, 0, 0, 0, time.UTC)),
			),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager) {
				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(assert.AnError)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(domain.Movement{}, repository.ErrMovementNotFound)
				mockRecRepo.On("FindByID", fixture.MovementID).Return(domain.RecurrentMovement{}, assert.AnError)
			},
			expectedMovement: domain.Movement{},
			expectedError:    assert.AnError,
		},
		"should return error for credit card movement": {
			id: fixture.MovementID,
			newMovement: fixture.MovementMock(
				fixture.WithMovementDescription("Compra no cartão"),
				fixture.WithMovementRecurrentID(),
				fixture.WithMovementDate(time.Date(2023, 9, 15, 10, 0, 0, 0, time.UTC)),
				fixture.WithMovementTypePayment(string(domain.TypePaymentCreditCard)),
				fixture.WithMovementCreditCardID(&fixture.CreditCardID),
			),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager) {
				existingMovement := fixture.MovementMock(
					fixture.WithMovementTypePayment(string(domain.TypePaymentCreditCard)),
					fixture.WithMovementCreditCardID(&fixture.CreditCardID),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(assert.AnError)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(existingMovement, nil)
			},
			expectedMovement: domain.Movement{},
			expectedError:    assert.AnError,
		},
		"should return error when fails to update recurrent": {
			id: fixture.MovementID,
			newMovement: fixture.MovementMock(
				fixture.WithMovementDescription("Assinatura"),
				fixture.WithMovementRecurrentID(),
				fixture.WithMovementDate(time.Date(2023, 9, 15, 10, 0, 0, 0, time.UTC)),
			),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager) {
				existingMovement := fixture.MovementMock()
				recurrentMovement := fixture.RecurrentMovementMock()

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(assert.AnError)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(existingMovement, nil)
				mockRecRepo.On("FindByID", fixture.RecurrentMovementID).Return(recurrentMovement, nil)
				mockRecRepo.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(domain.RecurrentMovement{}, assert.AnError)
			},
			expectedMovement: domain.Movement{},
			expectedError:    assert.AnError,
		},
		"should return error when fails to create new recurrent": {
			id: fixture.MovementID,
			newMovement: fixture.MovementMock(
				fixture.WithMovementDescription("Assinatura"),
				fixture.WithMovementRecurrentID(),
				fixture.WithMovementDate(time.Date(2023, 9, 15, 10, 0, 0, 0, time.UTC)),
			),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager) {
				existingMovement := fixture.MovementMock()
				recurrentMovement := fixture.RecurrentMovementMock()

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(assert.AnError)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(existingMovement, nil)
				mockRecRepo.On("FindByID", fixture.RecurrentMovementID).Return(recurrentMovement, nil)
				mockRecRepo.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(recurrentMovement, nil)
				mockRecRepo.On("Add", mock.Anything, mock.Anything, mock.Anything).Return(domain.RecurrentMovement{}, assert.AnError)
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

			result, err := usecase.UpdateAllNext(context.Background(), tt.id, tt.newMovement)

			if tt.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedMovement, result)
			}

			mockMovRepo.AssertExpectations(t)
			mockRecRepo.AssertExpectations(t)
			mockWalletRepo.AssertExpectations(t)
			mockSubCat.AssertExpectations(t)
			mockTxManager.AssertExpectations(t)
		})
	}
}

func TestMovement_UpdateAllNext_NewRecurrentHasNewData(t *testing.T) {
	t.Run("should create new recurrent with updated data from newMovement", func(t *testing.T) {
		mockMovRepo := new(MockMovementRepository)
		mockRecRepo := new(MockRecurrentRepository)
		mockWalletRepo := new(MockWalletRepository)
		mockSubCat := new(MockSubCategory)
		mockTxManager := new(MockTransactionManager)

		existingMovement := fixture.MovementMock(
			fixture.WithMovementDescription("Assinatura original"),
			fixture.WithMovementAmount(-100.0),
			fixture.WithMovementIsPaid(false),
		)

		newMovement := fixture.MovementMock(
			fixture.WithMovementDescription("Assinatura REAJUSTADA"),
			fixture.WithMovementAmount(-150.0),
			fixture.WithMovementRecurrentID(),
			fixture.WithMovementDate(time.Date(2023, 9, 15, 10, 0, 0, 0, time.UTC)),
		)

		originalRecurrent := fixture.RecurrentMovementMock(
			fixture.WithRecurrentMovementDescription("Assinatura original"),
			fixture.WithRecurrentMovementAmount(-100.0),
		)

		var capturedNewRecurrent domain.RecurrentMovement

		mockTxManager.On("WithTransaction", mock.Anything).
			Run(func(args mock.Arguments) {
				fn := args.Get(0).(func(*gorm.DB) error)
				_ = fn(nil)
			}).Return(nil)

		mockMovRepo.On("FindByID", fixture.MovementID).Return(existingMovement, nil)
		mockRecRepo.On("FindByID", fixture.RecurrentMovementID).Return(originalRecurrent, nil)

		mockRecRepo.On("Update", mock.Anything, &fixture.RecurrentMovementID, mock.Anything).Return(originalRecurrent, nil)

		mockRecRepo.On("Add", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				capturedNewRecurrent = args.Get(1).(domain.RecurrentMovement)
			}).Return(originalRecurrent, nil)

		mockMovRepo.On("Update", mock.Anything, fixture.MovementID, mock.Anything).Return(existingMovement, nil)

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

		_, err := usecase.UpdateAllNext(context.Background(), fixture.MovementID, newMovement)

		assert.NoError(t, err)
		assert.Equal(t, "Assinatura REAJUSTADA", capturedNewRecurrent.Description)
		assert.Equal(t, -150.0, capturedNewRecurrent.Amount)

		mockRecRepo.AssertExpectations(t)
	})
}

// Teste para corner case: preservar EndDate original quando recorrência tem limite definido
func TestMovement_UpdateAllNext_PreservesOriginalEndDate(t *testing.T) {
	t.Run("should preserve original EndDate when recurrent has defined end", func(t *testing.T) {
		mockMovRepo := new(MockMovementRepository)
		mockRecRepo := new(MockRecurrentRepository)
		mockWalletRepo := new(MockWalletRepository)
		mockSubCat := new(MockSubCategory)
		mockTxManager := new(MockTransactionManager)

		// Recorrência com EndDate definido (ex: termina em julho/2023)
		// InitialDate deve ser ANTES de T-1 (Janeiro/2023) para usar fluxo de Update
		originalInitialDate := time.Date(2022, 11, 15, 10, 0, 0, 0, time.UTC) // Novembro/2022
		originalEndDate := time.Date(2023, 7, 15, 10, 0, 0, 0, time.UTC)
		originalRecurrent := fixture.RecurrentMovementMock(
			fixture.WithRecurrentMovementDescription("Assinatura com fim"),
			fixture.WithRecurrentMovementAmount(-100.0),
			fixture.WithRecurrentMovementInitialDate(originalInitialDate),
			fixture.WithRecurrentMovementEndDate(originalEndDate),
		)

		existingMovement := fixture.MovementMock(
			fixture.WithMovementDescription("Assinatura"),
			fixture.WithMovementAmount(-100.0),
			fixture.WithMovementIsPaid(false),
			fixture.WithMovementRecurrentID(),
			fixture.WithMovementDate(time.Date(2023, 2, 15, 10, 0, 0, 0, time.UTC)), // Fevereiro
		)

		newMovement := fixture.MovementMock(
			fixture.WithMovementDescription("Assinatura atualizada"),
			fixture.WithMovementAmount(-150.0),
			fixture.WithMovementRecurrentID(),
			fixture.WithMovementDate(time.Date(2023, 2, 15, 10, 0, 0, 0, time.UTC)),
		)

		var capturedNewRecurrent domain.RecurrentMovement

		mockTxManager.On("WithTransaction", mock.Anything).
			Run(func(args mock.Arguments) {
				fn := args.Get(0).(func(*gorm.DB) error)
				_ = fn(nil)
			}).Return(nil)

		mockMovRepo.On("FindByID", fixture.MovementID).Return(existingMovement, nil)
		mockRecRepo.On("FindByID", fixture.RecurrentMovementID).Return(originalRecurrent, nil)
		mockRecRepo.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(originalRecurrent, nil)
		mockRecRepo.On("Add", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				capturedNewRecurrent = args.Get(1).(domain.RecurrentMovement)
			}).Return(originalRecurrent, nil)
		mockMovRepo.On("Update", mock.Anything, fixture.MovementID, mock.Anything).Return(existingMovement, nil)

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

		_, err := usecase.UpdateAllNext(context.Background(), fixture.MovementID, newMovement)

		assert.NoError(t, err)

		// CRÍTICO: Nova recorrência deve preservar o EndDate original (julho/2023)
		assert.NotNil(t, capturedNewRecurrent.EndDate, "New recurrent should preserve EndDate")
		assert.Equal(t, originalEndDate.Month(), capturedNewRecurrent.EndDate.Month(), "EndDate month should match original")
		assert.Equal(t, originalEndDate.Year(), capturedNewRecurrent.EndDate.Year(), "EndDate year should match original")

		// Nova recorrência começa em março (T+1 de fevereiro)
		assert.Equal(t, time.March, capturedNewRecurrent.InitialDate.Month(), "New recurrent should start in T+1")

		mockRecRepo.AssertExpectations(t)
	})
}

// Teste para corner case: não criar nova recorrência quando update é no último mês
func TestMovement_UpdateAllNext_DoesNotCreateNewRecurrentWhenUpdateIsOnLastMonth(t *testing.T) {
	t.Run("should not create new recurrent when update is on the last month of recurrence", func(t *testing.T) {
		mockMovRepo := new(MockMovementRepository)
		mockRecRepo := new(MockRecurrentRepository)
		mockWalletRepo := new(MockWalletRepository)
		mockSubCat := new(MockSubCategory)
		mockTxManager := new(MockTransactionManager)

		// Recorrência termina em Julho/2023
		endDate := time.Date(2023, 7, 15, 10, 0, 0, 0, time.UTC)
		recurrent := fixture.RecurrentMovementMock(
			fixture.WithRecurrentMovementDescription("Assinatura que termina em julho"),
			fixture.WithRecurrentMovementAmount(-100.0),
			fixture.WithRecurrentMovementEndDate(endDate),
		)

		// Movement existente no mês de julho (último mês)
		existingMovement := fixture.MovementMock(
			fixture.WithMovementDescription("Assinatura"),
			fixture.WithMovementAmount(-100.0),
			fixture.WithMovementIsPaid(false),
			fixture.WithMovementRecurrentID(),
			fixture.WithMovementIsRecurrent(true),
			fixture.WithMovementDate(time.Date(2023, 7, 15, 10, 0, 0, 0, time.UTC)),
		)

		newMovement := fixture.MovementMock(
			fixture.WithMovementDescription("Assinatura atualizada para todas"),
			fixture.WithMovementAmount(-150.0),
			fixture.WithMovementRecurrentID(),
			fixture.WithMovementIsRecurrent(true),
			fixture.WithMovementDate(time.Date(2023, 7, 15, 10, 0, 0, 0, time.UTC)),
		)

		mockTxManager.On("WithTransaction", mock.Anything).
			Run(func(args mock.Arguments) {
				fn := args.Get(0).(func(*gorm.DB) error)
				_ = fn(nil)
			}).Return(nil)

		mockMovRepo.On("FindByID", fixture.MovementID).Return(existingMovement, nil)
		mockRecRepo.On("FindByID", fixture.RecurrentMovementID).Return(recurrent, nil)
		mockRecRepo.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(recurrent, nil)
		mockMovRepo.On("Update", mock.Anything, fixture.MovementID, mock.Anything).Return(existingMovement, nil)
		// NÃO deve chamar Add - essa é a validação crítica

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

		_, err := usecase.UpdateAllNext(context.Background(), fixture.MovementID, newMovement)

		assert.NoError(t, err)

		// Verifica que Add NÃO foi chamado
		mockRecRepo.AssertNotCalled(t, "Add", mock.Anything, mock.Anything)
		mockRecRepo.AssertExpectations(t)
	})
}
