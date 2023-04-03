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

	walletSvc "personal-finance/internal/domain/wallet/service"

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
	}{
		{
			name: "Success",
			inputTransaction: model.Movement{
				Description: movementsMock[0].Description,
				StatusID:    2,
			},
			MockedTransaction:   movementsMock[0],
			expectedTransaction: movementsMock[0],
			MockedError:         nil,
			expectedErr:         nil,
		}, {
			name: "repository error",
			inputTransaction: model.Movement{
				Description: movementsMock[0].Description,
				StatusID:    2,
			},
			MockedTransaction:   model.Movement{},
			expectedTransaction: model.Movement{},
			MockedError:         errors.New("repository error"),
			expectedErr:         fmt.Errorf("error to add transactions: %w", errors.New("repository error")),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			repoMock := &repository.Mock{}
			walletSvcMock := &walletSvc.Mock{}
			repoMock.On("Add", tc.inputTransaction, "userID").
				Return(tc.MockedTransaction, tc.MockedError)

			svc := service.NewMovementService(repoMock, walletSvcMock)

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
			walletSvcMock := &walletSvc.Mock{}
			repoMock.On("FindByPeriod", tc.inputPeriod, "userID").
				Return(tc.expectedTransactions, tc.mockedError)
			svc := service.NewMovementService(repoMock, walletSvcMock)

			result, err := svc.FindByPeriod(context.Background(), tc.inputPeriod, "userID")
			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.expectedTransactions, result)
		})
	}
}

func TestService_BalanceByPeriod(t *testing.T) {
	tt := []struct {
		name               string
		inputPeriod        model.Period
		mockedTransactions []model.Movement
		expectedBalance    model.Balance
		mockedError        error
		expectedErr        error
	}{
		{
			name: "Success - zero income",
			inputPeriod: model.Period{
				From: *movementsMock[0].Date,
				To:   *movementsMock[1].Date,
			},
			mockedTransactions: []model.Movement{
				movementsMock[0],
				movementsMock[1],
			},
			expectedBalance: model.Balance{
				Period: model.Period{
					From: *movementsMock[0].Date,
					To:   *movementsMock[1].Date,
				},
				Expense: -1300.0,
			},
			mockedError: nil,
			expectedErr: nil,
		},
		{
			name: "Success - zero expense",
			inputPeriod: model.Period{
				From: *movementsMock[0].Date,
				To:   *movementsMock[1].Date,
			},
			mockedTransactions: []model.Movement{
				movementsMock[2],
			},
			expectedBalance: model.Balance{
				Period: model.Period{
					From: *movementsMock[0].Date,
					To:   *movementsMock[1].Date,
				},
				Income: 120.0,
			},
			mockedError: nil,
			expectedErr: nil,
		},
		{
			name:               "repository error",
			mockedTransactions: []model.Movement{},
			expectedBalance:    model.Balance{},
			mockedError:        errors.New("repository error"),
			expectedErr:        fmt.Errorf("error to find transactions: %w", errors.New("repository error")),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			repoMock := &repository.Mock{}
			walletSvcMock := &walletSvc.Mock{}
			repoMock.On("FindByPeriod", tc.inputPeriod, "userID").
				Return(tc.mockedTransactions, tc.mockedError)
			repoMock.On("BalanceByPeriod", tc.inputPeriod, "userID").
				Return(tc.expectedBalance, tc.mockedError)
			svc := service.NewMovementService(repoMock, walletSvcMock)

			result, err := svc.BalanceByPeriod(context.Background(), tc.inputPeriod, "userID")
			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.expectedBalance, result)
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
			walletSvcMock := &walletSvc.Mock{}
			repoMock.On("FindByID", tc.inputID, "userID").
				Return(tc.expectedTransaction, tc.mockedError)
			svc := service.NewMovementService(repoMock, walletSvcMock)

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
			walletSvcMock := &walletSvc.Mock{}
			repoMock.On("Update", tc.inputID, tc.inputTransaction, "userID").
				Return(tc.mockedTransaction, tc.mockedError)

			svc := service.NewMovementService(repoMock, walletSvcMock)

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
			walletSvcMock := &walletSvc.Mock{}
			repoMock.On("Delete", tc.inputID, "userID").
				Return(tc.mockedErr)
			svc := service.NewMovementService(repoMock, walletSvcMock)

			err := svc.Delete(context.Background(), tc.inputID, "userID")
			require.Equal(t, tc.expectedErr, err)
		})
	}
}
