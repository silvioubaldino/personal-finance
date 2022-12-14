package repository

import (
	"context"

	"github.com/google/uuid"

	"personal-finance/internal/model"
	"personal-finance/internal/model/eager"

	"github.com/stretchr/testify/mock"
)

type Mock struct {
	mock.Mock
}

func (m *Mock) Add(_ context.Context, transaction model.Transaction) (model.Transaction, error) {
	args := m.Called(transaction)
	return args.Get(0).(model.Transaction), args.Error(1)
}

func (m *Mock) FindAll(_ context.Context) ([]model.Transaction, error) {
	args := m.Called()
	return args.Get(0).([]model.Transaction), args.Error(1)
}

func (m *Mock) FindByID(_ context.Context, _ int) (model.Transaction, error) {
	args := m.Called()
	return args.Get(0).(model.Transaction), args.Error(1)
}

func (m *Mock) FindByIDEager(_ context.Context, _ int) (eager.Transaction, error) {
	args := m.Called()
	return args.Get(0).(eager.Transaction), args.Error(1)
}

func (m *Mock) FindByMonth(_ context.Context, _ model.Period) ([]model.Transaction, error) {
	args := m.Called()
	return args.Get(0).([]model.Transaction), args.Error(1)
}

func (m *Mock) BalanceByPeriod(_ context.Context, _ model.Period) (model.Period, error) {
	args := m.Called()
	return args.Get(0).(model.Period), args.Error(1)
}

func (m *Mock) FindByIDByTransactionStatusID(_ context.Context, _ int, _ int) (model.Transaction, error) {
	args := m.Called()
	return args.Get(0).(model.Transaction), args.Error(1)
}

func (m *Mock) FindByParentTransactionID(_ context.Context, _ uuid.UUID, _ int) ([]model.Transaction, error) {
	args := m.Called()
	return args.Get(0).([]model.Transaction), args.Error(1)
}

func (m *Mock) FindByTransactionStatusIDByPeriod(_ context.Context, _ int, _ model.Period) ([]model.Transaction, error) {
	args := m.Called()
	return args.Get(0).([]model.Transaction), args.Error(1)
}

func (m *Mock) Update(_ context.Context, _ int, _ model.Transaction) (model.Transaction, error) {
	args := m.Called()
	return args.Get(0).(model.Transaction), args.Error(1)
}

func (m *Mock) Delete(_ context.Context, _ int) error {
	args := m.Called()
	return args.Error(0)
}
