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

func (s *Mock) Add(_ context.Context, transaction model.Movement, userID string) (model.Movement, error) {
	args := s.Called(transaction, userID)
	return args.Get(0).(model.Movement), args.Error(1)
}

func (s *Mock) FindByID(_ context.Context, id uuid.UUID, userID string) (model.Movement, error) {
	args := s.Called(id, userID)
	return args.Get(0).(model.Movement), args.Error(1)
}

func (s *Mock) FindByPeriod(_ context.Context, period model.Period, userID string) ([]model.Movement, error) {
	args := s.Called(period, userID)
	return args.Get(0).([]model.Movement), args.Error(1)
}

func (s *Mock) Update(_ context.Context, id uuid.UUID, transaction model.Movement, userID string) (model.Movement, error) {
	args := s.Called(id, transaction, userID)
	return args.Get(0).(model.Movement), args.Error(1)
}

func (s *Mock) Delete(_ context.Context, id uuid.UUID, userID string) error {
	args := s.Called(id, userID)
	return args.Error(0)
}
