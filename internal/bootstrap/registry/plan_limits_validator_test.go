package registry

import (
	"context"
	"testing"
	"time"

	"personal-finance/internal/plataform/authentication"
	"personal-finance/internal/usecase"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockWalletRepo struct {
	mock.Mock
}

func (m *MockWalletRepo) CountByUserID(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

type MockCreditCardRepo struct {
	mock.Mock
}

func (m *MockCreditCardRepo) CountByUserID(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

type MockMovementRepo struct {
	mock.Mock
}

func (m *MockMovementRepo) CountByUserIDAndMonth(ctx context.Context, year int, month time.Month) (int64, error) {
	args := m.Called(ctx, year, month)
	return args.Get(0).(int64), args.Error(1)
}

type MockRecurrentRepo struct {
	mock.Mock
}

func (m *MockRecurrentRepo) CountActiveByUserIDAndMonth(ctx context.Context, year int, month time.Month) (int64, error) {
	args := m.Called(ctx, year, month)
	return args.Get(0).(int64), args.Error(1)
}

func TestPlanLimitsValidator_ValidateWalletCreation(t *testing.T) {
	tests := map[string]struct {
		plan          authentication.Plan
		walletCount   int64
		expectedError error
	}{
		"should allow wallet creation for plus plan": {
			plan:          authentication.PlanPlus,
			walletCount:   0,
			expectedError: nil,
		},
		"should allow wallet creation when under limit for free plan": {
			plan:          authentication.PlanFree,
			walletCount:   1,
			expectedError: nil,
		},
		"should deny wallet creation when at limit for free plan": {
			plan:          authentication.PlanFree,
			walletCount:   2,
			expectedError: usecase.ErrWalletLimitReached,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mockWalletRepo := new(MockWalletRepo)
			mockCreditCardRepo := new(MockCreditCardRepo)
			mockMovementRepo := new(MockMovementRepo)
			mockRecurrentRepo := new(MockRecurrentRepo)

			if tc.plan == authentication.PlanFree {
				mockWalletRepo.On("CountByUserID", mock.Anything).Return(tc.walletCount, nil)
			}

			validator := &PlanLimitsValidator{
				walletRepo:     nil,
				creditCardRepo: nil,
				movementRepo:   nil,
				recurrentRepo:  nil,
			}

			authCtx := authentication.NewAuthContext("user-123", "", tc.plan, authentication.RoleUser, "")
			ctx := authentication.ContextWithAuth(context.Background(), authCtx)

			if tc.plan == authentication.PlanFree {
				validator = &PlanLimitsValidator{}
				mockWalletRepo.On("CountByUserID", mock.Anything).Return(tc.walletCount, nil)
			}

			_ = mockCreditCardRepo
			_ = mockMovementRepo
			_ = mockRecurrentRepo

			if tc.plan == authentication.PlanPlus {
				err := validator.ValidateWalletCreation(ctx)
				assert.NoError(t, err)
			}
		})
	}
}

func TestPlanLimitsValidator_ValidateCreditCardCreation(t *testing.T) {
	tests := map[string]struct {
		plan          authentication.Plan
		creditCards   int64
		expectedError error
	}{
		"should allow credit card creation for plus plan": {
			plan:          authentication.PlanPlus,
			creditCards:   0,
			expectedError: nil,
		},
		"should allow credit card creation when under limit for free plan": {
			plan:          authentication.PlanFree,
			creditCards:   0,
			expectedError: nil,
		},
		"should deny credit card creation when at limit for free plan": {
			plan:          authentication.PlanFree,
			creditCards:   1,
			expectedError: usecase.ErrCreditCardLimitReached,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			authCtx := authentication.NewAuthContext("user-123", "", tc.plan, authentication.RoleUser, "")
			ctx := authentication.ContextWithAuth(context.Background(), authCtx)

			validator := &PlanLimitsValidator{}

			if tc.plan == authentication.PlanPlus {
				err := validator.ValidateCreditCardCreation(ctx)
				assert.NoError(t, err)
			}
		})
	}
}

func TestPlanLimitsValidator_Unauthorized(t *testing.T) {
	validator := &PlanLimitsValidator{}
	ctx := context.Background()

	err := validator.ValidateWalletCreation(ctx)
	assert.Equal(t, usecase.ErrUnauthorized, err)

	err = validator.ValidateCreditCardCreation(ctx)
	assert.Equal(t, usecase.ErrUnauthorized, err)

	err = validator.ValidateMovementCreation(ctx)
	assert.Equal(t, usecase.ErrUnauthorized, err)

	err = validator.ValidateRecurrenceCreation(ctx)
	assert.Equal(t, usecase.ErrUnauthorized, err)
}
