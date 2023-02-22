package repository

import (
	"context"

	"github.com/stretchr/testify/mock"

	"personal-finance/internal/model"
)

type Mock struct {
	mock.Mock
}

func (m *Mock) Add(_ context.Context, wallet model.Wallet, userID string) (model.Wallet, error) {
	args := m.Called(wallet, userID)
	return args.Get(0).(model.Wallet), args.Error(1)
}

func (m *Mock) FindAll(_ context.Context, userID string) ([]model.Wallet, error) {
	args := m.Called(userID)
	return args.Get(0).([]model.Wallet), args.Error(1)
}

func (m *Mock) FindByID(_ context.Context, id int, userID string) (model.Wallet, error) {
	args := m.Called(id, userID)
	return args.Get(0).(model.Wallet), args.Error(1)
}

func (m *Mock) Update(_ context.Context, id int, wallet model.Wallet, userID string) (model.Wallet, error) {
	args := m.Called(id, wallet, userID)
	return args.Get(0).(model.Wallet), args.Error(1)
}

func (m *Mock) Delete(_ context.Context, _ int) error {
	args := m.Called()
	return args.Error(0)
}
