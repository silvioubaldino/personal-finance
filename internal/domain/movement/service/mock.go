package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/stretchr/testify/mock"

	"personal-finance/internal/model"
)

type Mock struct {
	mock.Mock
}

func (s *Mock) Add(_ context.Context, transaction model.Movement) (model.Movement, error) {
	args := s.Called(transaction)
	return args.Get(0).(model.Movement), args.Error(1)
}

func (s *Mock) FindAll(_ context.Context) ([]model.Movement, error) {
	args := s.Called()
	return args.Get(0).([]model.Movement), args.Error(1)
}

func (s *Mock) FindByID(_ context.Context, id uuid.UUID) (model.Movement, error) {
	args := s.Called(id)
	return args.Get(0).(model.Movement), args.Error(1)
}

func (s *Mock) FindByPeriod(_ context.Context, _ model.Period) ([]model.Movement, error) {
	args := s.Called()
	return args.Get(0).([]model.Movement), args.Error(1)
}

func (s *Mock) FindParentTransactionByID(_ context.Context, id int) (model.Transaction, error) {
	args := s.Called(id)
	return args.Get(0).(model.Transaction), args.Error(1)
}

func (s *Mock) BalanceByPeriod(_ context.Context, _ model.Period) (model.Balance, error) {
	args := s.Called()
	return args.Get(0).(model.Balance), args.Error(1)
}

func (s *Mock) FindConsolidatedTransactionByID(_ context.Context, id int) (model.Transaction, error) {
	args := s.Called(id)
	return args.Get(0).(model.Transaction), args.Error(1)
}

func (s *Mock) FindConsolidatedTransactionByPeriod(_ context.Context, _ model.Period) ([]model.Transaction, error) {
	args := s.Called()
	return args.Get(0).([]model.Transaction), args.Error(1)
}

func (s *Mock) Update(_ context.Context, id uuid.UUID, transaction model.Movement) (model.Movement, error) {
	args := s.Called(id, transaction)
	return args.Get(0).(model.Movement), args.Error(1)
}

func (s *Mock) Delete(_ context.Context, id uuid.UUID) error {
	args := s.Called(id)
	return args.Error(0)
}
