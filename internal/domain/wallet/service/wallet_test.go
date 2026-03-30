package service_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"personal-finance/internal/domain/wallet/repository"
	"personal-finance/internal/domain/wallet/service"
	"personal-finance/internal/model"
)

var (
	id1 = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	id2 = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	id3 = uuid.MustParse("00000000-0000-0000-0000-000000000003")

	now         = time.Now()
	walletsMock = []model.Wallet{
		{
			ID:          &id1,
			Description: "Alimentacao",
			Balance:     0,
			UserID:      "userID",
			DateCreate:  now,
			DateUpdate:  now,
		},
		{
			ID:          &id2,
			Description: "Casa",
			Balance:     0,
			UserID:      "userID",
			DateCreate:  now,
			DateUpdate:  now,
		},
		{
			ID:          &id3,
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
			repoMock.On("Add", tc.inputWallet).
				Return(tc.MockedWallet, tc.MockedError)

			svc := service.NewWalletService(repoMock, nil)

			result, err := svc.Add(context.Background(), tc.inputWallet)
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
			name:               "no wallets found",
			expectedCategories: []model.Wallet{},
			mockedError:        errors.New("repository error"),
			expectedErr:        fmt.Errorf("error to find wallets: %w", errors.New("repository error")),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			repoMock := repository.Mock{}
			repoMock.On("FindAll").
				Return(tc.expectedCategories, tc.mockedError)
			svc := service.NewWalletService(&repoMock, nil)

			result, err := svc.FindAll(context.Background())
			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.expectedCategories, result)
		})
	}
}

func TestService_FindByID(t *testing.T) {
	tt := []struct {
		name           string
		inputID        *uuid.UUID
		expectedWallet model.Wallet
		mockedError    error
		expectedErr    error
	}{
		{
			name:           "Success",
			inputID:        &id1,
			expectedWallet: walletsMock[0],
			mockedError:    nil,
			expectedErr:    nil,
		},
		{
			name:           "no wallets found",
			inputID:        &id1,
			expectedWallet: model.Wallet{},
			mockedError:    errors.New("repository error"),
			expectedErr:    fmt.Errorf("error to find wallets: %w", errors.New("repository error")),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			repoMock := repository.Mock{}
			repoMock.On("FindByID", tc.inputID).
				Return(tc.expectedWallet, tc.mockedError)
			svc := service.NewWalletService(&repoMock, nil)

			result, err := svc.FindByID(context.Background(), tc.inputID)
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
		inputID        *uuid.UUID
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
			inputID:     &id1,
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
			inputID:        &id1,
			mockedError:    errors.New("repository error"),
			expectedErr:    fmt.Errorf("error updating wallets: %w", errors.New("repository error")),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			repoMock := repository.Mock{}
			repoMock.On("Update", tc.inputID, tc.inputWallet).
				Return(tc.mockedWallet, tc.mockedError)

			svc := service.NewWalletService(&repoMock, nil)

			result, err := svc.Update(context.Background(), tc.inputID, tc.inputWallet)
			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.expectedWallet, result)
		})
	}
}

func TestService_Delete(t *testing.T) {
	tt := []struct {
		name        string
		inputID     *uuid.UUID
		mockedErr   error
		expectedErr error
	}{
		{
			name:        "Success",
			inputID:     &id1,
			mockedErr:   nil,
			expectedErr: nil,
		},
		{
			name:        "fail",
			inputID:     &id1,
			mockedErr:   errors.New("repository error"),
			expectedErr: fmt.Errorf("error deleting wallets: %w", errors.New("repository error")),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			repoMock := repository.Mock{}
			repoMock.On("Delete", tc.inputID).
				Return(tc.mockedErr)
			svc := service.NewWalletService(&repoMock, nil)

			err := svc.Delete(context.Background(), tc.inputID)
			require.Equal(t, tc.expectedErr, err)
		})
	}
}
