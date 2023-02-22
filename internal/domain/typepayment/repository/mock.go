package repository

import (
	"context"

	"personal-finance/internal/model"

	"github.com/stretchr/testify/mock"
)

type Mock struct {
	mock.Mock
}

func (m *Mock) Add(_ context.Context, typePayment model.TypePayment, userID string) (model.TypePayment, error) {
	args := m.Called(typePayment, userID)
	return args.Get(0).(model.TypePayment), args.Error(1)
}

func (m *Mock) FindAll(_ context.Context, userID string) ([]model.TypePayment, error) {
	args := m.Called(userID)
	return args.Get(0).([]model.TypePayment), args.Error(1)
}

func (m *Mock) FindByID(_ context.Context, id int, userID string) (model.TypePayment, error) {
	args := m.Called(id, userID)
	return args.Get(0).(model.TypePayment), args.Error(1)
}

func (m *Mock) Update(_ context.Context, id int, typePayment model.TypePayment, userID string) (model.TypePayment, error) {
	args := m.Called(id, typePayment, userID)
	return args.Get(0).(model.TypePayment), args.Error(1)
}

func (m *Mock) Delete(_ context.Context, _ int) error {
	args := m.Called()
	return args.Error(0)
}
