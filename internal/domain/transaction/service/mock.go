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

func (s *Mock) AddDoneTransaction(_ context.Context, doneMovement model.Movement) (model.Transaction, error) {
	args := s.Called(doneMovement)
	return args.Get(0).(model.Transaction), args.Error(1)
}

func (s *Mock) FindByID(_ context.Context, id uuid.UUID, userID string) (model.Transaction, error) {
	args := s.Called(id, userID)
	return args.Get(0).(model.Transaction), args.Error(1)
}

func (s *Mock) FindByPeriod(_ context.Context, period model.Period, userID string) ([]model.Transaction, error) {
	args := s.Called(period, userID)
	return args.Get(0).([]model.Transaction), args.Error(1)
}
