package service_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"personal-finance/internal/domain/movement/repository"
	"personal-finance/internal/domain/movement/service"
	transactionService "personal-finance/internal/domain/transaction/service"
	"personal-finance/internal/model"

	"github.com/stretchr/testify/require"
)

var (
	mockedUUID = uuid.New()

	now               = time.Now()
	aluguelmockedTime = time.Date(2022, time.September, 0o1, 0, 0, 0, 0, time.Local)
	energiaMockedTime = time.Date(2022, time.September, 15, 0, 0, 0, 0, time.Local)
	aguaMockedTime    = time.Date(2022, time.September, 30, 0, 0, 0, 0, time.Local)
	movementsMock     = []model.Movement{
		{
			Description:   "Aluguel",
			Amount:        -1000.0,
			Date:          &aluguelmockedTime,
			WalletID:      1,
			TypePaymentID: 1,
			CategoryID:    2,
			DateCreate:    now,
			DateUpdate:    now,
		},
		{
			Description:   "Energia",
			Amount:        -300.0,
			Date:          &energiaMockedTime,
			WalletID:      1,
			TypePaymentID: 1,
			CategoryID:    2,
			DateCreate:    now,
			DateUpdate:    now,
		},
		{
			Description:   "Agua",
			Amount:        120.0,
			Date:          &aguaMockedTime,
			WalletID:      1,
			TypePaymentID: 1,
			CategoryID:    2,
			DateCreate:    now,
			DateUpdate:    now,
		},
	}
)

func TestService_Add(t *testing.T) {
	tt := []struct {
		name                string
		inputTransaction    model.Movement
		MockedTransaction   model.Movement
		expectedTransaction model.Movement
		MockedError         error
		expectedErr         error
		movementMockSvc     func() *repository.Mock
		transactionMockSvc  func() *transactionService.Mock
	}{
		{
			name:                "Success - estimate",
			inputTransaction:    mockMovement(movementsMock[0].Description, 2, nil),
			expectedTransaction: movementsMock[0],
			expectedErr:         nil,
			movementMockSvc: func() *repository.Mock {
				repo := &repository.Mock{}
				repo.On("Add", mockMovement(movementsMock[0].Description, 2, nil), "userID").
					Return(movementsMock[0], nil)
				return repo
			},
			transactionMockSvc: func() *transactionService.Mock {
				return nil
			},
		},
		{
			name:                "Success - direct done",
			inputTransaction:    mockMovement(movementsMock[0].Description, 1, nil),
			expectedTransaction: movementsMock[0],
			expectedErr:         nil,
			movementMockSvc: func() *repository.Mock {
				return nil
			},
			transactionMockSvc: func() *transactionService.Mock {
				repo := &transactionService.Mock{}
				repo.On("AddDirectDoneTransaction", mockMovement(movementsMock[0].Description, 1, nil)).
					Return(mockTransaction(nil, &movementsMock[0], nil, model.MovementList{movementsMock[0]}),
						nil)
				return repo
			},
		},
		{
			name:                "Success - add done",
			inputTransaction:    mockMovement(movementsMock[0].Description, 1, &mockedUUID),
			expectedTransaction: movementsMock[0],
			expectedErr:         nil,
			movementMockSvc: func() *repository.Mock {
				repo := &repository.Mock{}
				repo.On("AddUpdatingWallet",
					mockMovement(movementsMock[0].Description, 1, &mockedUUID),
					"userID").Return(movementsMock[0], nil)
				return repo
			},
			transactionMockSvc: func() *transactionService.Mock {
				return nil
			},
		},
		{
			name:             "Error - planned transactions must not have transactionID",
			inputTransaction: mockMovement(movementsMock[0].Description, 2, &mockedUUID),
			expectedErr:      errors.New("planned transactions must not have transactionID"),
			movementMockSvc: func() *repository.Mock {
				return nil
			},
			transactionMockSvc: func() *transactionService.Mock {
				return nil
			},
		},
		{
			name: "repository error",
			inputTransaction: model.Movement{
				Description: movementsMock[0].Description,
				StatusID:    2,
			},
			expectedTransaction: model.Movement{},
			expectedErr:         fmt.Errorf("error to add transactions: %w", errors.New("repository error")),
			movementMockSvc: func() *repository.Mock {
				repo := &repository.Mock{}
				repo.On("Add", mockMovement(movementsMock[0].Description, 2, nil), "userID").
					Return(model.Movement{}, errors.New("repository error"))
				return repo
			},
			transactionMockSvc: func() *transactionService.Mock {
				return nil
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			movementRepoMock := tc.movementMockSvc()
			transactionRepoMock := tc.transactionMockSvc()
			svc := service.NewMovementService(movementRepoMock, transactionRepoMock)

			result, err := svc.Add(context.Background(), tc.inputTransaction, "userID")
			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.expectedTransaction, result)
		})
	}
}

