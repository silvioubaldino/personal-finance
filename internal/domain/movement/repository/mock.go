package repository

import (
	"context"

	"github.com/google/uuid"

	"personal-finance/internal/model"

	"github.com/stretchr/testify/mock"
)

type Mock struct {
	mock.Mock
}

func (m *Mock) Add(_ context.Context, transaction model.Movement) (model.Movement, error) {
	args := m.Called(transaction)
	return args.Get(0).(model.Movement), args.Error(1)
}

func (m *Mock) FindAll(_ context.Context) ([]model.Movement, error) {
	args := m.Called()
	return args.Get(0).([]model.Movement), args.Error(1)
}

func (m *Mock) FindByID(_ context.Context, id uuid.UUID) (model.Movement, error) {
	args := m.Called(id)
	return args.Get(0).(model.Movement), args.Error(1)
}

func (m *Mock) FindByPeriod(_ context.Context, period model.Period) ([]model.Movement, error) {
	args := m.Called(period)
	return args.Get(0).([]model.Movement), args.Error(1)
}

func (m *Mock) BalanceByPeriod(_ context.Context, _ model.Period) (model.Period, error) {
	args := m.Called()
	return args.Get(0).(model.Period), args.Error(1)
}

func (m *Mock) FindByIDByTransactionStatusID(_ context.Context, _ int, _ int) (model.Movement, error) {
	args := m.Called()
	return args.Get(0).(model.Movement), args.Error(1)
}

func (m *Mock) FindByTransactionID(_ context.Context, parentID uuid.UUID, transactionStatusID int) (model.MovementList, error) {
	args := m.Called(parentID, transactionStatusID)
	return args.Get(0).(model.MovementList), args.Error(1)
}

func (m *Mock) FindByStatusByPeriod(_ context.Context, transactionStatusID int, period model.Period) ([]model.Movement, error) {
	args := m.Called(transactionStatusID, period)
	return args.Get(0).([]model.Movement), args.Error(1)
}

func (m *Mock) FindSingleTransactionByPeriod(_ context.Context, transactionStatusID int, period model.Period) ([]model.Movement, error) {
	args := m.Called(transactionStatusID, period)
	return args.Get(0).([]model.Movement), args.Error(1)
}

func (m *Mock) Update(_ context.Context, id uuid.UUID, transaction model.Movement) (model.Movement, error) {
	args := m.Called(id, transaction)
	return args.Get(0).(model.Movement), args.Error(1)
}

func (m *Mock) Delete(_ context.Context, id uuid.UUID) error {
	args := m.Called(id)
	return args.Error(0)
}
