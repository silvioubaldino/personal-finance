package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"

	"personal-finance/internal/domain"
)

func TestMovement_Add(t *testing.T) {
	tests := map[string]struct {
		movementInput    domain.Movement
		mockSetup        func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager)
		expectedMovement domain.Movement
		expectedError    error
	}{
		"should add regular movement with success": {
			movementInput: domain.MovementMock(
				domain.WithMovementDescription("Compra no supermercado"),
				domain.AsMovementExpense(50.0),
			),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager) {
				movement := domain.MovementMock(
					domain.WithMovementDescription("Compra no supermercado"),
					domain.AsMovementExpense(50.0),
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
				mockWalletRepo.On("AddConsistent", mock.Anything, updatedWallet).Return(updatedWallet, nil)
			},
			expectedMovement: domain.MovementMock(
				domain.WithMovementDescription("Compra no supermercado"),
				domain.AsMovementExpense(50.0),
			),
			expectedError: nil,
		},
		"should add unpaid movement with success": {
			movementInput: domain.MovementMock(
				domain.WithMovementDescription("Compra parcelada"),
				domain.AsMovementExpense(200.0),
				domain.AsMovementUnpaid(),
			),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager) {
				movement := domain.MovementMock(
					domain.WithMovementDescription("Compra parcelada"),
					domain.AsMovementExpense(200.0),
					domain.AsMovementUnpaid(),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(nil)

				mockMovRepo.On("Add", mock.Anything, movement).Return(movement, nil)
			},
			expectedMovement: domain.MovementMock(
				domain.WithMovementDescription("Compra parcelada"),
				domain.AsMovementExpense(200.0),
				domain.AsMovementUnpaid(),
			),
			expectedError: nil,
		},
		"should add recurrent movement with success": {
			movementInput: domain.MovementMock(
				domain.WithMovementDescription("Assinatura mensal"),
				domain.AsMovementExpense(30.0),
				domain.AsMovementRecurrent(),
			),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager) {
				movementWithoutRecurrentID := domain.MovementMock(
					domain.WithMovementDescription("Assinatura mensal"),
					domain.AsMovementExpense(30.0),
					domain.AsMovementRecurrent(),
				)

				movementWithRecurrentID := domain.MovementMock(
					domain.WithMovementDescription("Assinatura mensal"),
					domain.AsMovementExpense(30.0),
					domain.AsMovementRecurrent(),
					domain.WithMovementRecurrentID(),
				)

				recurrent := domain.ToRecurrentMovement(movementWithoutRecurrentID)
				createdRecurrent := recurrent
				createdRecurrent.ID = &domain.RecurrentID

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
				mockWalletRepo.On("AddConsistent", mock.Anything, updatedWallet).Return(updatedWallet, nil)
			},
			expectedMovement: domain.MovementMock(
				domain.WithMovementDescription("Assinatura mensal"),
				domain.AsMovementExpense(30.0),
				domain.AsMovementRecurrent(),
				domain.WithMovementRecurrentID(),
			),
			expectedError: nil,
		},
		"should return error when subcategory does not belong to category": {
			movementInput: domain.MovementMock(
				domain.WithMovementDescription("Movimento com subcategoria inválida"),
				domain.WithMovementSubCategoryID(domain.SubCategoryID),
			),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager) {
				movement := domain.MovementMock(
					domain.WithMovementDescription("Movimento com subcategoria inválida"),
					domain.WithMovementSubCategoryID(domain.SubCategoryID),
				)

				mockSubCat.On("IsSubCategoryBelongsToCategory", domain.SubCategoryID, *movement.CategoryID).Return(false, nil)
			},
			expectedMovement: domain.Movement{},
			expectedError:    errors.New("subcategoria não pertence à categoria informada"),
		},
		"should return error when fails to add movement": {
			movementInput: domain.MovementMock(
				domain.WithMovementDescription("Movimento com erro"),
			),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager) {
				movement := domain.MovementMock(
					domain.WithMovementDescription("Movimento com erro"),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(errors.New("erro ao adicionar movimento"))

				mockMovRepo.On("Add", mock.Anything, movement).Return(domain.Movement{}, errors.New("erro ao adicionar movimento"))
			},
			expectedMovement: domain.Movement{},
			expectedError:    errors.New("erro ao adicionar movimento"),
		},
		"should return error when fails to find wallet": {
			movementInput: domain.MovementMock(
				domain.WithMovementDescription("Movimento com erro na carteira"),
				domain.AsMovementExpense(100.0),
			),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager) {
				movement := domain.MovementMock(
					domain.WithMovementDescription("Movimento com erro na carteira"),
					domain.AsMovementExpense(100.0),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(errors.New("erro ao buscar carteira"))

				mockMovRepo.On("Add", mock.Anything, movement).Return(movement, nil)

				mockWalletRepo.On("FindByID", movement.WalletID).Return(domain.Wallet{}, errors.New("erro ao buscar carteira"))
			},
			expectedMovement: domain.Movement{},
			expectedError:    errors.New("erro ao buscar carteira"),
		},
		"should return error when fails to update wallet balance": {
			movementInput: domain.MovementMock(
				domain.WithMovementDescription("Movimento com erro na atualização da carteira"),
				domain.AsMovementExpense(150.0),
			),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager) {
				movement := domain.MovementMock(
					domain.WithMovementDescription("Movimento com erro na atualização da carteira"),
					domain.AsMovementExpense(150.0),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(errors.New("erro ao atualizar carteira"))

				mockMovRepo.On("Add", mock.Anything, movement).Return(movement, nil)

				mockWalletRepo.On("FindByID", movement.WalletID).Return(domain.Wallet{
					ID:      movement.WalletID,
					Balance: 1000.0,
				}, nil)

				updatedWallet := domain.Wallet{
					ID:      movement.WalletID,
					Balance: 850.0,
				}
				mockWalletRepo.On("AddConsistent", mock.Anything, updatedWallet).Return(domain.Wallet{}, errors.New("erro ao atualizar carteira"))
			},
			expectedMovement: domain.Movement{},
			expectedError:    errors.New("erro ao atualizar carteira"),
		},
		"should return error when fails to create recurrence": {
			movementInput: domain.MovementMock(
				domain.WithMovementDescription("Movimento recorrente com erro"),
				domain.AsMovementExpense(200.0),
				domain.AsMovementRecurrent(),
			),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager) {
				movement := domain.MovementMock(
					domain.WithMovementDescription("Movimento recorrente com erro"),
					domain.AsMovementExpense(200.0),
					domain.AsMovementRecurrent(),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(errors.New("erro ao criar recorrência"))

				recurrent := domain.ToRecurrentMovement(movement)

				mockRecRepo.On("Add", mock.Anything, recurrent).Return(domain.RecurrentMovement{}, errors.New("erro ao criar recorrência"))
			},
			expectedMovement: domain.Movement{},
			expectedError:    errors.New("erro ao criar recorrência"),
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
