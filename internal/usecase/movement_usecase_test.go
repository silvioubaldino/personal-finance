package usecase

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/domain/fixture"

	"github.com/google/uuid"
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
			expectedError: fmt.Errorf("validate subcategory: %w: %s",
				fmt.Errorf("invalid input"),
				"subcategory does not belong to the provided category",
			),
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
		expectedMovements domain.MovementList
		expectedError     error
	}{
		"should find only non-recurrent movement with success": {
			periodInput: domain.Period{
				From: time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2025, 5, 31, 23, 59, 59, 0, time.UTC),
			},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository) {
				mockMovRepo.On("FindByPeriod", mock.Anything, mock.Anything).Return(domain.MovementList{
					fixture.MovementMock(fixture.WithMovementDescription("Compra no supermercado")),
				}, nil)

				mockRecRepo.On("FindByMonth", mock.Anything, mock.Anything).Return([]domain.RecurrentMovement{}, nil)
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
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository) {
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
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository) {
				movement := fixture.MovementMock(
					fixture.WithMovementDescription("Compra no supermercado"),
				)

				recurrent := fixture.RecurrentMovementMock(
					fixture.WithRecurrentMovementDescription("Assinatura mensal"),
				)

				mockMovRepo.On("FindByPeriod", mock.Anything, mock.Anything).Return(domain.MovementList{movement}, nil)
				mockRecRepo.On("FindByMonth", mock.Anything, mock.Anything).Return([]domain.RecurrentMovement{recurrent}, nil)
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
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository) {
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
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository) {
				mockMovRepo.On("FindByPeriod", mock.Anything, mock.Anything).Return(domain.MovementList{
					fixture.MovementMock(fixture.WithMovementDescription("Compra no supermercado")),
				}, nil)

				mockRecRepo.On("FindByMonth", mock.Anything, mock.Anything).Return([]domain.RecurrentMovement{}, errors.New("error to find recurrents"))
			},
			expectedMovements: nil,
			expectedError: fmt.Errorf("error to find recurrents: %w: %s",
				fmt.Errorf("internal system error"),
				"error to find recurrents",
			),
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

			usecase := NewMovement(
				mockMovRepo,
				mockRecRepo,
				mockWalletRepo,
				new(MockSubCategory),
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

			usecase := NewMovement(
				mockMovRepo,
				mockRecRepo,
				mockWalletRepo,
				new(MockSubCategory),
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
		movementID       uuid.UUID
		updateMovement   domain.Movement
		mockSetup        func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager)
		expectedMovement domain.Movement
		expectedError    error
	}{
		"should update existing unpaid movement with success": {
			movementID: fixture.MovementID,
			updateMovement: domain.Movement{
				Description: "Descrição atualizada",
				Amount:      -75.0,
			},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager) {
				existingMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Descrição original"),
					fixture.AsMovementExpense(50.0),
					fixture.AsMovementUnpaid(),
				)

				expectedUpdatedMovement := fixture.MovementMock(
					fixture.WithMovementDescription("Descrição atualizada"),
					fixture.AsMovementExpense(75.0),
					fixture.AsMovementUnpaid(),
				)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(nil)

				mockMovRepo.On("FindByID", fixture.MovementID).Return(existingMovement, nil)
				mockMovRepo.On("UpdateOne", mock.Anything, fixture.MovementID, mock.MatchedBy(func(movement domain.Movement) bool {
					return movement.Description == "Descrição atualizada" && movement.Amount == -75.0 && !movement.IsPaid
				})).Return(expectedUpdatedMovement, nil)
			},
			expectedMovement: fixture.MovementMock(
				fixture.WithMovementDescription("Descrição atualizada"),
				fixture.AsMovementExpense(75.0),
				fixture.AsMovementUnpaid(),
			),
			expectedError: nil,
		},
		"should return error when subcategory does not belong to category": {
			movementID: fixture.MovementID,
			updateMovement: fixture.MovementMock(
				fixture.WithMovementSubCategoryID(fixture.SubCategoryID),
				fixture.WithMovementCategoryID(fixture.CategoryID),
			),
			mockSetup: func(mockMovRepo *MockMovementRepository, mockRecRepo *MockRecurrentRepository, mockWalletRepo *MockWalletRepository, mockSubCat *MockSubCategory, mockTxManager *MockTransactionManager) {
				mockSubCat.On("IsSubCategoryBelongsToCategory", fixture.SubCategoryID, fixture.CategoryID).Return(false, nil)
			},
			expectedMovement: domain.Movement{},
			expectedError: fmt.Errorf("validate subcategory: %w: %s",
				fmt.Errorf("invalid input"),
				"subcategory does not belong to the provided category",
			),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mockMovRepo := new(MockMovementRepository)
			mockRecRepo := new(MockRecurrentRepository)
			mockWalletRepo := new(MockWalletRepository)
			mockSubCat := new(MockSubCategory)
			mockTxManager := new(MockTransactionManager)

			tc.mockSetup(mockMovRepo, mockRecRepo, mockWalletRepo, mockSubCat, mockTxManager)

			usecase := NewMovement(
				mockMovRepo,
				mockRecRepo,
				mockWalletRepo,
				mockSubCat,
				mockTxManager,
			)

			result, err := usecase.UpdateOne(context.Background(), tc.movementID, tc.updateMovement)

			assert.Equal(t, tc.expectedError, err)
			assert.Equal(t, tc.expectedMovement, result)

			mockMovRepo.AssertExpectations(t)
			mockRecRepo.AssertExpectations(t)
			mockWalletRepo.AssertExpectations(t)
			mockSubCat.AssertExpectations(t)
			mockTxManager.AssertExpectations(t)
		})
	}
}
