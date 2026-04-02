package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"

	"personal-finance/internal/model"
)

type Mock struct {
	mock.Mock
}

func (m *Mock) RecalculateBalance(_ context.Context, id *uuid.UUID) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *Mock) Add(_ context.Context, wallet model.Wallet) (model.Wallet, error) {
	args := m.Called(wallet)
	return args.Get(0).(model.Wallet), args.Error(1)
}

func (m *Mock) UpdateConsistent(_ context.Context, tx *gorm.DB, wallet model.Wallet) (model.Wallet, error) {
	args := m.Called(tx, wallet)
	return args.Get(0).(model.Wallet), args.Error(1)
}

func (m *Mock) FindAll(_ context.Context) ([]model.Wallet, error) {
	args := m.Called()
	return args.Get(0).([]model.Wallet), args.Error(1)
}

func (m *Mock) FindByID(_ context.Context, id *uuid.UUID) (model.Wallet, error) {
	args := m.Called(id)
	return args.Get(0).(model.Wallet), args.Error(1)
}

func (m *Mock) Update(_ context.Context, id *uuid.UUID, wallet model.Wallet) (model.Wallet, error) {
	args := m.Called(id, wallet)
	return args.Get(0).(model.Wallet), args.Error(1)
}

func (m *Mock) Delete(_ context.Context, id *uuid.UUID) error {
	args := m.Called(id)
	return args.Error(0)
}
