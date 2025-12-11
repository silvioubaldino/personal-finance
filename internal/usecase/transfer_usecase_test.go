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

var (
	originWalletID      = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	destinationWalletID = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	transferDate        = time.Date(2023, 6, 15, 10, 0, 0, 0, time.UTC)
)

func TestTransfer_Execute(t *testing.T) {
	tests := map[string]struct {
		input          TransferInput
		mockSetup      func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager)
		expectedError  error
		validateResult func(t *testing.T, result TransferOutput)
	}{
		"should create transfer with success when is_paid is true": {
			input: TransferInput{
				OriginWalletID:      originWalletID,
				DestinationWalletID: destinationWalletID,
				Amount:              500.0,
				Date:                transferDate,
				IsPaid:              true,
			},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				originWallet := fixture.WalletMock(
					fixture.WithWalletID(originWalletID),
					fixture.WithWalletDescription("Conta Corrente"),
					fixture.WithWalletBalance(1000.0),
				)

				destinationWallet := fixture.WalletMock(
					fixture.WithWalletID(destinationWalletID),
					fixture.WithWalletDescription("Poupança"),
					fixture.WithWalletBalance(500.0),
				)

				mockWalletRepo.On("FindByID", &originWalletID).Return(originWallet, nil)
				mockWalletRepo.On("FindByID", &destinationWalletID).Return(destinationWallet, nil)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(nil)

				mockMovRepo.On("Add", mock.Anything, mock.MatchedBy(func(m domain.Movement) bool {
					return m.Amount == -500.0 && *m.WalletID == originWalletID
				})).Return(domain.Movement{Amount: -500.0, WalletID: &originWalletID}, nil)

				mockMovRepo.On("Add", mock.Anything, mock.MatchedBy(func(m domain.Movement) bool {
					return m.Amount == 500.0 && *m.WalletID == destinationWalletID
				})).Return(domain.Movement{Amount: 500.0, WalletID: &destinationWalletID}, nil)

				mockWalletRepo.On("UpdateAmount", mock.Anything, &originWalletID, 500.0).Return(nil)
				mockWalletRepo.On("UpdateAmount", mock.Anything, &destinationWalletID, 1000.0).Return(nil)
			},
			expectedError: nil,
			validateResult: func(t *testing.T, result TransferOutput) {
				assert.NotEqual(t, uuid.Nil, result.PairID)
				assert.Equal(t, -500.0, result.OriginMovement.Amount)
				assert.Equal(t, 500.0, result.DestinationMovement.Amount)
			},
		},
		"should create transfer with success when is_paid is false": {
			input: TransferInput{
				OriginWalletID:      originWalletID,
				DestinationWalletID: destinationWalletID,
				Amount:              300.0,
				Date:                transferDate,
				IsPaid:              false,
			},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				originWallet := fixture.WalletMock(
					fixture.WithWalletID(originWalletID),
					fixture.WithWalletDescription("Conta Corrente"),
					fixture.WithWalletBalance(1000.0),
				)

				destinationWallet := fixture.WalletMock(
					fixture.WithWalletID(destinationWalletID),
					fixture.WithWalletDescription("Poupança"),
					fixture.WithWalletBalance(500.0),
				)

				mockWalletRepo.On("FindByID", &originWalletID).Return(originWallet, nil)
				mockWalletRepo.On("FindByID", &destinationWalletID).Return(destinationWallet, nil)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(nil)

				mockMovRepo.On("Add", mock.Anything, mock.MatchedBy(func(m domain.Movement) bool {
					return m.Amount == -300.0 && *m.WalletID == originWalletID
				})).Return(domain.Movement{Amount: -300.0, WalletID: &originWalletID}, nil)

				mockMovRepo.On("Add", mock.Anything, mock.MatchedBy(func(m domain.Movement) bool {
					return m.Amount == 300.0 && *m.WalletID == destinationWalletID
				})).Return(domain.Movement{Amount: 300.0, WalletID: &destinationWalletID}, nil)
			},
			expectedError: nil,
			validateResult: func(t *testing.T, result TransferOutput) {
				assert.NotEqual(t, uuid.Nil, result.PairID)
			},
		},
		"should create transfer with custom description": {
			input: TransferInput{
				OriginWalletID:      originWalletID,
				DestinationWalletID: destinationWalletID,
				Amount:              200.0,
				Date:                transferDate,
				Description:         "Reserva de emergência",
				IsPaid:              true,
			},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				originWallet := fixture.WalletMock(
					fixture.WithWalletID(originWalletID),
					fixture.WithWalletDescription("Conta Corrente"),
					fixture.WithWalletBalance(1000.0),
				)

				destinationWallet := fixture.WalletMock(
					fixture.WithWalletID(destinationWalletID),
					fixture.WithWalletDescription("Poupança"),
					fixture.WithWalletBalance(500.0),
				)

				mockWalletRepo.On("FindByID", &originWalletID).Return(originWallet, nil)
				mockWalletRepo.On("FindByID", &destinationWalletID).Return(destinationWallet, nil)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(nil)

				mockMovRepo.On("Add", mock.Anything, mock.MatchedBy(func(m domain.Movement) bool {
					return m.Description == "Reserva de emergência"
				})).Return(domain.Movement{Description: "Reserva de emergência"}, nil)

				mockMovRepo.On("Add", mock.Anything, mock.MatchedBy(func(m domain.Movement) bool {
					return m.Description == "Reserva de emergência"
				})).Return(domain.Movement{Description: "Reserva de emergência"}, nil)

				mockWalletRepo.On("UpdateAmount", mock.Anything, &originWalletID, mock.Anything).Return(nil)
				mockWalletRepo.On("UpdateAmount", mock.Anything, &destinationWalletID, mock.Anything).Return(nil)
			},
			expectedError: nil,
			validateResult: func(t *testing.T, result TransferOutput) {
				assert.NotEqual(t, uuid.Nil, result.PairID)
			},
		},
		"should return error when origin and destination wallets are the same": {
			input: TransferInput{
				OriginWalletID:      originWalletID,
				DestinationWalletID: originWalletID,
				Amount:              500.0,
				Date:                transferDate,
				IsPaid:              true,
			},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
			},
			expectedError: ErrSameWalletTransfer,
			validateResult: func(t *testing.T, result TransferOutput) {
				assert.Equal(t, TransferOutput{}, result)
			},
		},
		"should return error when amount is zero": {
			input: TransferInput{
				OriginWalletID:      originWalletID,
				DestinationWalletID: destinationWalletID,
				Amount:              0,
				Date:                transferDate,
				IsPaid:              true,
			},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
			},
			expectedError: ErrInvalidTransferAmount,
			validateResult: func(t *testing.T, result TransferOutput) {
				assert.Equal(t, TransferOutput{}, result)
			},
		},
		"should return error when amount is negative": {
			input: TransferInput{
				OriginWalletID:      originWalletID,
				DestinationWalletID: destinationWalletID,
				Amount:              -100.0,
				Date:                transferDate,
				IsPaid:              true,
			},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
			},
			expectedError: ErrInvalidTransferAmount,
			validateResult: func(t *testing.T, result TransferOutput) {
				assert.Equal(t, TransferOutput{}, result)
			},
		},
		"should return error when date is zero": {
			input: TransferInput{
				OriginWalletID:      originWalletID,
				DestinationWalletID: destinationWalletID,
				Amount:              500.0,
				Date:                time.Time{},
				IsPaid:              true,
			},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
			},
			expectedError: ErrDateRequired,
			validateResult: func(t *testing.T, result TransferOutput) {
				assert.Equal(t, TransferOutput{}, result)
			},
		},
		"should return error when origin wallet not found": {
			input: TransferInput{
				OriginWalletID:      originWalletID,
				DestinationWalletID: destinationWalletID,
				Amount:              500.0,
				Date:                transferDate,
				IsPaid:              true,
			},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				mockWalletRepo.On("FindByID", &originWalletID).Return(domain.Wallet{}, errors.New("wallet not found"))
			},
			expectedError: errors.New("error finding origin wallet: wallet not found"),
			validateResult: func(t *testing.T, result TransferOutput) {
				assert.Equal(t, TransferOutput{}, result)
			},
		},
		"should return error when destination wallet not found": {
			input: TransferInput{
				OriginWalletID:      originWalletID,
				DestinationWalletID: destinationWalletID,
				Amount:              500.0,
				Date:                transferDate,
				IsPaid:              true,
			},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				originWallet := fixture.WalletMock(
					fixture.WithWalletID(originWalletID),
					fixture.WithWalletBalance(1000.0),
				)

				mockWalletRepo.On("FindByID", &originWalletID).Return(originWallet, nil)
				mockWalletRepo.On("FindByID", &destinationWalletID).Return(domain.Wallet{}, errors.New("wallet not found"))
			},
			expectedError: errors.New("error finding destination wallet: wallet not found"),
			validateResult: func(t *testing.T, result TransferOutput) {
				assert.Equal(t, TransferOutput{}, result)
			},
		},
		"should return error when origin wallet has insufficient balance": {
			input: TransferInput{
				OriginWalletID:      originWalletID,
				DestinationWalletID: destinationWalletID,
				Amount:              1500.0,
				Date:                transferDate,
				IsPaid:              true,
			},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				originWallet := fixture.WalletMock(
					fixture.WithWalletID(originWalletID),
					fixture.WithWalletBalance(1000.0),
				)

				destinationWallet := fixture.WalletMock(
					fixture.WithWalletID(destinationWalletID),
					fixture.WithWalletBalance(500.0),
				)

				mockWalletRepo.On("FindByID", &originWalletID).Return(originWallet, nil)
				mockWalletRepo.On("FindByID", &destinationWalletID).Return(destinationWallet, nil)
			},
			expectedError: ErrInsufficientBalance,
			validateResult: func(t *testing.T, result TransferOutput) {
				assert.Equal(t, TransferOutput{}, result)
			},
		},
		"should return error when fails to add origin movement": {
			input: TransferInput{
				OriginWalletID:      originWalletID,
				DestinationWalletID: destinationWalletID,
				Amount:              500.0,
				Date:                transferDate,
				IsPaid:              true,
			},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				originWallet := fixture.WalletMock(
					fixture.WithWalletID(originWalletID),
					fixture.WithWalletBalance(1000.0),
				)

				destinationWallet := fixture.WalletMock(
					fixture.WithWalletID(destinationWalletID),
					fixture.WithWalletBalance(500.0),
				)

				mockWalletRepo.On("FindByID", &originWalletID).Return(originWallet, nil)
				mockWalletRepo.On("FindByID", &destinationWalletID).Return(destinationWallet, nil)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(errors.New("error creating origin movement: database error"))

				mockMovRepo.On("Add", mock.Anything, mock.Anything).Return(domain.Movement{}, errors.New("database error"))
			},
			expectedError: errors.New("error creating origin movement: database error"),
			validateResult: func(t *testing.T, result TransferOutput) {
				assert.Equal(t, TransferOutput{}, result)
			},
		},
		"should return error when fails to add destination movement": {
			input: TransferInput{
				OriginWalletID:      originWalletID,
				DestinationWalletID: destinationWalletID,
				Amount:              500.0,
				Date:                transferDate,
				IsPaid:              true,
			},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				originWallet := fixture.WalletMock(
					fixture.WithWalletID(originWalletID),
					fixture.WithWalletBalance(1000.0),
				)

				destinationWallet := fixture.WalletMock(
					fixture.WithWalletID(destinationWalletID),
					fixture.WithWalletBalance(500.0),
				)

				mockWalletRepo.On("FindByID", &originWalletID).Return(originWallet, nil)
				mockWalletRepo.On("FindByID", &destinationWalletID).Return(destinationWallet, nil)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(errors.New("error creating destination movement: database error"))

				mockMovRepo.On("Add", mock.Anything, mock.MatchedBy(func(m domain.Movement) bool {
					return m.Amount == -500.0
				})).Return(domain.Movement{Amount: -500.0}, nil)

				mockMovRepo.On("Add", mock.Anything, mock.MatchedBy(func(m domain.Movement) bool {
					return m.Amount == 500.0
				})).Return(domain.Movement{}, errors.New("database error"))
			},
			expectedError: errors.New("error creating destination movement: database error"),
			validateResult: func(t *testing.T, result TransferOutput) {
				assert.Equal(t, TransferOutput{}, result)
			},
		},
		"should return error when fails to update origin wallet balance": {
			input: TransferInput{
				OriginWalletID:      originWalletID,
				DestinationWalletID: destinationWalletID,
				Amount:              500.0,
				Date:                transferDate,
				IsPaid:              true,
			},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				originWallet := fixture.WalletMock(
					fixture.WithWalletID(originWalletID),
					fixture.WithWalletBalance(1000.0),
				)

				destinationWallet := fixture.WalletMock(
					fixture.WithWalletID(destinationWalletID),
					fixture.WithWalletBalance(500.0),
				)

				mockWalletRepo.On("FindByID", &originWalletID).Return(originWallet, nil)
				mockWalletRepo.On("FindByID", &destinationWalletID).Return(destinationWallet, nil)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(errors.New("error updating origin wallet balance: database error"))

				mockMovRepo.On("Add", mock.Anything, mock.MatchedBy(func(m domain.Movement) bool {
					return m.Amount == -500.0
				})).Return(domain.Movement{Amount: -500.0, WalletID: &originWalletID}, nil)

				mockMovRepo.On("Add", mock.Anything, mock.MatchedBy(func(m domain.Movement) bool {
					return m.Amount == 500.0
				})).Return(domain.Movement{Amount: 500.0, WalletID: &destinationWalletID}, nil)

				mockWalletRepo.On("UpdateAmount", mock.Anything, &originWalletID, mock.Anything).Return(errors.New("database error"))
			},
			expectedError: errors.New("error updating origin wallet balance: database error"),
			validateResult: func(t *testing.T, result TransferOutput) {
				assert.Equal(t, TransferOutput{}, result)
			},
		},
		"should return error when fails to update destination wallet balance": {
			input: TransferInput{
				OriginWalletID:      originWalletID,
				DestinationWalletID: destinationWalletID,
				Amount:              500.0,
				Date:                transferDate,
				IsPaid:              true,
			},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				originWallet := fixture.WalletMock(
					fixture.WithWalletID(originWalletID),
					fixture.WithWalletBalance(1000.0),
				)

				destinationWallet := fixture.WalletMock(
					fixture.WithWalletID(destinationWalletID),
					fixture.WithWalletBalance(500.0),
				)

				mockWalletRepo.On("FindByID", &originWalletID).Return(originWallet, nil)
				mockWalletRepo.On("FindByID", &destinationWalletID).Return(destinationWallet, nil)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(errors.New("error updating destination wallet balance: database error"))

				mockMovRepo.On("Add", mock.Anything, mock.MatchedBy(func(m domain.Movement) bool {
					return m.Amount == -500.0
				})).Return(domain.Movement{Amount: -500.0, WalletID: &originWalletID}, nil)

				mockMovRepo.On("Add", mock.Anything, mock.MatchedBy(func(m domain.Movement) bool {
					return m.Amount == 500.0
				})).Return(domain.Movement{Amount: 500.0, WalletID: &destinationWalletID}, nil)

				mockWalletRepo.On("UpdateAmount", mock.Anything, &originWalletID, 500.0).Return(nil)
				mockWalletRepo.On("UpdateAmount", mock.Anything, &destinationWalletID, mock.Anything).Return(errors.New("database error"))
			},
			expectedError: errors.New("error updating destination wallet balance: database error"),
			validateResult: func(t *testing.T, result TransferOutput) {
				assert.Equal(t, TransferOutput{}, result)
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockMovRepo := new(MockMovementRepository)
			mockWalletRepo := new(MockWalletRepository)
			mockTxManager := new(MockTransactionManager)

			if tt.mockSetup != nil {
				tt.mockSetup(mockMovRepo, mockWalletRepo, mockTxManager)
			}

			usecase := NewTransfer(
				mockMovRepo,
				mockWalletRepo,
				mockTxManager,
			)

			result, err := usecase.Execute(context.Background(), tt.input)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}

			if tt.validateResult != nil {
				tt.validateResult(t, result)
			}

			mockMovRepo.AssertExpectations(t)
			mockWalletRepo.AssertExpectations(t)
			mockTxManager.AssertExpectations(t)
		})
	}
}
