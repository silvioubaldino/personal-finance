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

func (m *Mock) Add(_ context.Context, movement model.Movement, userID string) (model.Movement, error) {
	args := m.Called(movement, userID)
	return args.Get(0).(model.Movement), args.Error(1)
}

func (m *Mock) AddConsistent(_ context.Context, tx *gorm.DB, movement model.Movement, userID string) (model.Movement, error) {
	args := m.Called(tx, movement, userID)
	return args.Get(0).(model.Movement), args.Error(1)
}

func (m *Mock) AddUpdatingWallet(_ context.Context, tx *gorm.DB, movement model.Movement, userID string) (model.Movement, error) {
	args := m.Called(tx, movement, userID)
	return args.Get(0).(model.Movement), args.Error(1)
}

func (m *Mock) FindByID(_ context.Context, id uuid.UUID, userID string) (model.Movement, error) {
	args := m.Called(id, userID)
	return args.Get(0).(model.Movement), args.Error(1)
}

func (m *Mock) FindByPeriod(_ context.Context, period model.Period, userID string) ([]model.Movement, error) {
	args := m.Called(period, userID)
	return args.Get(0).([]model.Movement), args.Error(1)
}

func (m *Mock) BalanceByPeriod(_ context.Context, period model.Period, userID string) (model.Period, error) {
	args := m.Called(period, userID)
	return args.Get(0).(model.Period), args.Error(1)
}

func (m *Mock) FindByTransactionID(_ context.Context, parentID uuid.UUID, transactionStatusID int, userID string) (model.MovementList, error) {
	args := m.Called(parentID, transactionStatusID, userID)
	return args.Get(0).(model.MovementList), args.Error(1)
}

func (m *Mock) FindByStatusByPeriod(_ context.Context, transactionStatusID int, period model.Period, userID string) ([]model.Movement, error) {
	args := m.Called(transactionStatusID, period, userID)
	return args.Get(0).([]model.Movement), args.Error(1)
}

func (m *Mock) Update(_ context.Context, id uuid.UUID, transaction model.Movement, userID string) (model.Movement, error) {
	args := m.Called(id, transaction, userID)
	return args.Get(0).(model.Movement), args.Error(1)
}

func (m *Mock) Delete(_ context.Context, id uuid.UUID, userID string) error {
	args := m.Called(id, userID)
	return args.Error(0)
}
