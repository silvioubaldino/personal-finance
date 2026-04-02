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

func (s *Mock) RecalculateBalance(_ context.Context, id *uuid.UUID) error {
	args := s.Called(id)
	return args.Error(0)
}

func (s *Mock) Add(_ context.Context, wallet model.Wallet) (model.Wallet, error) {
	args := s.Called(wallet)
	return args.Get(0).(model.Wallet), args.Error(1)
}

func (s *Mock) FindAll(_ context.Context) ([]model.Wallet, error) {
	args := s.Called()
	return args.Get(0).([]model.Wallet), args.Error(1)
}

func (s *Mock) FindByID(_ context.Context, id *uuid.UUID) (model.Wallet, error) {
	args := s.Called(id)
	return args.Get(0).(model.Wallet), args.Error(1)
}

func (s *Mock) Update(_ context.Context, id *uuid.UUID, wallet model.Wallet) (model.Wallet, error) {
	args := s.Called(id, wallet)
	return args.Get(0).(model.Wallet), args.Error(1)
}

func (s *Mock) Delete(_ context.Context, id *uuid.UUID) error {
	args := s.Called(id)
	return args.Error(0)
}
