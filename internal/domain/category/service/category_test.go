package service_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"personal-finance/internal/domain/category/repository"
	"personal-finance/internal/domain/category/service"

	"personal-finance/internal/model"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

var (
	now   = time.Now()
	uuid1 = uuid.New()
	uuid2 = uuid.New()
	uuid3 = uuid.New()

	categoriesMock = []model.Category{
		{
			ID:          &uuid1,
			Description: "Alimentacao",
			DateCreate:  now,
			DateUpdate:  now,
		},
		{
			ID:          &uuid2,
			Description: "Casa",
			DateCreate:  now,
			DateUpdate:  now,
		},
		{
			ID:          &uuid3,
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
		{
			name: "invalid color",
			inputCategory: model.Category{
				Description: categoriesMock[0].Description,
				Color:       "invalid",
			},
			MockedCategory:   model.Category{},
			expectedCategory: model.Category{},
			MockedError:      nil,
			expectedErr:      fmt.Errorf("error to add categories: %w", errors.New("invalid color format")),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			repoMock := &repository.Mock{}
			// Only expect call if validation passes (expectedErr is nil or repository error)
			if tc.name != "invalid color" {
				repoMock.On("Add", tc.inputCategory).
					Return(tc.MockedCategory, tc.MockedError)
			}

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
		inputID          uuid.UUID
		expectedCategory model.Category
		mockedError      error
		expectedErr      error
	}{
		{
			name:             "Success",
			inputID:          uuid1,
			expectedCategory: categoriesMock[0],
			mockedError:      nil,
			expectedErr:      nil,
		},
		{
			name:             "no categories found",
			inputID:          uuid.Nil,
			expectedCategory: model.Category{},
			mockedError:      errors.New("repository error"),
			expectedErr:      fmt.Errorf("error to find categories: %w", errors.New("repository error")),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			repoMock := repository.Mock{}
			repoMock.On("FindByID", tc.inputID).
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
		inputID          uuid.UUID
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
			inputID:     uuid1,
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
			inputID:          uuid1,
			mockedError:      errors.New("repository error"),
			expectedErr:      fmt.Errorf("error updating categories: %w", errors.New("repository error")),
		},
		{
			name: "invalid color",
			inputCategory: model.Category{
				Description: categoriesMock[1].Description,
				Color:       "invalid",
			},
			mockedCategory:   model.Category{},
			expectedCategory: model.Category{},
			inputID:          uuid1,
			mockedError:      nil,
			expectedErr:      fmt.Errorf("error updating categories: %w", errors.New("invalid color format")),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			repoMock := &repository.Mock{}
			if tc.name != "invalid color" {
				repoMock.On("Update", tc.inputID, tc.inputCategory).
					Return(tc.mockedCategory, tc.mockedError)
			}

			svc := service.NewCategoryService(repoMock)

			result, err := svc.Update(context.Background(), tc.inputID, tc.inputCategory)
			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.expectedCategory, result)
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
			inputID:     uuid1,
			mockedErr:   nil,
			expectedErr: nil,
		},
		{
			name:        "fail",
			inputID:     uuid1,
			mockedErr:   errors.New("repository error"),
			expectedErr: fmt.Errorf("error deleting categories: %w", errors.New("repository error")),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			repoMock := repository.Mock{}
			repoMock.On("Delete", tc.inputID).
				Return(tc.mockedErr)
			svc := service.NewCategoryService(&repoMock)

			err := svc.Delete(context.Background(), tc.inputID)
			require.Equal(t, tc.expectedErr, err)
		})
	}
}
