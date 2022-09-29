package repository

import (
	"context"

	"personal-finance/internal/model"

	"github.com/stretchr/testify/mock"
)

type Mock struct {
	mock.Mock
}

func (m *Mock) Add(_ context.Context, typePayment model.TypePayment) (model.TypePayment, error) {
	args := m.Called(typePayment)
	return args.Get(0).(model.TypePayment), args.Error(1)
}

func (m *Mock) FindAll(_ context.Context) ([]model.TypePayment, error) {
	args := m.Called()
	return args.Get(0).([]model.TypePayment), args.Error(1)
}

func (m *Mock) FindByID(_ context.Context, _ int) (model.TypePayment, error) {
	args := m.Called()
	return args.Get(0).(model.TypePayment), args.Error(1)
}

func (m *Mock) Update(_ context.Context, _ int, _ model.TypePayment) (model.TypePayment, error) {
	args := m.Called()
	return args.Get(0).(model.TypePayment), args.Error(1)
}

func (m *Mock) Delete(_ context.Context, _ int) error {
	args := m.Called()
	return args.Error(0)
}
