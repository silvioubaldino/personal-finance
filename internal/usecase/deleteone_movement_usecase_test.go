package usecase

import (
	"context"
	"errors"
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

var errRecurrentNotFound = errors.New("recurrent not found")

func TestMovement_DeleteOne(t *testing.T) {
	tests := map[string]struct {
		id            uuid.UUID
		targetDate    time.Time
		mockSetup     func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager)
		expectedError error
	}{
		"should delete existing non-recurrent movement with success": {
			id:         fixture.MovementID,
			targetDate: time.Date(2023, 9, 15, 0, 0, 0, 0, time.UTC),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				existingMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Compra única"),
					fixture.WithMovementAmount(-100.0),
					fixture.WithMovementIsPaid(false),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(nil)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(existingMovement, nil)
				mockMovRepo.On("Delete", mock.Anything, mock.Anything).Return(nil)
			},
			expectedError: nil,
		},
		"should delete existing non-recurrent paid movement and revert wallet balance": {
			id:         fixture.MovementID,
			targetDate: time.Date(2023, 9, 15, 0, 0, 0, 0, time.UTC),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				existingMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Compra paga"),
					fixture.WithMovementAmount(-150.0),
					fixture.WithMovementIsPaid(true),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(nil)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(existingMovement, nil)
				mockWalletRepo.On("UpdateAmount", mock.Anything, mock.Anything, float64(150.0)).Return(nil)
				mockMovRepo.On("Delete", mock.Anything, mock.Anything).Return(nil)
			},
			expectedError: nil,
		},
		"should delete recurrent movement and split series": {
			id:         fixture.MovementID,
			targetDate: time.Date(2023, 9, 15, 0, 0, 0, 0, time.UTC),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				recurrentID := fixture.RecurrentID
				movementDate := time.Date(2023, 9, 15, 0, 0, 0, 0, time.UTC)
				existingMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Assinatura mensal"),
					fixture.WithMovementAmount(-50.0),
					fixture.WithMovementIsPaid(false),
					fixture.WithMovementDate(movementDate),
				)
				existingMovement.RecurrentID = &recurrentID

				originalRecurrent := fixture.RecurrentMovementMock(
					fixture.WithRecurrentMovementInitialDate(time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)),
					fixture.WithoutRecurrentMovementEndDate(),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(nil)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(existingMovement, nil)
				mockRecRepo.On("FindByID", recurrentID).Return(originalRecurrent, nil)
				mockRecRepo.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(originalRecurrent, nil)
				mockRecRepo.On("Add", mock.Anything, mock.Anything).Return(originalRecurrent, nil)
				mockMovRepo.On("Delete", mock.Anything, mock.Anything).Return(nil)
			},
			expectedError: nil,
		},
		"should delete virtual recurrent movement without deleting from DB": {
			id:         fixture.RecurrentID, // ID igual ao RecurrentID = virtual
			targetDate: time.Date(2023, 9, 15, 0, 0, 0, 0, time.UTC),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				originalRecurrent := fixture.RecurrentMovementMock(
					fixture.WithRecurrentMovementInitialDate(time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)),
					fixture.WithoutRecurrentMovementEndDate(),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(nil)

				// Movement não encontrada pelo ID = é virtual
				mockMovRepo.On("FindByID", fixture.RecurrentID).Return(domain.Movement{}, repository.ErrMovementNotFound)
				mockRecRepo.On("FindByID", fixture.RecurrentID).Return(originalRecurrent, nil)
				mockRecRepo.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(originalRecurrent, nil)
				mockRecRepo.On("Add", mock.Anything, mock.Anything).Return(originalRecurrent, nil)
				// Não deve chamar Delete no movimento, pois é virtual
			},
			expectedError: nil,
		},
		"should return error when movement and recurrent not found": {
			id:         fixture.MovementID,
			targetDate: time.Date(2023, 9, 15, 0, 0, 0, 0, time.UTC),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(assert.AnError)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(domain.Movement{}, repository.ErrMovementNotFound)
				mockRecRepo.On("FindByID", fixture.MovementID).Return(domain.RecurrentMovement{}, errRecurrentNotFound)
			},
			expectedError: assert.AnError,
		},
		"should return error when update recurrent fails": {
			id:         fixture.MovementID,
			targetDate: time.Date(2023, 9, 15, 0, 0, 0, 0, time.UTC),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				recurrentID := fixture.RecurrentID
				movementDate := time.Date(2023, 9, 15, 0, 0, 0, 0, time.UTC)
				existingMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Assinatura"),
					fixture.WithMovementAmount(-50.0),
					fixture.WithMovementIsPaid(false),
					fixture.WithMovementDate(movementDate),
				)
				existingMovement.RecurrentID = &recurrentID

				originalRecurrent := fixture.RecurrentMovementMock(
					fixture.WithRecurrentMovementInitialDate(time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)),
					fixture.WithoutRecurrentMovementEndDate(),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(assert.AnError)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(existingMovement, nil)
				mockRecRepo.On("FindByID", recurrentID).Return(originalRecurrent, nil)
				mockRecRepo.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(domain.RecurrentMovement{}, assert.AnError)
			},
			expectedError: assert.AnError,
		},
		"should return error when wallet update fails on paid movement": {
			id:         fixture.MovementID,
			targetDate: time.Date(2023, 9, 15, 0, 0, 0, 0, time.UTC),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				existingMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Compra paga"),
					fixture.WithMovementAmount(-100.0),
					fixture.WithMovementIsPaid(true),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(assert.AnError)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(existingMovement, nil)
				mockWalletRepo.On("UpdateAmount", mock.Anything, mock.Anything, float64(100.0)).Return(assert.AnError)
			},
			expectedError: assert.AnError,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockMovRepo := new(MockMovementRepository)
			mockRecRepo := new(MockRecurrentRepository)
			mockWalletRepo := new(MockWalletRepository)
			mockSubCat := new(MockSubCategory)
			mockTxManager := new(MockTransactionManager)

			tt.mockSetup(mockMovRepo, mockRecRepo, mockWalletRepo, mockTxManager)

			uc := NewMovement(
				mockMovRepo,
				mockRecRepo,
				mockWalletRepo,
				mockSubCat,
				new(MockInvoiceRepository),
				new(MockInvoice),
				new(MockCreditCardRepository),
				mockTxManager,
			)

			err := uc.DeleteOne(context.Background(), tt.id, tt.targetDate)

			if tt.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockMovRepo.AssertExpectations(t)
			mockRecRepo.AssertExpectations(t)
			mockWalletRepo.AssertExpectations(t)
			mockTxManager.AssertExpectations(t)
		})
	}
}

