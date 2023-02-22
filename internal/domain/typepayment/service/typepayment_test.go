package service_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"personal-finance/internal/domain/typepayment/repository"
	"personal-finance/internal/domain/typepayment/service"
	"personal-finance/internal/model"

	"github.com/stretchr/testify/require"
)

var (
	now              = time.Now()
	typePaymentsMock = []model.TypePayment{
		{
			ID:          1,
			Description: "Débito",
			DateCreate:  now,
			DateUpdate:  now,
		},
		{
			ID:          2,
			Description: "Crédito",
			DateCreate:  now,
			DateUpdate:  now,
		},
		{
			ID:          3,
			Description: "Pix",
			DateCreate:  now,
			DateUpdate:  now,
		},
	}
)

func TestService_Add(t *testing.T) {
	tt := []struct {
		name                string
		inputTypePayment    model.TypePayment
		MockedTypePayment   model.TypePayment
		expectedTypePayment model.TypePayment
		MockedError         error
		expectedErr         error
	}{
		{
			name: "Success",
			inputTypePayment: model.TypePayment{
				Description: typePaymentsMock[0].Description,
			},
			MockedTypePayment:   typePaymentsMock[0],
			expectedTypePayment: typePaymentsMock[0],
			MockedError:         nil,
			expectedErr:         nil,
		}, {
			name: "repository error",
			inputTypePayment: model.TypePayment{
				Description: typePaymentsMock[0].Description,
			},
			MockedTypePayment:   model.TypePayment{},
			expectedTypePayment: model.TypePayment{},
			MockedError:         errors.New("repository error"),
			expectedErr:         fmt.Errorf("error to add typePayments: %w", errors.New("repository error")),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			repoMock := &repository.Mock{}
			repoMock.On("Add", tc.inputTypePayment, "userID").
				Return(tc.MockedTypePayment, tc.MockedError)

			svc := service.NewTypePaymentService(repoMock)

			result, err := svc.Add(context.Background(), tc.inputTypePayment, "userID")
			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.expectedTypePayment, result)
		})
	}
}

func TestService_FindAll(t *testing.T) {
	tt := []struct {
		name               string
		expectedCategories []model.TypePayment
		mockedError        error
		expectedErr        error
	}{
		{
			name:               "Success",
			expectedCategories: typePaymentsMock,
			mockedError:        nil,
			expectedErr:        nil,
		},
		{
			name:               "no cars found",
			expectedCategories: []model.TypePayment{},
			mockedError:        errors.New("repository error"),
			expectedErr:        fmt.Errorf("error to find typePayments: %w", errors.New("repository error")),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			repoMock := repository.Mock{}
			repoMock.On("FindAll", "userID").
				Return(tc.expectedCategories, tc.mockedError)
			svc := service.NewTypePaymentService(&repoMock)

			result, err := svc.FindAll(context.Background(), "userID")
			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.expectedCategories, result)
		})
	}
}

func TestService_FindByID(t *testing.T) {
	tt := []struct {
		name                string
		inputID             int
		expectedTypePayment model.TypePayment
		mockedError         error
		expectedErr         error
	}{
		{
			name:                "Success",
			inputID:             1,
			expectedTypePayment: typePaymentsMock[0],
			mockedError:         nil,
			expectedErr:         nil,
		},
		{
			name:                "no typePayments found",
			inputID:             0,
			expectedTypePayment: model.TypePayment{},
			mockedError:         errors.New("repository error"),
			expectedErr:         fmt.Errorf("error to find typePayments: %w", errors.New("repository error")),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			repoMock := repository.Mock{}
			repoMock.On("FindByID", tc.inputID, "userID").
				Return(tc.expectedTypePayment, tc.mockedError)
			svc := service.NewTypePaymentService(&repoMock)

			result, err := svc.FindByID(context.Background(), tc.inputID, "userID")
			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.expectedTypePayment, result)
		})
	}
}

func TestService_Update(t *testing.T) {
	tt := []struct {
		name                string
		inputTypePayment    model.TypePayment
		mockedTypePayment   model.TypePayment
		expectedTypePayment model.TypePayment
		inputID             int
		mockedError         error
		expectedErr         error
	}{
		{
			name: "Success",
			inputTypePayment: model.TypePayment{
				Description: typePaymentsMock[1].Description,
			},
			mockedTypePayment: model.TypePayment{
				ID:          typePaymentsMock[0].ID,
				Description: typePaymentsMock[1].Description,
				DateCreate:  typePaymentsMock[0].DateCreate,
			},
			expectedTypePayment: model.TypePayment{
				ID:          typePaymentsMock[0].ID,
				Description: typePaymentsMock[1].Description,
				DateCreate:  typePaymentsMock[0].DateCreate,
			},
			inputID:     1,
			mockedError: nil,
			expectedErr: nil,
		},
		{
			name: "repository error",
			inputTypePayment: model.TypePayment{
				Description: typePaymentsMock[1].Description,
			},
			mockedTypePayment:   model.TypePayment{},
			expectedTypePayment: model.TypePayment{},
			inputID:             1,
			mockedError:         errors.New("repository error"),
			expectedErr:         fmt.Errorf("error updating typePayments: %w", errors.New("repository error")),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			repoMock := repository.Mock{}
			repoMock.On("Update", tc.inputID, tc.inputTypePayment, "userID").
				Return(tc.mockedTypePayment, tc.mockedError)

			svc := service.NewTypePaymentService(&repoMock)

			result, err := svc.Update(context.Background(), tc.inputID, tc.inputTypePayment, "userID")
			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.expectedTypePayment, result)
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
			expectedErr: fmt.Errorf("error deleting typePayments: %w", errors.New("repository error")),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			repoMock := repository.Mock{}
			repoMock.On("Delete").
				Return(tc.mockedErr)
			svc := service.NewTypePaymentService(&repoMock)

			err := svc.Delete(context.Background(), tc.inputID)
			require.Equal(t, tc.expectedErr, err)
		})
	}
}
