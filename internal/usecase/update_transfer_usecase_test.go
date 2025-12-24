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
	updateTransferPairID    = uuid.MustParse("33333333-3333-3333-3333-333333333333")
	updateOriginMovementID  = uuid.MustParse("44444444-4444-4444-4444-444444444444")
	updateDestMovementID    = uuid.MustParse("55555555-5555-5555-5555-555555555555")
	updateOriginWalletID    = uuid.MustParse("66666666-6666-6666-6666-666666666666")
	updateDestWalletID      = uuid.MustParse("77777777-7777-7777-7777-777777777777")
	updateNewOriginWalletID = uuid.MustParse("88888888-8888-8888-8888-888888888888")
	updateNewDestWalletID   = uuid.MustParse("99999999-9999-9999-9999-999999999999")
	updateTransferDate      = time.Date(2023, 6, 15, 10, 0, 0, 0, time.UTC)
	updateTransferNewDate   = time.Date(2023, 7, 20, 10, 0, 0, 0, time.UTC)
	outCategoryID           = uuid.MustParse(domain.InternalTransferOutCategoryID)
	inCategoryID            = uuid.MustParse(domain.InternalTransferInCategoryID)
)

func TestUpdateTransfer_Execute(t *testing.T) {
	tests := map[string]struct {
		input          UpdateTransferInput
		mockSetup      func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager)
		expectedError  error
		validateResult func(t *testing.T, result TransferOutput)
	}{
		"should update only date when only date changed": {
			input: UpdateTransferInput{
				MovementID:          updateOriginMovementID,
				PairID:              updateTransferPairID,
				OriginWalletID:      updateOriginWalletID,
				DestinationWalletID: updateDestWalletID,
				Amount:              500.0,
				Date:                updateTransferNewDate,
				Description:         "Transferência de Conta Corrente para Poupança",
			},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				originMovement := fixture.MovementMock(
					fixture.WithMovementID(updateOriginMovementID),
					fixture.WithMovementAmount(-500.0),
					fixture.WithMovementDate(updateTransferDate),
					fixture.WithMovementWalletID(updateOriginWalletID),
					fixture.WithMovementTypePayment(string(domain.TypePaymentInternalTransfer)),
					fixture.WithMovementPairID(updateTransferPairID),
					fixture.WithMovementDescription("Transferência de Conta Corrente para Poupança"),
					fixture.WithMovementCategoryID(outCategoryID),
				)

				destMovement := fixture.MovementMock(
					fixture.WithMovementID(updateDestMovementID),
					fixture.WithMovementAmount(500.0),
					fixture.WithMovementDate(updateTransferDate),
					fixture.WithMovementWalletID(updateDestWalletID),
					fixture.WithMovementTypePayment(string(domain.TypePaymentInternalTransfer)),
					fixture.WithMovementPairID(updateTransferPairID),
					fixture.WithMovementDescription("Transferência de Conta Corrente para Poupança"),
					fixture.WithMovementCategoryID(inCategoryID),
				)

				originWallet := fixture.WalletMock(
					fixture.WithWalletID(updateOriginWalletID),
					fixture.WithWalletDescription("Conta Corrente"),
					fixture.WithWalletBalance(1000.0),
				)

				destWallet := fixture.WalletMock(
					fixture.WithWalletID(updateDestWalletID),
					fixture.WithWalletDescription("Poupança"),
					fixture.WithWalletBalance(500.0),
				)

				mockMovRepo.On("FindByID", updateOriginMovementID).Return(originMovement, nil)
				mockMovRepo.On("FindByPairID", updateTransferPairID).Return(domain.MovementList{originMovement, destMovement}, nil)

				mockWalletRepo.On("FindByID", &updateOriginWalletID).Return(originWallet, nil)
				mockWalletRepo.On("FindByID", &updateDestWalletID).Return(destWallet, nil)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(nil)

				mockMovRepo.On("Update", mock.Anything, updateOriginMovementID, mock.MatchedBy(func(m domain.Movement) bool {
					return m.Date.Equal(updateTransferNewDate)
				})).Return(domain.Movement{
					ID:          &updateOriginMovementID,
					Amount:      -500.0,
					Date:        &updateTransferNewDate,
					WalletID:    &updateOriginWalletID,
					TypePayment: domain.TypePaymentInternalTransfer,
					PairID:      &updateTransferPairID,
				}, nil)
			},
			expectedError: nil,
			validateResult: func(t *testing.T, result TransferOutput) {
				assert.Equal(t, updateTransferPairID, result.PairID)
			},
		},
		"should update both movements when amount changed": {
			input: UpdateTransferInput{
				MovementID:          updateOriginMovementID,
				PairID:              updateTransferPairID,
				OriginWalletID:      updateOriginWalletID,
				DestinationWalletID: updateDestWalletID,
				Amount:              700.0,
				Date:                updateTransferDate,
				Description:         "",
			},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				originMovement := fixture.MovementMock(
					fixture.WithMovementID(updateOriginMovementID),
					fixture.WithMovementAmount(-500.0),
					fixture.WithMovementDate(updateTransferDate),
					fixture.WithMovementWalletID(updateOriginWalletID),
					fixture.WithMovementTypePayment(string(domain.TypePaymentInternalTransfer)),
					fixture.WithMovementPairID(updateTransferPairID),
					fixture.WithMovementIsPaid(false),
					fixture.WithMovementCategoryID(outCategoryID),
				)

				destMovement := fixture.MovementMock(
					fixture.WithMovementID(updateDestMovementID),
					fixture.WithMovementAmount(500.0),
					fixture.WithMovementDate(updateTransferDate),
					fixture.WithMovementWalletID(updateDestWalletID),
					fixture.WithMovementTypePayment(string(domain.TypePaymentInternalTransfer)),
					fixture.WithMovementPairID(updateTransferPairID),
					fixture.WithMovementIsPaid(false),
					fixture.WithMovementCategoryID(inCategoryID),
				)

				originWallet := fixture.WalletMock(
					fixture.WithWalletID(updateOriginWalletID),
					fixture.WithWalletDescription("Conta Corrente"),
					fixture.WithWalletBalance(1000.0),
				)

				destWallet := fixture.WalletMock(
					fixture.WithWalletID(updateDestWalletID),
					fixture.WithWalletDescription("Poupança"),
					fixture.WithWalletBalance(500.0),
				)

				mockMovRepo.On("FindByID", updateOriginMovementID).Return(originMovement, nil)
				mockMovRepo.On("FindByPairID", updateTransferPairID).Return(domain.MovementList{originMovement, destMovement}, nil)

				mockWalletRepo.On("FindByID", &updateOriginWalletID).Return(originWallet, nil)
				mockWalletRepo.On("FindByID", &updateDestWalletID).Return(destWallet, nil)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(nil)

				mockMovRepo.On("Update", mock.Anything, updateOriginMovementID, mock.MatchedBy(func(m domain.Movement) bool {
					return m.Amount == -700.0
				})).Return(domain.Movement{
					ID:          &updateOriginMovementID,
					Amount:      -700.0,
					WalletID:    &updateOriginWalletID,
					TypePayment: domain.TypePaymentInternalTransfer,
					PairID:      &updateTransferPairID,
				}, nil)

				mockMovRepo.On("Update", mock.Anything, updateDestMovementID, mock.MatchedBy(func(m domain.Movement) bool {
					return m.Amount == 700.0
				})).Return(domain.Movement{
					ID:          &updateDestMovementID,
					Amount:      700.0,
					WalletID:    &updateDestWalletID,
					TypePayment: domain.TypePaymentInternalTransfer,
					PairID:      &updateTransferPairID,
				}, nil)
			},
			expectedError: nil,
			validateResult: func(t *testing.T, result TransferOutput) {
				assert.Equal(t, updateTransferPairID, result.PairID)
				assert.Equal(t, -700.0, result.OriginMovement.Amount)
				assert.Equal(t, 700.0, result.DestinationMovement.Amount)
			},
		},
		"should update both movements and wallets when paid and wallets changed": {
			input: UpdateTransferInput{
				MovementID:          updateOriginMovementID,
				PairID:              updateTransferPairID,
				OriginWalletID:      updateNewOriginWalletID,
				DestinationWalletID: updateNewDestWalletID,
				Amount:              500.0,
				Date:                updateTransferDate,
				Description:         "",
			},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				originMovement := fixture.MovementMock(
					fixture.WithMovementID(updateOriginMovementID),
					fixture.WithMovementAmount(-500.0),
					fixture.WithMovementDate(updateTransferDate),
					fixture.WithMovementWalletID(updateOriginWalletID),
					fixture.WithMovementTypePayment(string(domain.TypePaymentInternalTransfer)),
					fixture.WithMovementPairID(updateTransferPairID),
					fixture.WithMovementIsPaid(true),
					fixture.WithMovementCategoryID(outCategoryID),
				)

				destMovement := fixture.MovementMock(
					fixture.WithMovementID(updateDestMovementID),
					fixture.WithMovementAmount(500.0),
					fixture.WithMovementDate(updateTransferDate),
					fixture.WithMovementWalletID(updateDestWalletID),
					fixture.WithMovementTypePayment(string(domain.TypePaymentInternalTransfer)),
					fixture.WithMovementPairID(updateTransferPairID),
					fixture.WithMovementIsPaid(true),
					fixture.WithMovementCategoryID(inCategoryID),
				)

				oldOriginWallet := fixture.WalletMock(
					fixture.WithWalletID(updateOriginWalletID),
					fixture.WithWalletDescription("Conta Corrente"),
					fixture.WithWalletBalance(500.0),
				)

				oldDestWallet := fixture.WalletMock(
					fixture.WithWalletID(updateDestWalletID),
					fixture.WithWalletDescription("Poupança"),
					fixture.WithWalletBalance(1000.0),
				)

				newOriginWallet := fixture.WalletMock(
					fixture.WithWalletID(updateNewOriginWalletID),
					fixture.WithWalletDescription("Nova Conta Corrente"),
					fixture.WithWalletBalance(2000.0),
				)

				newDestWallet := fixture.WalletMock(
					fixture.WithWalletID(updateNewDestWalletID),
					fixture.WithWalletDescription("Nova Poupança"),
					fixture.WithWalletBalance(100.0),
				)

				mockMovRepo.On("FindByID", updateOriginMovementID).Return(originMovement, nil)
				mockMovRepo.On("FindByPairID", updateTransferPairID).Return(domain.MovementList{originMovement, destMovement}, nil)

				mockWalletRepo.On("FindByID", &updateNewOriginWalletID).Return(newOriginWallet, nil)
				mockWalletRepo.On("FindByID", &updateNewDestWalletID).Return(newDestWallet, nil)
				mockWalletRepo.On("FindByID", &updateOriginWalletID).Return(oldOriginWallet, nil)
				mockWalletRepo.On("FindByID", &updateDestWalletID).Return(oldDestWallet, nil)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(nil)

				mockWalletRepo.On("UpdateAmount", mock.Anything, &updateOriginWalletID, 1000.0).Return(nil)
				mockWalletRepo.On("UpdateAmount", mock.Anything, &updateDestWalletID, 500.0).Return(nil)

				mockMovRepo.On("Update", mock.Anything, updateOriginMovementID, mock.Anything).Return(domain.Movement{
					ID:          &updateOriginMovementID,
					Amount:      -500.0,
					WalletID:    &updateNewOriginWalletID,
					TypePayment: domain.TypePaymentInternalTransfer,
					PairID:      &updateTransferPairID,
				}, nil)

				mockMovRepo.On("Update", mock.Anything, updateDestMovementID, mock.Anything).Return(domain.Movement{
					ID:          &updateDestMovementID,
					Amount:      500.0,
					WalletID:    &updateNewDestWalletID,
					TypePayment: domain.TypePaymentInternalTransfer,
					PairID:      &updateTransferPairID,
				}, nil)

				mockWalletRepo.On("UpdateAmount", mock.Anything, &updateNewOriginWalletID, 1500.0).Return(nil)
				mockWalletRepo.On("UpdateAmount", mock.Anything, &updateNewDestWalletID, 600.0).Return(nil)
			},
			expectedError: nil,
			validateResult: func(t *testing.T, result TransferOutput) {
				assert.Equal(t, updateTransferPairID, result.PairID)
			},
		},
		"should return error when origin and destination wallets are the same": {
			input: UpdateTransferInput{
				MovementID:          updateOriginMovementID,
				PairID:              updateTransferPairID,
				OriginWalletID:      updateOriginWalletID,
				DestinationWalletID: updateOriginWalletID,
				Amount:              500.0,
				Date:                updateTransferDate,
			},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
			},
			expectedError: ErrSameWalletTransfer,
			validateResult: func(t *testing.T, result TransferOutput) {
				assert.Equal(t, TransferOutput{}, result)
			},
		},
		"should return error when amount is zero": {
			input: UpdateTransferInput{
				MovementID:          updateOriginMovementID,
				PairID:              updateTransferPairID,
				OriginWalletID:      updateOriginWalletID,
				DestinationWalletID: updateDestWalletID,
				Amount:              0,
				Date:                updateTransferDate,
			},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
			},
			expectedError: ErrInvalidTransferAmount,
			validateResult: func(t *testing.T, result TransferOutput) {
				assert.Equal(t, TransferOutput{}, result)
			},
		},
		"should return error when date is zero": {
			input: UpdateTransferInput{
				MovementID:          updateOriginMovementID,
				PairID:              updateTransferPairID,
				OriginWalletID:      updateOriginWalletID,
				DestinationWalletID: updateDestWalletID,
				Amount:              500.0,
				Date:                time.Time{},
			},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
			},
			expectedError: ErrDateRequired,
			validateResult: func(t *testing.T, result TransferOutput) {
				assert.Equal(t, TransferOutput{}, result)
			},
		},
		"should return error when movement not found": {
			input: UpdateTransferInput{
				MovementID:          updateOriginMovementID,
				PairID:              updateTransferPairID,
				OriginWalletID:      updateOriginWalletID,
				DestinationWalletID: updateDestWalletID,
				Amount:              500.0,
				Date:                updateTransferDate,
			},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				mockMovRepo.On("FindByID", updateOriginMovementID).Return(domain.Movement{}, errors.New("movement not found"))
			},
			expectedError: errors.New("error finding movement: movement not found"),
			validateResult: func(t *testing.T, result TransferOutput) {
				assert.Equal(t, TransferOutput{}, result)
			},
		},
		"should return error when movement is not internal transfer": {
			input: UpdateTransferInput{
				MovementID:          updateOriginMovementID,
				PairID:              updateTransferPairID,
				OriginWalletID:      updateOriginWalletID,
				DestinationWalletID: updateDestWalletID,
				Amount:              500.0,
				Date:                updateTransferDate,
			},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				movement := fixture.MovementMock(
					fixture.WithMovementID(updateOriginMovementID),
					fixture.WithMovementTypePayment(string(domain.TypePaymentPix)),
				)
				mockMovRepo.On("FindByID", updateOriginMovementID).Return(movement, nil)
			},
			expectedError: ErrMovementNotInternalTransfer,
			validateResult: func(t *testing.T, result TransferOutput) {
				assert.Equal(t, TransferOutput{}, result)
			},
		},
		"should return error when pair_id does not match": {
			input: UpdateTransferInput{
				MovementID:          updateOriginMovementID,
				PairID:              updateTransferPairID,
				OriginWalletID:      updateOriginWalletID,
				DestinationWalletID: updateDestWalletID,
				Amount:              500.0,
				Date:                updateTransferDate,
			},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				differentPairID := uuid.New()
				movement := fixture.MovementMock(
					fixture.WithMovementID(updateOriginMovementID),
					fixture.WithMovementTypePayment(string(domain.TypePaymentInternalTransfer)),
					fixture.WithMovementPairID(differentPairID),
				)
				mockMovRepo.On("FindByID", updateOriginMovementID).Return(movement, nil)
			},
			expectedError: ErrTransferPairMismatch,
			validateResult: func(t *testing.T, result TransferOutput) {
				assert.Equal(t, TransferOutput{}, result)
			},
		},
		"should return error when pair has only one movement": {
			input: UpdateTransferInput{
				MovementID:          updateOriginMovementID,
				PairID:              updateTransferPairID,
				OriginWalletID:      updateOriginWalletID,
				DestinationWalletID: updateDestWalletID,
				Amount:              500.0,
				Date:                updateTransferDate,
			},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				originMovement := fixture.MovementMock(
					fixture.WithMovementID(updateOriginMovementID),
					fixture.WithMovementAmount(-500.0),
					fixture.WithMovementTypePayment(string(domain.TypePaymentInternalTransfer)),
					fixture.WithMovementPairID(updateTransferPairID),
				)

				mockMovRepo.On("FindByID", updateOriginMovementID).Return(originMovement, nil)
				mockMovRepo.On("FindByPairID", updateTransferPairID).Return(domain.MovementList{originMovement}, nil)
			},
			expectedError: ErrTransferPairNotFound,
			validateResult: func(t *testing.T, result TransferOutput) {
				assert.Equal(t, TransferOutput{}, result)
			},
		},
		"should return error when insufficient balance for paid transfer": {
			input: UpdateTransferInput{
				MovementID:          updateOriginMovementID,
				PairID:              updateTransferPairID,
				OriginWalletID:      updateNewOriginWalletID,
				DestinationWalletID: updateNewDestWalletID,
				Amount:              5000.0,
				Date:                updateTransferDate,
			},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockWalletRepo *MockWalletRepository, mockTxManager *MockTransactionManager) {
				originMovement := fixture.MovementMock(
					fixture.WithMovementID(updateOriginMovementID),
					fixture.WithMovementAmount(-500.0),
					fixture.WithMovementDate(updateTransferDate),
					fixture.WithMovementWalletID(updateOriginWalletID),
					fixture.WithMovementTypePayment(string(domain.TypePaymentInternalTransfer)),
					fixture.WithMovementPairID(updateTransferPairID),
					fixture.WithMovementIsPaid(true),
					fixture.WithMovementCategoryID(outCategoryID),
				)

				destMovement := fixture.MovementMock(
					fixture.WithMovementID(updateDestMovementID),
					fixture.WithMovementAmount(500.0),
					fixture.WithMovementDate(updateTransferDate),
					fixture.WithMovementWalletID(updateDestWalletID),
					fixture.WithMovementTypePayment(string(domain.TypePaymentInternalTransfer)),
					fixture.WithMovementPairID(updateTransferPairID),
					fixture.WithMovementIsPaid(true),
					fixture.WithMovementCategoryID(inCategoryID),
				)

				oldOriginWallet := fixture.WalletMock(
					fixture.WithWalletID(updateOriginWalletID),
					fixture.WithWalletDescription("Conta Corrente"),
					fixture.WithWalletBalance(500.0),
				)

				oldDestWallet := fixture.WalletMock(
					fixture.WithWalletID(updateDestWalletID),
					fixture.WithWalletDescription("Poupança"),
					fixture.WithWalletBalance(1000.0),
				)

				newOriginWallet := fixture.WalletMock(
					fixture.WithWalletID(updateNewOriginWalletID),
					fixture.WithWalletDescription("Nova Conta"),
					fixture.WithWalletBalance(100.0),
				)

				newDestWallet := fixture.WalletMock(
					fixture.WithWalletID(updateNewDestWalletID),
					fixture.WithWalletDescription("Nova Poupança"),
					fixture.WithWalletBalance(100.0),
				)

				mockMovRepo.On("FindByID", updateOriginMovementID).Return(originMovement, nil)
				mockMovRepo.On("FindByPairID", updateTransferPairID).Return(domain.MovementList{originMovement, destMovement}, nil)

				mockWalletRepo.On("FindByID", &updateNewOriginWalletID).Return(newOriginWallet, nil)
				mockWalletRepo.On("FindByID", &updateNewDestWalletID).Return(newDestWallet, nil)
				mockWalletRepo.On("FindByID", &updateOriginWalletID).Return(oldOriginWallet, nil)
				mockWalletRepo.On("FindByID", &updateDestWalletID).Return(oldDestWallet, nil)

				mockTxManager.On("WithTransaction", mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(0).(func(*gorm.DB) error)
						_ = fn(nil)
					}).Return(ErrInsufficientBalance)

				mockWalletRepo.On("UpdateAmount", mock.Anything, &updateOriginWalletID, 1000.0).Return(nil)
				mockWalletRepo.On("UpdateAmount", mock.Anything, &updateDestWalletID, 500.0).Return(nil)
			},
			expectedError: ErrInsufficientBalance,
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

			uc := NewUpdateTransfer(
				mockMovRepo,
				mockWalletRepo,
				mockTxManager,
			)

			result, err := uc.Execute(context.Background(), tt.input)

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