func TestMovement_DeleteOne_SplitRecurrentCreatesNewWithOriginalData(t *testing.T) {
	t.Run("should create new recurrent with original data (not modified)", func(t *testing.T) {
		mockMovRepo := new(MockMovementRepository)
		mockRecRepo := new(MockRecurrentRepository)
		mockWalletRepo := new(MockWalletRepository)
		mockSubCat := new(MockSubCategory)
		mockTxManager := new(MockTransactionManager)

		originalAmount := float64(-100.0)
		originalDescription := "Assinatura original"

		originalRecurrent := fixture.RecurrentMovementMock(
			fixture.WithRecurrentMovementAmount(originalAmount),
			fixture.WithRecurrentMovementDescription(originalDescription),
		)

		recurrentID := *originalRecurrent.ID
		existingMovement := fixture.MovementMock(
			fixture.WithMovementDescription("Assinatura"),
			fixture.WithMovementAmount(-100.0),
			fixture.WithMovementIsPaid(false),
			fixture.WithMovementDate(time.Date(2023, 9, 15, 0, 0, 0, 0, time.UTC)),
		)
		existingMovement.RecurrentID = &recurrentID

		var capturedNewRecurrent domain.RecurrentMovement

		mockTxManager.On("WithTransaction", mock.Anything).
			Run(func(args mock.Arguments) {
				fn := args.Get(0).(func(*gorm.DB) error)
				_ = fn(nil)
			}).Return(nil)

		mockMovRepo.On("FindByID", fixture.MovementID).Return(existingMovement, nil)
		mockRecRepo.On("FindByID", recurrentID).Return(originalRecurrent, nil)
		mockRecRepo.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(originalRecurrent, nil)
		mockRecRepo.On("Add", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				capturedNewRecurrent = args.Get(1).(domain.RecurrentMovement)
			}).Return(originalRecurrent, nil)
		mockMovRepo.On("Delete", mock.Anything, mock.Anything).Return(nil)

		uc := NewMovement(
			mockMovRepo,
			mockRecRepo,
			mockWalletRepo,
			mockSubCat,
			new(MockInvoiceRepository),
			new(MockInvoice),
			new(MockCreditCardRepository),
			mockTxManager,
		)

		err := uc.DeleteOne(context.Background(), fixture.MovementID, time.Date(2023, 9, 15, 0, 0, 0, 0, time.UTC))

		assert.NoError(t, err)

		// Valida que a nova recorrência mantém os dados ORIGINAIS
		assert.Equal(t, originalAmount, capturedNewRecurrent.Amount, "New recurrent should have ORIGINAL amount")
		assert.Equal(t, originalDescription, capturedNewRecurrent.Description, "New recurrent should have ORIGINAL description")

		// Valida que a nova recorrência começa em outubro (T+1 = setembro + 1)
		assert.Equal(t, time.October, capturedNewRecurrent.InitialDate.Month(), "New recurrent should start in month T+1")

		mockMovRepo.AssertExpectations(t)
		mockRecRepo.AssertExpectations(t)
	})
}

