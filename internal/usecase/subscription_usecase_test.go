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

func (m *MockMPGateway) CreateSubscriptionURL(ctx context.Context, payerEmail, externalID, backURL string, price float64) (string, error) {
	args := m.Called(ctx, payerEmail, externalID, backURL, price)
	return args.String(0), args.Error(1)
}

type MockAppSettingsReader struct {
	mock.Mock
}

func (m *MockAppSettingsReader) GetFloat(ctx context.Context, key string) (float64, error) {
	args := m.Called(ctx, key)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockAppSettingsReader) SetFloat(ctx context.Context, key string, value float64) error {
	args := m.Called(ctx, key, value)
	return args.Error(0)
}

func (m *MockMPGateway) GetSubscription(ctx context.Context, id string) (gateway.MPSubscription, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(gateway.MPSubscription), args.Error(1)
}

func (m *MockMPGateway) CancelSubscription(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

type MockFirebaseSubGateway struct {
	mock.Mock
}

func (m *MockFirebaseSubGateway) SetUserSubscription(ctx context.Context, userID string, plan authentication.Plan, mpSubscriptionID string, subscriptionSource authentication.SubscriptionSource, expiresAt int64) error {
	args := m.Called(ctx, userID, plan, mpSubscriptionID, subscriptionSource, expiresAt)
	return args.Error(0)
}

func TestSubscription_CreateCheckout(t *testing.T) {
	tests := map[string]struct {
		authCtx       *authentication.AuthContext
		backURL       string
		mockSetup     func(*MockMPGateway, *MockAppSettingsReader)
		expectedURL   string
		expectedError error
	}{
		"should create checkout with back_url": {
			authCtx: &authentication.AuthContext{UserID: "user-123"},
			backURL: "https://api.domain.com/subscription/return",
			mockSetup: func(m *MockMPGateway, s *MockAppSettingsReader) {
				s.On("GetFloat", mock.Anything, "plus_price").Return(9.90, nil)
				m.On("CreateSubscriptionURL", mock.Anything, mock.Anything, "user-123", "https://api.domain.com/subscription/return", 9.90).
					Return("http://mp.com/pay", nil)
			},
			expectedURL:   "http://mp.com/pay",
			expectedError: nil,
		},
		"should create checkout with empty back_url (falls back to env)": {
			authCtx: &authentication.AuthContext{UserID: "user-123"},
			backURL: "",
			mockSetup: func(m *MockMPGateway, s *MockAppSettingsReader) {
				s.On("GetFloat", mock.Anything, "plus_price").Return(9.90, nil)
				m.On("CreateSubscriptionURL", mock.Anything, mock.Anything, "user-123", "", 9.90).
					Return("http://mp.com/pay", nil)
			},
			expectedURL:   "http://mp.com/pay",
			expectedError: nil,
		},
		"should return unauthorized if no user in context": {
			authCtx: nil,
			backURL: "",
			mockSetup: func(m *MockMPGateway, s *MockAppSettingsReader) {
			},
			expectedURL:   "",
			expectedError: ErrUnauthorized,
		},
		"should return gateway error if MP fails": {
			authCtx: &authentication.AuthContext{UserID: "user-123"},
			backURL: "",
			mockSetup: func(m *MockMPGateway, s *MockAppSettingsReader) {
				s.On("GetFloat", mock.Anything, "plus_price").Return(9.90, nil)
				m.On("CreateSubscriptionURL", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("", errors.New("mp error"))
			},
			expectedURL:   "",
			expectedError: ErrMercadoPagoGateway,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mockMP := new(MockMPGateway)
			mockFS := new(MockFirebaseSubGateway)
			mockSettings := new(MockAppSettingsReader)
			tc.mockSetup(mockMP, mockSettings)

			s := NewSubscription(mockMP, mockFS, mockSettings)

			ctx := context.Background()
			if tc.authCtx != nil {
				ctx = authentication.ContextWithAuth(ctx, *tc.authCtx)
			}

			resp, err := s.CreateCheckout(ctx, tc.backURL)

			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tc.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedURL, resp)
			}
			mockMP.AssertExpectations(t)
			mockSettings.AssertExpectations(t)
		})
	}
}

