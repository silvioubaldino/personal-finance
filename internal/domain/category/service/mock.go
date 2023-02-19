package service

import (
	"context"

	"github.com/stretchr/testify/mock"

	"personal-finance/internal/model"
)

type Mock struct {
	mock.Mock
}

func (s *Mock) Add(_ context.Context, category model.Category, userID string) (model.Category, error) {
	args := s.Called(category, userID)
	return args.Get(0).(model.Category), args.Error(1)
}

func (s *Mock) FindAll(_ context.Context, userID string) ([]model.Category, error) {
	args := s.Called(userID)
	return args.Get(0).([]model.Category), args.Error(1)
}

func (s *Mock) FindByID(_ context.Context, id int, userID string) (model.Category, error) {
	args := s.Called(id, userID)
	return args.Get(0).(model.Category), args.Error(1)
}

func (s *Mock) Update(_ context.Context, _ int, category model.Category, userID string) (model.Category, error) {
	args := s.Called(category, userID)
	return args.Get(0).(model.Category), args.Error(1)
}

func (s *Mock) Delete(_ context.Context, id int) error {
	args := s.Called(id)
	return args.Error(0)
}