// Teste para corner case: preservar EndDate original quando recorrência tem limite definido
func TestMovement_DeleteOne_PreservesOriginalEndDate(t *testing.T) {
	t.Run("should preserve original EndDate when recurrent has defined end", func(t *testing.T) {
		mockMovRepo := new(MockMovementRepository)
		mockRecRepo := new(MockRecurrentRepository)
		mockWalletRepo := new(MockWalletRepository)
		mockSubCat := new(MockSubCategory)
		mockTxManager := new(MockTransactionManager)

		// Recorrência com EndDate definido (ex: termina em julho/2023)
		// InitialDate deve ser ANTES de T-1 (Janeiro/2023) para entrar no fluxo de Update
		originalInitialDate := time.Date(2022, 11, 15, 10, 0, 0, 0, time.UTC) // Novembro/2022
		originalEndDate := time.Date(2023, 7, 15, 10, 0, 0, 0, time.UTC)
		originalRecurrent := fixture.RecurrentMovementMock(
			fixture.WithRecurrentMovementDescription("Assinatura com fim"),
			fixture.WithRecurrentMovementAmount(-100.0),
			fixture.WithRecurrentMovementInitialDate(originalInitialDate),
			fixture.WithRecurrentMovementEndDate(originalEndDate),
		)

		recurrentID := *originalRecurrent.ID
		existingMovement := fixture.MovementMock(
			fixture.WithMovementDescription("Assinatura"),
			fixture.WithMovementAmount(-100.0),
			fixture.WithMovementIsPaid(false),
			fixture.WithMovementDate(time.Date(2023, 2, 15, 0, 0, 0, 0, time.UTC)), // Fevereiro
		)
		existingMovement.RecurrentID = &recurrentID

		var capturedNewRecurrent domain.RecurrentMovement

		mockTxManager.On("WithTransaction", mock.Anything).
			Run(func(args mock.Arguments) {
				fn := args.Get(0).(func(*gorm.DB) error)
				_ = fn(nil)
			}).Return(nil)

		mockMovRepo.On("FindByID", fixture.MovementID).Return(existingMovement, nil)
		mockRecRepo.On("FindByID", recurrentID).Return(originalRecurrent, nil)
		mockRecRepo.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(originalRecurrent, nil)
		mockRecRepo.On("Add", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				capturedNewRecurrent = args.Get(1).(domain.RecurrentMovement)
			}).Return(originalRecurrent, nil)
		mockMovRepo.On("Delete", mock.Anything, mock.Anything).Return(nil)

		uc := NewMovement(
			mockMovRepo,
			mockRecRepo,
			mockWalletRepo,
			mockSubCat,
			new(MockInvoiceRepository),
			new(MockInvoice),
			new(MockCreditCardRepository),
			mockTxManager,
		)

		err := uc.DeleteOne(context.Background(), fixture.MovementID, time.Date(2023, 2, 15, 0, 0, 0, 0, time.UTC))

		assert.NoError(t, err)

		// CRÍTICO: Nova recorrência deve preservar o EndDate original (julho/2023)
		assert.NotNil(t, capturedNewRecurrent.EndDate, "New recurrent should preserve EndDate")
		assert.Equal(t, originalEndDate.Month(), capturedNewRecurrent.EndDate.Month(), "EndDate month should match original")
		assert.Equal(t, originalEndDate.Year(), capturedNewRecurrent.EndDate.Year(), "EndDate year should match original")

		// Nova recorrência começa em março (T+1 de fevereiro)
		assert.Equal(t, time.March, capturedNewRecurrent.InitialDate.Month(), "New recurrent should start in T+1")

		mockMovRepo.AssertExpectations(t)
		mockRecRepo.AssertExpectations(t)
	})
}

