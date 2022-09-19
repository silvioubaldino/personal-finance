package repository

import (
	"context"

	"github.com/stretchr/testify/mock"

	"personal-finance/internal/model"
)

type Mock struct {
	mock.Mock
}

func (m *Mock) Add(_ context.Context, wallet model.Wallet) (model.Wallet, error) {
	args := m.Called(wallet)
	return args.Get(0).(model.Wallet), args.Error(1)
}

func (m *Mock) FindAll(_ context.Context) ([]model.Wallet, error) {
	args := m.Called()
	return args.Get(0).([]model.Wallet), args.Error(1)
}

func (m *Mock) FindByID(_ context.Context, _ int) (model.Wallet, error) {
	args := m.Called()
	return args.Get(0).(model.Wallet), args.Error(1)
}

func (m *Mock) Update(_ context.Context, _ int, _ model.Wallet) (model.Wallet, error) {
	args := m.Called()
	return args.Get(0).(model.Wallet), args.Error(1)
}

func (m *Mock) Delete(_ context.Context, _ int) error {
	args := m.Called()
	return args.Error(0)
}
