package service_test

import (
	"context"
	"errors"
	"fmt"
	"personal-finance/internal/model/eager"
	"testing"
	"time"

	"personal-finance/internal/domain/transaction/repository"
	"personal-finance/internal/domain/transaction/service"
	"personal-finance/internal/model"

	"github.com/stretchr/testify/require"
)

var (
	now              = time.Now()
	transactionsMock = []model.Transaction{
		{
			ID:            1,
			Description:   "Aluguel",
			Amount:        1000.0,
			Date:          time.Date(2022, time.September, 01, 0, 0, 0, 0, time.Local),
			WalletID:      1,
			TypePaymentID: 1,
			CategoryID:    2,
			DateCreate:    now,
			DateUpdate:    now,
		},
		{
			ID:            2,
			Description:   "Energia",
			Amount:        300.0,
			Date:          time.Date(2022, time.September, 15, 0, 0, 0, 0, time.Local),
			WalletID:      1,
			TypePaymentID: 1,
			CategoryID:    2,
			DateCreate:    now,
			DateUpdate:    now,
		},
		{
			ID:            3,
			Description:   "Agua",
			Amount:        120.0,
			Date:          time.Date(2022, time.September, 30, 0, 0, 0, 0, time.Local),
			WalletID:      1,
			TypePaymentID: 1,
			CategoryID:    2,
			DateCreate:    now,
			DateUpdate:    now,
		},
	}
	transactionEagerMock = eager.Transaction{
		ID:          1,
		Description: "Aluguel",
		Amount:      1000.0,
		Date:        now,
		WalletID:    0,
		Wallet: model.Wallet{
			ID:          1,
			Description: "Alimentacao",
			Balance:     0,
			DateCreate:  now,
			DateUpdate:  now,
		},
		TypePaymentID: 0,
		TypePayment: model.TypePayment{
			ID:          1,
			Description: "Débito",
			DateCreate:  now,
			DateUpdate:  now,
		},
		CategoryID: 0,
		Category: model.Category{
			ID:          2,
			Description: "Casa",
			DateCreate:  now,
			DateUpdate:  now,
		},
		DateCreate: now,
		DateUpdate: now,
	}
)

func TestService_Add(t *testing.T) {
	tt := []struct {
		name                string
		inputTransaction    model.Transaction
		MockedTransaction   model.Transaction
		expectedTransaction model.Transaction
		MockedError         error
		expectedErr         error
	}{
		{
			name: "Success",
			inputTransaction: model.Transaction{
				Description: transactionsMock[0].Description,
			},
			MockedTransaction:   transactionsMock[0],
			expectedTransaction: transactionsMock[0],
			MockedError:         nil,
			expectedErr:         nil,
		}, {
			name: "repository error",
			inputTransaction: model.Transaction{
				Description: transactionsMock[0].Description,
			},
			MockedTransaction:   model.Transaction{},
			expectedTransaction: model.Transaction{},
			MockedError:         errors.New("repository error"),
			expectedErr:         fmt.Errorf("error to add transactions: %w", errors.New("repository error")),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			repoMock := &repository.Mock{}
			repoMock.On("Add", tc.inputTransaction).
				Return(tc.MockedTransaction, tc.MockedError)

			svc := service.NewTransactionService(repoMock)

			result, err := svc.Add(context.Background(), tc.inputTransaction)
			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.expectedTransaction, result)
		})
	}
}

func TestService_FindAll(t *testing.T) {
	tt := []struct {
		name               string
		expectedCategories []model.Transaction
		mockedError        error
		expectedErr        error
	}{
		{
			name:               "Success",
			expectedCategories: transactionsMock,
			mockedError:        nil,
			expectedErr:        nil,
		},
		{
			name:               "no cars found",
			expectedCategories: []model.Transaction{},
			mockedError:        errors.New("repository error"),
			expectedErr:        fmt.Errorf("error to find transactions: %w", errors.New("repository error")),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			repoMock := repository.Mock{}
			repoMock.On("FindAll").
				Return(tc.expectedCategories, tc.mockedError)
			svc := service.NewTransactionService(&repoMock)

			result, err := svc.FindAll(context.Background())
			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.expectedCategories, result)
		})
	}
}