func TestSubscription_HandleWebhook(t *testing.T) {
	// Note: We skip signature validation in these tests by not setting MERCADOPAGO_WEBHOOK_SECRET.

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
				m.On("SetUserSubscription", mock.Anything, "user-123", authentication.PlanPlus, "sub-123", authentication.SubscriptionSourceMP, int64(0)).Return(nil)
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
				m.On("SetUserSubscription", mock.Anything, "user-123", authentication.PlanFree, "", authentication.SubscriptionSourceMP, int64(0)).Return(nil)
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

			s := NewSubscription(mockMP, mockFS, new(MockAppSettingsReader))

			err := s.HandleWebhook(context.Background(), "", "", []byte(tc.body))

			assert.Equal(t, tc.expectedError, err)
			mockMP.AssertExpectations(t)
			mockFS.AssertExpectations(t)
		})
	}
}

func TestSubscription_HandleRevenueCatWebhook(t *testing.T) {
	tests := map[string]struct {
		body          string
		authHeader    string
		mockFSSetup   func(*MockFirebaseSubGateway)
		expectedError bool
	}{
		"should update to plus on INITIAL_PURCHASE": {
			body:       `{"api_version":"4.0","event":{"type":"INITIAL_PURCHASE","app_user_id":"user-123","entitlement_ids":["plus"],"expiration_at_ms":1700000000000,"product_id":"plus_monthly"}}`,
			authHeader: "",
			mockFSSetup: func(m *MockFirebaseSubGateway) {
				m.On("SetUserSubscription", mock.Anything, "user-123", authentication.PlanPlus, "", authentication.SubscriptionSourceIAP, int64(0)).Return(nil)
			},
			expectedError: false,
		},
		"should update to plus on RENEWAL": {
			body:       `{"api_version":"4.0","event":{"type":"RENEWAL","app_user_id":"user-123","entitlement_ids":["plus"],"expiration_at_ms":1700000000000,"product_id":"plus_monthly"}}`,
			authHeader: "",
			mockFSSetup: func(m *MockFirebaseSubGateway) {
				m.On("SetUserSubscription", mock.Anything, "user-123", authentication.PlanPlus, "", authentication.SubscriptionSourceIAP, int64(0)).Return(nil)
			},
			expectedError: false,
		},
		"should set expiration on CANCELLATION": {
			body:       `{"api_version":"4.0","event":{"type":"CANCELLATION","app_user_id":"user-123","entitlement_ids":["plus"],"expiration_at_ms":1700000000000,"product_id":"plus_monthly"}}`,
			authHeader: "",
			mockFSSetup: func(m *MockFirebaseSubGateway) {
				m.On("SetUserSubscription", mock.Anything, "user-123", authentication.PlanPlus, "", authentication.SubscriptionSourceIAP, int64(1700000000)).Return(nil)
			},
			expectedError: false,
		},
		"should downgrade to free on EXPIRATION": {
			body:       `{"api_version":"4.0","event":{"type":"EXPIRATION","app_user_id":"user-123","entitlement_ids":["plus"],"expiration_at_ms":1700000000000,"product_id":"plus_monthly"}}`,
			authHeader: "",
			mockFSSetup: func(m *MockFirebaseSubGateway) {
				m.On("SetUserSubscription", mock.Anything, "user-123", authentication.PlanFree, "", authentication.SubscriptionSourceIAP, int64(0)).Return(nil)
			},
			expectedError: false,
		},
		"should ignore unknown event types": {
			body:          `{"api_version":"4.0","event":{"type":"TEST","app_user_id":"user-123","entitlement_ids":["plus"]}}`,
			authHeader:    "",
			mockFSSetup:   func(m *MockFirebaseSubGateway) {},
			expectedError: false,
		},
		"should reject missing app_user_id": {
			body:          `{"api_version":"4.0","event":{"type":"INITIAL_PURCHASE","app_user_id":"","entitlement_ids":["plus"]}}`,
			authHeader:    "",
			mockFSSetup:   func(m *MockFirebaseSubGateway) {},
			expectedError: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mockMP := new(MockMPGateway)
			mockFS := new(MockFirebaseSubGateway)
			tc.mockFSSetup(mockFS)

			s := NewSubscription(mockMP, mockFS, new(MockAppSettingsReader))

			err := s.HandleRevenueCatWebhook(context.Background(), tc.authHeader, []byte(tc.body))

			if tc.expectedError {
				assert.Error(t, err)
				assert.ErrorIs(t, err, ErrRevenueCatWebhook)
			} else {
				assert.NoError(t, err)
			}
			mockFS.AssertExpectations(t)
		})
	}
}
