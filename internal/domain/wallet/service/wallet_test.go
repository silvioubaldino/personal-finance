package service_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"personal-finance/internal/domain/wallet/repository"
	"personal-finance/internal/domain/wallet/service"
	"personal-finance/internal/model"

	"github.com/stretchr/testify/require"
)

var (
	now         = time.Now()
	walletsMock = []model.Wallet{
		{
			ID:          1,
			Description: "Alimentacao",
			Balance:     0,
			UserID:      "userID",
			DateCreate:  now,
			DateUpdate:  now,
		},
		{
			ID:          2,
			Description: "Casa",
			Balance:     0,
			UserID:      "userID",
			DateCreate:  now,
			DateUpdate:  now,
		},
		{
			ID:          3,
			Description: "Carro",
			Balance:     0,
			UserID:      "userID",
			DateCreate:  now,
			DateUpdate:  now,
		},
	}
)

func TestService_Add(t *testing.T) {
	tt := []struct {
		name           string
		inputWallet    model.Wallet
		MockedWallet   model.Wallet
		expectedWallet model.Wallet
		MockedError    error
		expectedErr    error
	}{
		{
			name: "Success",
			inputWallet: model.Wallet{
				Description: walletsMock[0].Description,
			},
			MockedWallet:   walletsMock[0],
			expectedWallet: walletsMock[0],
			MockedError:    nil,
			expectedErr:    nil,
		}, {
			name: "repository error",
			inputWallet: model.Wallet{
				Description: walletsMock[0].Description,
			},
			MockedWallet:   model.Wallet{},
			expectedWallet: model.Wallet{},
			MockedError:    errors.New("repository error"),
			expectedErr:    fmt.Errorf("error to add wallets: %w", errors.New("repository error")),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			repoMock := &repository.Mock{}
			repoMock.On("Add", tc.inputWallet, "userID").
				Return(tc.MockedWallet, tc.MockedError)

			svc := service.NewWalletService(repoMock)

			result, err := svc.Add(context.Background(), tc.inputWallet, "userID")
			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.expectedWallet, result)
		})
	}
}

func TestService_FindAll(t *testing.T) {
	tt := []struct {
		name               string
		expectedCategories []model.Wallet
		mockedError        error
		expectedErr        error
	}{
		{
			name:               "Success",
			expectedCategories: walletsMock,
			mockedError:        nil,
			expectedErr:        nil,
		},
		{
			name:               "no cars found",
			expectedCategories: []model.Wallet{},
			mockedError:        errors.New("repository error"),
			expectedErr:        fmt.Errorf("error to find wallets: %w", errors.New("repository error")),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			repoMock := repository.Mock{}
			repoMock.On("FindAll", "userID").
				Return(tc.expectedCategories, tc.mockedError)
			svc := service.NewWalletService(&repoMock)

			result, err := svc.FindAll(context.Background(), "userID")
			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.expectedCategories, result)
		})
	}
}

func TestService_FindByID(t *testing.T) {
	tt := []struct {
		name           string
		inputID        int
		expectedWallet model.Wallet
		mockedError    error
		expectedErr    error
	}{
		{
			name:           "Success",
			inputID:        1,
			expectedWallet: walletsMock[0],
			mockedError:    nil,
			expectedErr:    nil,
		},
		{
			name:           "no wallets found",
			inputID:        0,
			expectedWallet: model.Wallet{},
			mockedError:    errors.New("repository error"),
			expectedErr:    fmt.Errorf("error to find wallets: %w", errors.New("repository error")),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			repoMock := repository.Mock{}
			repoMock.On("FindByID", tc.inputID, "userID").
				Return(tc.expectedWallet, tc.mockedError)
			svc := service.NewWalletService(&repoMock)

			result, err := svc.FindByID(context.Background(), tc.inputID, "userID")
			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.expectedWallet, result)
		})
	}
}

func TestService_Update(t *testing.T) {
	tt := []struct {
		name           string
		inputWallet    model.Wallet
		mockedWallet   model.Wallet
		expectedWallet model.Wallet
		inputID        int
		mockedError    error
		expectedErr    error
	}{
		{
			name: "Success",
			inputWallet: model.Wallet{
				Description: walletsMock[1].Description,
			},
			mockedWallet: model.Wallet{
				ID:          walletsMock[0].ID,
				Description: walletsMock[1].Description,
				DateCreate:  walletsMock[0].DateCreate,
			},
			expectedWallet: model.Wallet{
				ID:          walletsMock[0].ID,
				Description: walletsMock[1].Description,
				DateCreate:  walletsMock[0].DateCreate,
			},
			inputID:     1,
			mockedError: nil,
			expectedErr: nil,
		},
		{
			name: "repository error",
			inputWallet: model.Wallet{
				Description: walletsMock[1].Description,
			},
			mockedWallet:   model.Wallet{},
			expectedWallet: model.Wallet{},
			inputID:        1,
			mockedError:    errors.New("repository error"),
			expectedErr:    fmt.Errorf("error updating wallets: %w", errors.New("repository error")),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			repoMock := repository.Mock{}
			repoMock.On("Update", tc.inputID, tc.inputWallet, "userID").
				Return(tc.mockedWallet, tc.mockedError)

			svc := service.NewWalletService(&repoMock)

			result, err := svc.Update(context.Background(), tc.inputID, tc.inputWallet, "userID")
			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.expectedWallet, result)
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
			expectedErr: fmt.Errorf("error deleting wallets: %w", errors.New("repository error")),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			repoMock := repository.Mock{}
			repoMock.On("Delete").
				Return(tc.mockedErr)
			svc := service.NewWalletService(&repoMock)

			err := svc.Delete(context.Background(), tc.inputID)
			require.Equal(t, tc.expectedErr, err)
		})
	}
}
