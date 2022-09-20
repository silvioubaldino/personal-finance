package service

import (
	"context"

	"github.com/stretchr/testify/mock"

	"personal-finance/internal/model"
)

type Mock struct {
	mock.Mock
}

func (s *Mock) Add(_ context.Context, typePayment model.TypePayment) (model.TypePayment, error) {
	args := s.Called(typePayment)
	return args.Get(0).(model.TypePayment), args.Error(1)
}

func (s *Mock) FindAll(_ context.Context) ([]model.TypePayment, error) {
	args := s.Called()
	return args.Get(0).([]model.TypePayment), args.Error(1)
}

func (s *Mock) FindByID(_ context.Context, id int) (model.TypePayment, error) {
	args := s.Called(id)
	return args.Get(0).(model.TypePayment), args.Error(1)
}

func (s *Mock) Update(_ context.Context, _ int, typePayment model.TypePayment) (model.TypePayment, error) {
	args := s.Called(typePayment)
	return args.Get(0).(model.TypePayment), args.Error(1)
}

func (s *Mock) Delete(_ context.Context, id int) error {
	args := s.Called(id)
	return args.Error(0)
}
