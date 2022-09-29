package repository

import (
	"context"

	"personal-finance/internal/model"

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

func (m *Mock) FindByID(_ context.Context, _ int) (model.Category, error) {
	args := m.Called()
	return args.Get(0).(model.Category), args.Error(1)
}

func (m *Mock) Update(_ context.Context, _ int, _ model.Category) (model.Category, error) {
	args := m.Called()
	return args.Get(0).(model.Category), args.Error(1)
}

func (m *Mock) Delete(_ context.Context, _ int) error {
	args := m.Called()
	return args.Error(0)
}
