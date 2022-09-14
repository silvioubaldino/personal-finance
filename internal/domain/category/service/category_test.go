package service_test

import (
	"context"
	"errors"
	"fmt"
	"personal-finance/internal/domain/category/repository"
	"personal-finance/internal/domain/category/service"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"personal-finance/internal/model"
)

var (
	now            = time.Now()
	categoriesMock = []model.Category{
		{
			ID:          1,
			Description: "Alimentacao",
			DateCreate:  now,
			DateUpdate:  now,
		},
		{
			ID:          2,
			Description: "Casa",
			DateCreate:  now,
			DateUpdate:  now,
		},
		{
			ID:          3,
			Description: "Carro",
			DateCreate:  now,
			DateUpdate:  now,
		},
	}
)

func TestService_Add(t *testing.T) {
	tt := []struct {
		name             string
		inputCategory    model.Category
		MockedCategory   model.Category
		expectedCategory model.Category
		MockedError      error
		expectedErr      error
	}{
		{
			name: "Success",
			inputCategory: model.Category{
				Description: categoriesMock[0].Description,
			},
			MockedCategory:   categoriesMock[0],
			expectedCategory: categoriesMock[0],
			MockedError:      nil,
			expectedErr:      nil,
		}, {
			name: "repository error",
			inputCategory: model.Category{
				Description: categoriesMock[0].Description,
			},
			MockedCategory:   model.Category{},
			expectedCategory: model.Category{},
			MockedError:      errors.New("repository error"),
			expectedErr:      fmt.Errorf("error to add categories: %w", errors.New("repository error")),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			repoMock := &repository.Mock{}
			repoMock.On("Add", tc.inputCategory).
				Return(tc.MockedCategory, tc.MockedError)

			svc := service.NewCategoryService(repoMock)

			result, err := svc.Add(context.Background(), tc.inputCategory)
			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.expectedCategory, result)
		})
	}
}

func TestService_FindAll(t *testing.T) {
	tt := []struct {
		name               string
		expectedCategories []model.Category
		mockedError        error
		expectedErr        error
	}{
		{
			name:               "Success",
			expectedCategories: categoriesMock,
			mockedError:        nil,
			expectedErr:        nil,
		},
		{
			name:               "no cars found",
			expectedCategories: []model.Category{},
			mockedError:        errors.New("repository error"),
			expectedErr:        fmt.Errorf("error to find categories: %w", errors.New("repository error")),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			repoMock := repository.Mock{}
			repoMock.On("FindAll").
				Return(tc.expectedCategories, tc.mockedError)
			svc := service.NewCategoryService(&repoMock)

			result, err := svc.FindAll(context.Background())
			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.expectedCategories, result)
		})
	}
}

func TestService_FindByID(t *testing.T) {
	tt := []struct {
		name             string
		inputID          int
		expectedCategory model.Category
		mockedError      error
		expectedErr      error
	}{
		{
			name:             "Success",
			inputID:          1,
			expectedCategory: categoriesMock[0],
			mockedError:      nil,
			expectedErr:      nil,
		},
		{
			name:             "no categories found",
			inputID:          0,
			expectedCategory: model.Category{},
			mockedError:      errors.New("repository error"),
			expectedErr:      fmt.Errorf("error to find categories: %w", errors.New("repository error")),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			repoMock := repository.Mock{}
			repoMock.On("FindByID").
				Return(tc.expectedCategory, tc.mockedError)
			svc := service.NewCategoryService(&repoMock)

			result, err := svc.FindByID(context.Background(), tc.inputID)
			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.expectedCategory, result)
		})
	}
}

func TestService_Update(t *testing.T) {
	tt := []struct {
		name             string
		inputCategory    model.Category
		mockedCategory   model.Category
		expectedCategory model.Category
		inputID          int
		mockedError      error
		expectedErr      error
	}{
		{
			name: "Success",
			inputCategory: model.Category{
				Description: categoriesMock[1].Description,
			},
			mockedCategory: model.Category{
				ID:          categoriesMock[0].ID,
				Description: categoriesMock[1].Description,
				DateCreate:  categoriesMock[0].DateCreate,
			},
			expectedCategory: model.Category{
				ID:          categoriesMock[0].ID,
				Description: categoriesMock[1].Description,
				DateCreate:  categoriesMock[0].DateCreate,
			},
			inputID:     1,
			mockedError: nil,
			expectedErr: nil,
		},
		{
			name: "repository error",
			inputCategory: model.Category{
				Description: categoriesMock[1].Description,
			},
			mockedCategory:   model.Category{},
			expectedCategory: model.Category{},
			inputID:          1,
			mockedError:      errors.New("repository error"),
			expectedErr:      fmt.Errorf("error updating categories: %w", errors.New("repository error")),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			repoMock := repository.Mock{}
			repoMock.On("Update").
				Return(tc.mockedCategory, tc.mockedError)

			svc := service.NewCategoryService(&repoMock)

			result, err := svc.Update(context.Background(), tc.inputID, tc.inputCategory)
			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.expectedCategory, result)
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
			expectedErr: fmt.Errorf("error deleting categories: %w", errors.New("repository error")),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			repoMock := repository.Mock{}
			repoMock.On("Delete").
				Return(tc.mockedErr)
			svc := service.NewCategoryService(&repoMock)

			err := svc.Delete(context.Background(), tc.inputID)
			require.Equal(t, tc.expectedErr, err)
		})
	}
}
