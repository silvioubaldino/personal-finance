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

type MockMPGateway struct {
	mock.Mock
}

func (m *MockMPGateway) CreateSubscription(ctx context.Context, req gateway.MPCreateSubscriptionRequest) (gateway.MPCreateSubscriptionResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(gateway.MPCreateSubscriptionResponse), args.Error(1)
}

func (m *MockMPGateway) GetSubscription(ctx context.Context, id string) (gateway.MPSubscription, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(gateway.MPSubscription), args.Error(1)
}

type MockFirebaseSubGateway struct {
	mock.Mock
}

func (m *MockFirebaseSubGateway) SetUserPlan(ctx context.Context, userID string, plan authentication.Plan) error {
	args := m.Called(ctx, userID, plan)
	return args.Error(0)
}

func TestSubscription_CreateCheckout(t *testing.T) {
	tests := map[string]struct {
		authCtx       *authentication.AuthContext
		mockSetup     func(*MockMPGateway)
		expectedURL   string
		expectedError error
	}{
		"should create checkout successfully": {
			authCtx: &authentication.AuthContext{UserID: "user-123"},
			mockSetup: func(m *MockMPGateway) {
				m.On("CreateSubscription", mock.Anything, mock.MatchedBy(func(req gateway.MPCreateSubscriptionRequest) bool {
					return req.ExternalReference == "user-123"
				})).Return(gateway.MPCreateSubscriptionResponse{InitPoint: "http://mp.com/pay"}, nil)
			},
			expectedURL:   "http://mp.com/pay",
			expectedError: nil,
		},
		"should return unauthorized if no user in context": {
			authCtx:       nil,
			mockSetup:     func(m *MockMPGateway) {},
			expectedURL:   "",
			expectedError: ErrUnauthorized,
		},
		"should return gateway error if MP fails": {
			authCtx: &authentication.AuthContext{UserID: "user-123"},
			mockSetup: func(m *MockMPGateway) {
				m.On("CreateSubscription", mock.Anything, mock.Anything).Return(gateway.MPCreateSubscriptionResponse{}, errors.New("mp error"))
			},
			expectedURL:   "",
			expectedError: ErrMercadoPagoGateway,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mockMP := new(MockMPGateway)
			mockFS := new(MockFirebaseSubGateway)
			tc.mockSetup(mockMP)

			s := NewSubscription(mockMP, mockFS)

			ctx := context.Background()
			if tc.authCtx != nil {
				ctx = authentication.ContextWithAuth(ctx, *tc.authCtx)
			}

			resp, err := s.CreateCheckout(ctx)

			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tc.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedURL, resp.URL)
			}
			mockMP.AssertExpectations(t)
		})
	}
}

func TestSubscription_HandleWebhook(t *testing.T) {
	// Note: We skip signature validation in these tests by not setting MERCADOPAGO_WEBHOOK_SECRET
	// or passing empty signature.

	tests := map[string]struct {
		body          string
		mockMPSetup   func(*MockMPGateway)
		mockFSSetup   func(*MockFirebaseSubGateway)
		expectedError error
	}{
		"should update to plus on authorized": {
			body: `{"action":"created","type":"subscription_preapproval","data":{"id":"sub-123"}}`,
			mockMPSetup: func(m *MockMPGateway) {
				m.On("GetSubscription", mock.Anything, "sub-123").Return(gateway.MPSubscription{
					ID:                "sub-123",
					Status:            "authorized",
					ExternalReference: "user-123",
				}, nil)
			},
			mockFSSetup: func(m *MockFirebaseSubGateway) {
				m.On("SetUserPlan", mock.Anything, "user-123", authentication.PlanPlus).Return(nil)
			},
			expectedError: nil,
		},
		"should update to free on cancelled": {
			body: `{"action":"created","type":"subscription_preapproval","data":{"id":"sub-123"}}`,
			mockMPSetup: func(m *MockMPGateway) {
				m.On("GetSubscription", mock.Anything, "sub-123").Return(gateway.MPSubscription{
					ID:                "sub-123",
					Status:            "cancelled",
					ExternalReference: "user-123",
				}, nil)
			},
			mockFSSetup: func(m *MockFirebaseSubGateway) {
				m.On("SetUserPlan", mock.Anything, "user-123", authentication.PlanFree).Return(nil)
			},
			expectedError: nil,
		},
		"should ignore other types": {
			body:          `{"action":"created","type":"payment","data":{"id":"pay-123"}}`,
			mockMPSetup:   func(m *MockMPGateway) {},
			mockFSSetup:   func(m *MockFirebaseSubGateway) {},
			expectedError: nil,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mockMP := new(MockMPGateway)
			mockFS := new(MockFirebaseSubGateway)
			tc.mockMPSetup(mockMP)
			tc.mockFSSetup(mockFS)

			s := NewSubscription(mockMP, mockFS)

			err := s.HandleWebhook(context.Background(), []byte(tc.body), "")

			assert.Equal(t, tc.expectedError, err)
			mockMP.AssertExpectations(t)
			mockFS.AssertExpectations(t)
		})
	}
}
