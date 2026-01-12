package repository

import (
	"context"

	"personal-finance/internal/model"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type Mock struct {
	mock.Mock
}

func (m *Mock) Add(_ context.Context, category model.Category) (model.Category, error) {
	args := m.Called(category)
	return args.Get(0).(model.Category), args.Error(1)
}

func (m *Mock) FindAll(_ context.Context) ([]model.Category, error) {
	args := m.Called()
	return args.Get(0).([]model.Category), args.Error(1)
}

func (m *Mock) FindByID(_ context.Context, id uuid.UUID) (model.Category, error) {
	args := m.Called(id)
	return args.Get(0).(model.Category), args.Error(1)
}

func (m *Mock) Update(_ context.Context, id uuid.UUID, category model.Category) (model.Category, error) {
	args := m.Called(id, category)
	return args.Get(0).(model.Category), args.Error(1)
}

func (m *Mock) Delete(_ context.Context, id uuid.UUID) error {
	args := m.Called(id)
	return args.Error(0)
}