func TestService_FindByMonth(t *testing.T) {
	tt := []struct {
		name                 string
		mockedTransactions   []model.Transaction
		expectedTransactions []model.Transaction
		mockedError          error
		expectedErr          error
	}{
		{
			name: "Success",
			mockedTransactions: []model.Transaction{
				transactionsMock[0],
				transactionsMock[1],
			},
			expectedTransactions: transactionsMock,
			mockedError:          nil,
			expectedErr:          nil,
		},
		{
			name:                 "no cars found",
			mockedTransactions:   []model.Transaction{},
			expectedTransactions: []model.Transaction{},
			mockedError:          errors.New("repository error"),
			expectedErr:          fmt.Errorf("error to find transactions: %w", errors.New("repository error")),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			repoMock := repository.Mock{}
			repoMock.On("FindByMonth").
				Return(tc.expectedTransactions, tc.mockedError)
			svc := service.NewTransactionService(&repoMock)

			result, err := svc.FindByMonth(context.Background(), transactionsMock[0].Date, transactionsMock[1].Date)
			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.expectedTransactions, result)
		})
	}
}

func TestService_FindByID(t *testing.T) {
	tt := []struct {
		name                string
		inputID             int
		expectedTransaction model.Transaction
		mockedError         error
		expectedErr         error
	}{
		{
			name:                "Success",
			inputID:             1,
			expectedTransaction: transactionsMock[0],
			mockedError:         nil,
			expectedErr:         nil,
		},
		{
			name:                "no transactions found",
			inputID:             0,
			expectedTransaction: model.Transaction{},
			mockedError:         errors.New("repository error"),
			expectedErr:         fmt.Errorf("error to find transactions: %w", errors.New("repository error")),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			repoMock := repository.Mock{}
			repoMock.On("FindByID").
				Return(tc.expectedTransaction, tc.mockedError)
			svc := service.NewTransactionService(&repoMock)

			result, err := svc.FindByID(context.Background(), tc.inputID)
			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.expectedTransaction, result)
		})
	}
}

func TestService_Update(t *testing.T) {
	tt := []struct {
		name                string
		inputTransaction    model.Transaction
		mockedTransaction   model.Transaction
		expectedTransaction model.Transaction
		inputID             int
		mockedError         error
		expectedErr         error
	}{
		{
			name: "Success",
			inputTransaction: model.Transaction{
				Description: transactionsMock[1].Description,
			},
			mockedTransaction: model.Transaction{
				ID:          transactionsMock[0].ID,
				Description: transactionsMock[1].Description,
				DateCreate:  transactionsMock[0].DateCreate,
			},
			expectedTransaction: model.Transaction{
				ID:          transactionsMock[0].ID,
				Description: transactionsMock[1].Description,
				DateCreate:  transactionsMock[0].DateCreate,
			},
			inputID:     1,
			mockedError: nil,
			expectedErr: nil,
		},
		{
			name: "repository error",
			inputTransaction: model.Transaction{
				Description: transactionsMock[1].Description,
			},
			mockedTransaction:   model.Transaction{},
			expectedTransaction: model.Transaction{},
			inputID:             1,
			mockedError:         errors.New("repository error"),
			expectedErr:         fmt.Errorf("error updating transactions: %w", errors.New("repository error")),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			repoMock := repository.Mock{}
			repoMock.On("Update").
				Return(tc.mockedTransaction, tc.mockedError)

			svc := service.NewTransactionService(&repoMock)

			result, err := svc.Update(context.Background(), tc.inputID, tc.inputTransaction)
			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.expectedTransaction, result)
		})
	}
}

func TestService_Delete(t *testing.T) {
	tt := []struct {
		name        string
		inputID     int
		mockedErr   error
		expectedErr error
	}{
		{
			name:        "Success",
			inputID:     1,
			mockedErr:   nil,
			expectedErr: nil,
		},
		{
			name:        "fail",
			inputID:     1,
			mockedErr:   errors.New("repository error"),
			expectedErr: fmt.Errorf("error deleting transactions: %w", errors.New("repository error")),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			repoMock := repository.Mock{}
			repoMock.On("Delete").
				Return(tc.mockedErr)
			svc := service.NewTransactionService(&repoMock)

			err := svc.Delete(context.Background(), tc.inputID)
			require.Equal(t, tc.expectedErr, err)
		})
	}
}