func TestService_FindByPeriod(t *testing.T) {
	tt := []struct {
		name                 string
		inputPeriod          model.Period
		mockedTransactions   []model.Movement
		expectedTransactions []model.Movement
		mockedError          error
		expectedErr          error
	}{
		{
			name: "Success",
			inputPeriod: model.Period{
				From: *movementsMock[0].Date,
				To:   *movementsMock[1].Date,
			},
			mockedTransactions: []model.Movement{
				movementsMock[0],
				movementsMock[1],
			},
			expectedTransactions: movementsMock,
			mockedError:          nil,
			expectedErr:          nil,
		},
		{
			name: "no cars found",
			inputPeriod: model.Period{
				From: *movementsMock[0].Date,
				To:   *movementsMock[1].Date,
			},
			mockedTransactions:   []model.Movement{},
			expectedTransactions: []model.Movement{},
			mockedError:          errors.New("repository error"),
			expectedErr:          fmt.Errorf("error to find transactions: %w", errors.New("repository error")),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			repoMock := &repository.Mock{}
			repoMock.On("FindByPeriod", tc.inputPeriod, "userID").
				Return(tc.expectedTransactions, tc.mockedError)
			svc := service.NewMovementService(repoMock, nil)

			result, err := svc.FindByPeriod(context.Background(), tc.inputPeriod, "userID")
			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.expectedTransactions, result)
		})
	}
}

func TestService_FindByID(t *testing.T) {
	tt := []struct {
		name                string
		inputID             uuid.UUID
		expectedTransaction model.Movement
		mockedError         error
		expectedErr         error
	}{
		{
			name:                "Success",
			inputID:             mockedUUID,
			expectedTransaction: movementsMock[0],
			mockedError:         nil,
			expectedErr:         nil,
		},
		{
			name:                "no transactions found",
			inputID:             mockedUUID,
			expectedTransaction: model.Movement{},
			mockedError:         errors.New("repository error"),
			expectedErr:         fmt.Errorf("error to find transactions: %w", errors.New("repository error")),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			repoMock := &repository.Mock{}
			repoMock.On("FindByID", tc.inputID, "userID").
				Return(tc.expectedTransaction, tc.mockedError)
			svc := service.NewMovementService(repoMock, nil)

			result, err := svc.FindByID(context.Background(), tc.inputID, "userID")
			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.expectedTransaction, result)
		})
	}
}

func TestService_Update(t *testing.T) {
	tt := []struct {
		name                string
		inputTransaction    model.Movement
		mockedTransaction   model.Movement
		expectedTransaction model.Movement
		inputID             uuid.UUID
		mockedError         error
		expectedErr         error
	}{
		{
			name: "Success",
			inputTransaction: model.Movement{
				Description: movementsMock[1].Description,
			},
			mockedTransaction: model.Movement{
				ID:          movementsMock[0].ID,
				Description: movementsMock[1].Description,
				DateCreate:  movementsMock[0].DateCreate,
			},
			expectedTransaction: model.Movement{
				ID:          movementsMock[0].ID,
				Description: movementsMock[1].Description,
				DateCreate:  movementsMock[0].DateCreate,
			},
			inputID:     mockedUUID,
			mockedError: nil,
			expectedErr: nil,
		},
		{
			name: "repository error",
			inputTransaction: model.Movement{
				Description: movementsMock[1].Description,
			},
			mockedTransaction:   model.Movement{},
			expectedTransaction: model.Movement{},
			inputID:             mockedUUID,
			mockedError:         errors.New("repository error"),
			expectedErr:         fmt.Errorf("error updating transactions: %w", errors.New("repository error")),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			repoMock := &repository.Mock{}
			repoMock.On("Update", tc.inputID, tc.inputTransaction, "userID").
				Return(tc.mockedTransaction, tc.mockedError)

			svc := service.NewMovementService(repoMock, nil)

			result, err := svc.Update(context.Background(), tc.inputID, tc.inputTransaction, "userID")
			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.expectedTransaction, result)
		})
	}
}

func TestService_Delete(t *testing.T) {
	tt := []struct {
		name        string
		inputID     uuid.UUID
		mockedErr   error
		expectedErr error
	}{
		{
			name:        "Success",
			inputID:     mockedUUID,
			mockedErr:   nil,
			expectedErr: nil,
		},
		{
			name:        "fail",
			inputID:     mockedUUID,
			mockedErr:   errors.New("repository error"),
			expectedErr: fmt.Errorf("error deleting transactions: %w", errors.New("repository error")),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			repoMock := &repository.Mock{}
			repoMock.On("Delete", tc.inputID, "userID").
				Return(tc.mockedErr)
			svc := service.NewMovementService(repoMock, nil)

			err := svc.Delete(context.Background(), tc.inputID, "userID")
			require.Equal(t, tc.expectedErr, err)
		})
	}
}

func mockMovement(description string, statusID int, transactionID *uuid.UUID) model.Movement {
	return model.Movement{
		Description:   description,
		TransactionID: transactionID,
		StatusID:      statusID,
	}
}

func mockTransaction(
	transactionID *uuid.UUID,
	estimate *model.Movement,
	consolidation *model.Consolidation,
	doneList model.MovementList,
) model.Transaction {
	return model.Transaction{
		TransactionID: transactionID,
		Estimate:      estimate,
		Consolidation: consolidation,
		DoneList:      doneList,
	}
}