// Teste para corner case: deletar recorrência quando delete é no primeiro mês (evita "zumbi")
func TestMovement_DeleteOne_DeletesRecurrentWhenDeleteIsOnFirstMonth(t *testing.T) {
	t.Run("should delete and recreate recurrent when delete is on the first month of recurrence", func(t *testing.T) {
		mockMovRepo := new(MockMovementRepository)
		mockRecRepo := new(MockRecurrentRepository)
		mockWalletRepo := new(MockWalletRepository)
		mockSubCat := new(MockSubCategory)
		mockTxManager := new(MockTransactionManager)

		// Recorrência INFINITA que começa em Dezembro/2025 (mesmo mês do delete)
		initialDate := time.Date(2025, 12, 26, 0, 0, 0, 0, time.UTC)
		recurrent := fixture.RecurrentMovementMock(
			fixture.WithRecurrentMovementDescription("Nova assinatura"),
			fixture.WithRecurrentMovementAmount(-100.0),
			fixture.WithRecurrentMovementInitialDate(initialDate),
			fixture.WithoutRecurrentMovementEndDate(), // Recorrência infinita
		)

		// Delete no primeiro mês
		targetDate := time.Date(2025, 12, 26, 0, 0, 0, 0, time.UTC)

		mockTxManager.On("WithTransaction", mock.Anything).
			Run(func(args mock.Arguments) {
				fn := args.Get(0).(func(*gorm.DB) error)
				_ = fn(nil)
			}).Return(nil)

		// Movement não encontrada = virtual
		mockMovRepo.On("FindByID", fixture.RecurrentMovementID).Return(domain.Movement{}, repository.ErrMovementNotFound)
		mockRecRepo.On("FindByID", fixture.RecurrentMovementID).Return(recurrent, nil)
		// Deve DELETAR a recorrência original (evita zumbi)
		mockRecRepo.On("Delete", mock.Anything, fixture.RecurrentMovementID).Return(nil)
		// Deve CRIAR nova recorrência começando em T+1 (Janeiro/2026)
		mockRecRepo.On("Add", mock.Anything, mock.Anything).Return(recurrent, nil)

		uc := NewMovement(
			mockMovRepo,
			mockRecRepo,
			mockWalletRepo,
			mockSubCat,
			new(MockInvoiceRepository),
			new(MockInvoice),
			new(MockCreditCardRepository),
			mockTxManager,
		)

		err := uc.DeleteOne(context.Background(), fixture.RecurrentMovementID, targetDate)

		assert.NoError(t, err)

		// Verifica que Delete foi chamado (não Update)
		mockRecRepo.AssertCalled(t, "Delete", mock.Anything, fixture.RecurrentMovementID)
		mockRecRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything)
		mockRecRepo.AssertExpectations(t)
	})
}

