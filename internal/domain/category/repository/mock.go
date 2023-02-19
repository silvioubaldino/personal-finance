package repository

import (
	"context"

	"personal-finance/internal/model"

	"github.com/stretchr/testify/mock"
)

type Mock struct {
	mock.Mock
}

func (m *Mock) Add(_ context.Context, category model.Category, userID string) (model.Category, error) {
	args := m.Called(category, userID)
	return args.Get(0).(model.Category), args.Error(1)
}

func (m *Mock) FindAll(_ context.Context, userID string) ([]model.Category, error) {
	args := m.Called(userID)
	return args.Get(0).([]model.Category), args.Error(1)
}

func (m *Mock) FindByID(_ context.Context, id int, userID string) (model.Category, error) {
	args := m.Called(id, userID)
	return args.Get(0).(model.Category), args.Error(1)
}

func (m *Mock) Update(_ context.Context, id int, category model.Category, userID string) (model.Category, error) {
	args := m.Called(id, category, userID)
	return args.Get(0).(model.Category), args.Error(1)
}

func (m *Mock) Delete(_ context.Context, _ int) error {
	args := m.Called()
	return args.Error(0)
}
