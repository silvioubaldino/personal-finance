package api

import (
	"context"
	"time"

	"personal-finance/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type MockMovementUseCase struct {
	mock.Mock
}

func (m *MockMovementUseCase) Add(ctx context.Context, movement domain.Movement) (domain.Movement, error) {
	args := m.Called(ctx, movement)
	return args.Get(0).(domain.Movement), args.Error(1)
}

func (m *MockMovementUseCase) FindByPeriod(ctx context.Context, period domain.Period) (domain.MovementList, error) {
	args := m.Called(ctx, period)
	return args.Get(0).([]domain.Movement), args.Error(1)
}

func (m *MockMovementUseCase) Pay(ctx context.Context, id uuid.UUID, date time.Time) (domain.Movement, error) {
	args := m.Called(ctx, id, date)
	return args.Get(0).(domain.Movement), args.Error(1)
}

func (m *MockMovementUseCase) RevertPay(ctx context.Context, id uuid.UUID) (domain.Movement, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(domain.Movement), args.Error(1)
}