// Teste para corner case: deletar recorrência sem criar nova quando delete é no primeiro E único mês
func TestMovement_DeleteOne_OnlyDeletesWhenDeleteIsOnFirstAndOnlyMonth(t *testing.T) {
	t.Run("should only delete recurrent when delete is on first and only month", func(t *testing.T) {
		mockMovRepo := new(MockMovementRepository)
		mockRecRepo := new(MockRecurrentRepository)
		mockWalletRepo := new(MockWalletRepository)
		mockSubCat := new(MockSubCategory)
		mockTxManager := new(MockTransactionManager)

		// Recorrência que começa E termina em Dezembro/2025 (único mês)
		initialDate := time.Date(2025, 12, 26, 0, 0, 0, 0, time.UTC)
		recurrent := fixture.RecurrentMovementMock(
			fixture.WithRecurrentMovementDescription("Assinatura de 1 mês"),
			fixture.WithRecurrentMovementAmount(-100.0),
			fixture.WithRecurrentMovementInitialDate(initialDate),
			fixture.WithRecurrentMovementEndDate(initialDate), // Mesmo mês
		)

		// Delete no primeiro (e único) mês
		targetDate := time.Date(2025, 12, 26, 0, 0, 0, 0, time.UTC)

		mockTxManager.On("WithTransaction", mock.Anything).
			Run(func(args mock.Arguments) {
				fn := args.Get(0).(func(*gorm.DB) error)
				_ = fn(nil)
			}).Return(nil)

		mockMovRepo.On("FindByID", fixture.RecurrentMovementID).Return(domain.Movement{}, repository.ErrMovementNotFound)
		mockRecRepo.On("FindByID", fixture.RecurrentMovementID).Return(recurrent, nil)
		mockRecRepo.On("Delete", mock.Anything, fixture.RecurrentMovementID).Return(nil)
		// NÃO deve chamar Add - é o único mês

		uc := NewMovement(
			mockMovRepo,
			mockRecRepo,
			mockWalletRepo,
			mockSubCat,
			new(MockInvoiceRepository),
			new(MockInvoice),
			new(MockCreditCardRepository),
			mockTxManager,
		)

		err := uc.DeleteOne(context.Background(), fixture.RecurrentMovementID, targetDate)

		assert.NoError(t, err)

		mockRecRepo.AssertCalled(t, "Delete", mock.Anything, fixture.RecurrentMovementID)
		mockRecRepo.AssertNotCalled(t, "Add", mock.Anything, mock.Anything)
		mockRecRepo.AssertExpectations(t)
	})
}

// Teste para corner case: não criar nova recorrência quando delete é no último mês
func TestMovement_DeleteOne_DoesNotCreateNewRecurrentWhenDeleteIsOnLastMonth(t *testing.T) {
	t.Run("should not create new recurrent when delete is on the last month of recurrence", func(t *testing.T) {
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

		// Movement virtual no mês de julho (último mês)
		targetDate := time.Date(2023, 7, 15, 0, 0, 0, 0, time.UTC)

		mockTxManager.On("WithTransaction", mock.Anything).
			Run(func(args mock.Arguments) {
				fn := args.Get(0).(func(*gorm.DB) error)
				_ = fn(nil)
			}).Return(nil)

		// Movement não encontrada = virtual
		mockMovRepo.On("FindByID", fixture.RecurrentMovementID).Return(domain.Movement{}, repository.ErrMovementNotFound)
		mockRecRepo.On("FindByID", fixture.RecurrentMovementID).Return(recurrent, nil)
		mockRecRepo.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(recurrent, nil)
		// NÃO deve chamar Add - essa é a validação crítica

		uc := NewMovement(
			mockMovRepo,
			mockRecRepo,
			mockWalletRepo,
			mockSubCat,
			new(MockInvoiceRepository),
			new(MockInvoice),
			new(MockCreditCardRepository),
			mockTxManager,
		)

		err := uc.DeleteOne(context.Background(), fixture.RecurrentMovementID, targetDate)

		assert.NoError(t, err)

		// Verifica que Add NÃO foi chamado
		mockRecRepo.AssertNotCalled(t, "Add", mock.Anything, mock.Anything)
		mockRecRepo.AssertExpectations(t)
	})
}
