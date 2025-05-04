package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/domain/fixture"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

func TestMovement_Add(t *testing.T) {
	tests := map[string]struct {
		movementInput    domain.Movement
		mockSetup        func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager)
		expectedMovement domain.Movement
		expectedError    error
	}{
		"should add regular movement with success": {
			movementInput: fixture.MovementMock(
				fixture.WithMovementDescription("Compra no supermercado"),
				fixture.AsMovementExpense(50.0),
			),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager) {
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
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager) {
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
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager) {
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
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager) {
				movement := fixture.MovementMock(
					fixture.WithMovementDescription("Movimento com subcategoria inválida"),
					fixture.WithMovementSubCategoryID(fixture.SubCategoryID),
				)

				mockSubCat.On("IsSubCategoryBelongsToCategory", fixture.SubCategoryID, *movement.CategoryID).Return(false, nil)
			},
			expectedMovement: domain.Movement{},
			expectedError:    errors.New("subcategory does not belong to the provided category"),
		},
		"should return error when fails to add movement": {
			movementInput: fixture.MovementMock(
				fixture.WithMovementDescription("Movimento com erro"),
			),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager) {
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
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager) {
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
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager) {
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
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager) {
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
		mockSetup         func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository)
		expectedMovements []domain.Movement
		expectedError     error
	}{
		"should find movements by period with success": {
			periodInput: domain.Period{
				From: time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2025, 5, 31, 23, 59, 59, 0, time.UTC),
			},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository) {
				mockMovRepo.On("FindByPeriod", mock.Anything, mock.Anything).Return([]domain.Movement{
					fixture.MovementMock(fixture.WithMovementDescription("Compra no supermercado")),
				}, nil)

				mockRecRepo.On("FindByMonth", mock.Anything, mock.Anything).Return([]domain.RecurrentMovement{
					fixture.RecurrentMovementMock(fixture.WithRecurrentDescription("Assinatura mensal")),
				}, nil)
			},
			expectedMovements: []domain.Movement{
				fixture.MovementMock(fixture.WithMovementDescription("Compra no supermercado")),
				fixture.MovementMock(fixture.WithMovementDescription("Assinatura mensal")),
			},
			expectedError: nil,
		},
		"should return error when fails to find movements": {
			periodInput: domain.Period{
				From: time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2025, 5, 31, 23, 59, 59, 0, time.UTC),
			},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository) {
				mockMovRepo.On("FindByPeriod", mock.Anything, mock.Anything).Return(nil, errors.New("error to find transactions"))
			},
			expectedMovements: nil,
			expectedError:     errors.New("error to find transactions"),
		},
		"should return error when fails to find recurrent movements": {
			periodInput: domain.Period{
				From: time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2025, 5, 31, 23, 59, 59, 0, time.UTC),
			},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository) {
				mockMovRepo.On("FindByPeriod", mock.Anything, mock.Anything).Return([]domain.Movement{
					fixture.MovementMock(fixture.WithMovementDescription("Compra no supermercado")),
				}, nil)

				mockRecRepo.On("FindByMonth", mock.Anything, mock.Anything).Return(nil, errors.New("error to find recurrents"))
			},
			expectedMovements: nil,
			expectedError:     errors.New("error to find recurrents"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockMovRepo := new(MockMovementRepository)
			mockRecRepo := new(MockRecurrentRepository)

			if tt.mockSetup != nil {
				tt.mockSetup(mockMovRepo, mockRecRepo)
			}

			usecase := NewMovement(
				mockMovRepo,
				mockRecRepo,
				new(MockWalletRepository),
				new(MockSubCategory),
				new(MockTransactionManager),
			)

			result, err := usecase.FindByPeriod(context.Background(), tt.periodInput)

			assert.Equal(t, tt.expectedError, err)
			assert.Equal(t, tt.expectedMovements, result)

			mockMovRepo.AssertExpectations(t)
			mockRecRepo.AssertExpectations(t)
		})
	}
}
