package api

import (
	"context"

	"personal-finance/internal/domain"

	"github.com/stretchr/testify/mock"
)

type MockMovementUseCase struct {
	mock.Mock
}

func (m *MockMovementUseCase) Add(ctx context.Context, movement domain.Movement) (domain.Movement, error) {
	args := m.Called(ctx, movement)
	return args.Get(0).(domain.Movement), args.Error(1)
}

func (m *MockMovementUseCase) FindByPeriod(ctx context.Context, period domain.Period) ([]domain.Movement, error) {
	args := m.Called(ctx, period)
	return args.Get(0).([]domain.Movement), args.Error(1)
}
