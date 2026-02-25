package usecase

import (
	"context"
	"errors"
	"testing"

	"personal-finance/internal/infrastructure/gateway"
	"personal-finance/internal/plataform/authentication"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockFirebaseGateway struct {
	mock.Mock
}

func (m *MockFirebaseGateway) GetUserClaims(ctx context.Context, userID string) (gateway.UserClaims, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(gateway.UserClaims), args.Error(1)
}

func (m *MockFirebaseGateway) SetUserPlan(ctx context.Context, userID string, plan authentication.Plan) error {
	args := m.Called(ctx, userID, plan)
	return args.Error(0)
}

func (m *MockFirebaseGateway) SetUserRole(ctx context.Context, userID string, role authentication.Role) error {
	args := m.Called(ctx, userID, role)
	return args.Error(0)
}

func TestAdmin_GetUserClaims(t *testing.T) {
	tests := map[string]struct {
		role           authentication.Role
		targetUserID   string
		mockSetup      func(*MockFirebaseGateway)
		expectedResult UserClaimsResponse
		expectedError  error
	}{
		"should get user claims successfully": {
			role:         authentication.RoleAdmin,
			targetUserID: "target-user-123",
			mockSetup: func(m *MockFirebaseGateway) {
				m.On("GetUserClaims", mock.Anything, "target-user-123").Return(gateway.UserClaims{
					Plan: authentication.PlanPlus,
					Role: authentication.RoleUser,
				}, nil)
			},
			expectedResult: UserClaimsResponse{
				UserID: "target-user-123",
				Plan:   "plus",
				Role:   "user",
			},
			expectedError: nil,
		},
		"should return forbidden when not admin": {
			role:          authentication.RoleUser,
			targetUserID:  "target-user-123",
			mockSetup:     func(m *MockFirebaseGateway) {},
			expectedError: ErrForbidden,
		},
		"should return error when gateway fails": {
			role:         authentication.RoleAdmin,
			targetUserID: "target-user-123",
			mockSetup: func(m *MockFirebaseGateway) {
				m.On("GetUserClaims", mock.Anything, "target-user-123").Return(gateway.UserClaims{}, errors.New("gateway error"))
			},
			expectedError: errors.New("gateway error"),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mockGateway := new(MockFirebaseGateway)
			tc.mockSetup(mockGateway)

			admin := NewAdmin(mockGateway)

			authCtx := authentication.NewAuthContext("admin-123", "", authentication.PlanPlus, tc.role, "")
			ctx := authentication.ContextWithAuth(context.Background(), authCtx)

			result, err := admin.GetUserClaims(ctx, tc.targetUserID)

			if tc.expectedError != nil {
				assert.Error(t, err)
				if tc.expectedError == ErrForbidden {
					assert.Equal(t, ErrForbidden, err)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedResult, result)
			}

			mockGateway.AssertExpectations(t)
		})
	}
}

func TestAdmin_SetUserPlan(t *testing.T) {
	tests := map[string]struct {
		role          authentication.Role
		targetUserID  string
		plan          string
		mockSetup     func(*MockFirebaseGateway)
		expectedError error
	}{
		"should set user plan successfully": {
			role:         authentication.RoleAdmin,
			targetUserID: "target-user-123",
			plan:         "plus",
			mockSetup: func(m *MockFirebaseGateway) {
				m.On("SetUserPlan", mock.Anything, "target-user-123", authentication.PlanPlus).Return(nil)
			},
			expectedError: nil,
		},
		"should return forbidden when not admin": {
			role:          authentication.RoleUser,
			targetUserID:  "target-user-123",
			plan:          "plus",
			mockSetup:     func(m *MockFirebaseGateway) {},
			expectedError: ErrForbidden,
		},
		"should return error for invalid plan": {
			role:          authentication.RoleAdmin,
			targetUserID:  "target-user-123",
			plan:          "invalid",
			mockSetup:     func(m *MockFirebaseGateway) {},
			expectedError: ErrInvalidPlan,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mockGateway := new(MockFirebaseGateway)
			tc.mockSetup(mockGateway)

			admin := NewAdmin(mockGateway)

			authCtx := authentication.NewAuthContext("admin-123", "", authentication.PlanPlus, tc.role, "")
			ctx := authentication.ContextWithAuth(context.Background(), authCtx)

			err := admin.SetUserPlan(ctx, tc.targetUserID, tc.plan)

			assert.Equal(t, tc.expectedError, err)
			mockGateway.AssertExpectations(t)
		})
	}
}

func TestAdmin_SetUserRole(t *testing.T) {
	tests := map[string]struct {
		role          authentication.Role
		targetUserID  string
		newRole       string
		mockSetup     func(*MockFirebaseGateway)
		expectedError error
	}{
		"should set user role successfully": {
			role:         authentication.RoleAdmin,
			targetUserID: "target-user-123",
			newRole:      "admin",
			mockSetup: func(m *MockFirebaseGateway) {
				m.On("SetUserRole", mock.Anything, "target-user-123", authentication.RoleAdmin).Return(nil)
			},
			expectedError: nil,
		},
		"should return forbidden when not admin": {
			role:          authentication.RoleUser,
			targetUserID:  "target-user-123",
			newRole:       "admin",
			mockSetup:     func(m *MockFirebaseGateway) {},
			expectedError: ErrForbidden,
		},
		"should return error for invalid role": {
			role:          authentication.RoleAdmin,
			targetUserID:  "target-user-123",
			newRole:       "invalid",
			mockSetup:     func(m *MockFirebaseGateway) {},
			expectedError: ErrInvalidRole,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mockGateway := new(MockFirebaseGateway)
			tc.mockSetup(mockGateway)

			admin := NewAdmin(mockGateway)

			authCtx := authentication.NewAuthContext("admin-123", "", authentication.PlanPlus, tc.role, "")
			ctx := authentication.ContextWithAuth(context.Background(), authCtx)

			err := admin.SetUserRole(ctx, tc.targetUserID, tc.newRole)

			assert.Equal(t, tc.expectedError, err)
			mockGateway.AssertExpectations(t)
		})
	}
}

func TestAdmin_Unauthorized(t *testing.T) {
	mockGateway := new(MockFirebaseGateway)
	admin := NewAdmin(mockGateway)

	ctx := context.Background()

	_, err := admin.GetUserClaims(ctx, "user-123")
	assert.Equal(t, ErrUnauthorized, err)

	err = admin.SetUserPlan(ctx, "user-123", "plus")
	assert.Equal(t, ErrUnauthorized, err)

	err = admin.SetUserRole(ctx, "user-123", "admin")
	assert.Equal(t, ErrUnauthorized, err)
}
