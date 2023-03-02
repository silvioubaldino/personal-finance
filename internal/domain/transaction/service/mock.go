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

func (s *Mock) FindByID(_ context.Context, id uuid.UUID) (model.Transaction, error) {
	args := s.Called(id)
	return args.Get(0).(model.Transaction), args.Error(1)
}

func (s *Mock) FindByPeriod(_ context.Context, period model.Period) ([]model.Transaction, error) {
	args := s.Called(period)
	return args.Get(0).([]model.Transaction), args.Error(1)
}
