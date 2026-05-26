package usecase

import (
	"context"
	"errors"
	"testing"

	"personal-finance/internal/domain"
	"personal-finance/internal/infrastructure/gateway"
	"personal-finance/internal/infrastructure/repository"
	"personal-finance/internal/plataform/authentication"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockMPGateway struct {
	mock.Mock
}

func (m *MockMPGateway) CreateSubscriptionURL(ctx context.Context, payerEmail, externalReference, backURL string, plan gateway.SubscriptionPlanConfig) (string, error) {
	args := m.Called(ctx, payerEmail, externalReference, backURL, plan)
	return args.String(0), args.Error(1)
}

type MockSubscriptionPlanRepo struct {
	mock.Mock
}

func (m *MockSubscriptionPlanRepo) Create(ctx context.Context, plan domain.SubscriptionPlan) error {
	args := m.Called(ctx, plan)
	return args.Error(0)
}

func (m *MockSubscriptionPlanRepo) FindActive(ctx context.Context) ([]domain.SubscriptionPlan, error) {
	args := m.Called(ctx)
	return args.Get(0).([]domain.SubscriptionPlan), args.Error(1)
}

func (m *MockSubscriptionPlanRepo) FindActiveByID(ctx context.Context, id string) (domain.SubscriptionPlan, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(domain.SubscriptionPlan), args.Error(1)
}

func (m *MockSubscriptionPlanRepo) FindIDByStoreProduct(ctx context.Context, store, productID string) (string, error) {
	args := m.Called(ctx, store, productID)
	return args.String(0), args.Error(1)
}

func (m *MockMPGateway) GetSubscription(ctx context.Context, id string) (gateway.MPSubscription, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(gateway.MPSubscription), args.Error(1)
}

func (m *MockMPGateway) CancelSubscription(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

type MockSubscriptionRepo struct {
	mock.Mock
}

func (m *MockSubscriptionRepo) Upsert(ctx context.Context, sub domain.Subscription) (domain.Subscription, error) {
	args := m.Called(ctx, sub)
	return args.Get(0).(domain.Subscription), args.Error(1)
}

func (m *MockSubscriptionRepo) List(ctx context.Context, filter repository.SubscriptionListFilter) ([]domain.Subscription, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]domain.Subscription), args.Error(1)
}

type MockFirebaseSubGateway struct {
	mock.Mock
}

func (m *MockFirebaseSubGateway) SetUserSubscription(ctx context.Context, userID string, plan authentication.Plan, mpSubscriptionID string, subscriptionSource authentication.SubscriptionSource, expiresAt int64) error {
	args := m.Called(ctx, userID, plan, mpSubscriptionID, subscriptionSource, expiresAt)
	return args.Error(0)
}

var monthlyPlan = domain.SubscriptionPlan{
	ID:            "plus_monthly",
	Name:          "Plus Mensal",
	Price:         9.90,
	Currency:      "BRL",
	Frequency:     1,
	FrequencyType: "months",
	IsActive:      true,
}

var monthlyPlanConfig = gateway.SubscriptionPlanConfig{
	Price:         9.90,
	Currency:      "BRL",
	Frequency:     1,
	FrequencyType: "months",
}

func TestSubscription_CreateCheckout(t *testing.T) {
	tests := map[string]struct {
		authCtx       *authentication.AuthContext
		planID        string
		backURL       string
		mockSetup     func(*MockMPGateway, *MockSubscriptionPlanRepo)
		expectedURL   string
		expectedError error
	}{
		"should create checkout with back_url": {
			authCtx: &authentication.AuthContext{UserID: "user-123"},
			planID:  "plus_monthly",
			backURL: "https://api.domain.com/subscription/return",
			mockSetup: func(m *MockMPGateway, p *MockSubscriptionPlanRepo) {
				p.On("FindActiveByID", mock.Anything, "plus_monthly").Return(monthlyPlan, nil)
				m.On("CreateSubscriptionURL", mock.Anything, mock.Anything, "user-123|plus_monthly", "https://api.domain.com/subscription/return", monthlyPlanConfig).
					Return("http://mp.com/pay", nil)
			},
			expectedURL:   "http://mp.com/pay",
			expectedError: nil,
		},
		"should create checkout with empty back_url": {
			authCtx: &authentication.AuthContext{UserID: "user-123"},
			planID:  "plus_monthly",
			backURL: "",
			mockSetup: func(m *MockMPGateway, p *MockSubscriptionPlanRepo) {
				p.On("FindActiveByID", mock.Anything, "plus_monthly").Return(monthlyPlan, nil)
				m.On("CreateSubscriptionURL", mock.Anything, mock.Anything, "user-123|plus_monthly", "", monthlyPlanConfig).
					Return("http://mp.com/pay", nil)
			},
			expectedURL:   "http://mp.com/pay",
			expectedError: nil,
		},
		"should return unauthorized if no user in context": {
			authCtx:       nil,
			planID:        "plus_monthly",
			backURL:       "",
			mockSetup:     func(m *MockMPGateway, p *MockSubscriptionPlanRepo) {},
			expectedURL:   "",
			expectedError: ErrUnauthorized,
		},
		"should return error if plan not found": {
			authCtx: &authentication.AuthContext{UserID: "user-123"},
			planID:  "unknown_plan",
			backURL: "",
			mockSetup: func(m *MockMPGateway, p *MockSubscriptionPlanRepo) {
				p.On("FindActiveByID", mock.Anything, "unknown_plan").Return(domain.SubscriptionPlan{}, errors.New("not found"))
			},
			expectedURL:   "",
			expectedError: ErrSubscriptionPlanNotFound,
		},
		"should return gateway error if MP fails": {
			authCtx: &authentication.AuthContext{UserID: "user-123"},
			planID:  "plus_monthly",
			backURL: "",
			mockSetup: func(m *MockMPGateway, p *MockSubscriptionPlanRepo) {
				p.On("FindActiveByID", mock.Anything, "plus_monthly").Return(monthlyPlan, nil)
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
			mockPlan := new(MockSubscriptionPlanRepo)
			tc.mockSetup(mockMP, mockPlan)

			s := NewSubscription(mockMP, mockFS, mockPlan, nil, nil)

			ctx := context.Background()
			if tc.authCtx != nil {
				ctx = authentication.ContextWithAuth(ctx, *tc.authCtx)
			}

			resp, err := s.CreateCheckout(ctx, tc.planID, tc.backURL, "")

			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tc.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedURL, resp)
			}
			mockMP.AssertExpectations(t)
			mockPlan.AssertExpectations(t)
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

			s := NewSubscription(mockMP, mockFS, new(MockSubscriptionPlanRepo), nil, nil)

			err := s.HandleWebhook(context.Background(), "", "", []byte(tc.body))

			assert.Equal(t, tc.expectedError, err)
			mockMP.AssertExpectations(t)
			mockFS.AssertExpectations(t)
		})
	}
}

func TestSubscription_HandleRevenueCatWebhook(t *testing.T) {
	t.Setenv("REVENUECAT_WEBHOOK_AUTH_KEY", "test-key")

	tests := map[string]struct {
		body          string
		authHeader    string
		mockFSSetup   func(*MockFirebaseSubGateway)
		expectedError bool
	}{
		"should update to plus on INITIAL_PURCHASE": {
			body:       `{"api_version":"4.0","event":{"type":"INITIAL_PURCHASE","app_user_id":"user-123","entitlement_ids":["plus"],"expiration_at_ms":1700000000000,"product_id":"plus_monthly"}}`,
			authHeader: "Bearer test-key",
			mockFSSetup: func(m *MockFirebaseSubGateway) {
				m.On("SetUserSubscription", mock.Anything, "user-123", authentication.PlanPlus, "", authentication.SubscriptionSourceIAP, int64(0)).Return(nil)
			},
			expectedError: false,
		},
		"should update to plus on RENEWAL": {
			body:       `{"api_version":"4.0","event":{"type":"RENEWAL","app_user_id":"user-123","entitlement_ids":["plus"],"expiration_at_ms":1700000000000,"product_id":"plus_monthly"}}`,
			authHeader: "Bearer test-key",
			mockFSSetup: func(m *MockFirebaseSubGateway) {
				m.On("SetUserSubscription", mock.Anything, "user-123", authentication.PlanPlus, "", authentication.SubscriptionSourceIAP, int64(0)).Return(nil)
			},
			expectedError: false,
		},
		"should set expiration on CANCELLATION": {
			body:       `{"api_version":"4.0","event":{"type":"CANCELLATION","app_user_id":"user-123","entitlement_ids":["plus"],"expiration_at_ms":1700000000000,"product_id":"plus_monthly"}}`,
			authHeader: "Bearer test-key",
			mockFSSetup: func(m *MockFirebaseSubGateway) {
				m.On("SetUserSubscription", mock.Anything, "user-123", authentication.PlanPlus, "", authentication.SubscriptionSourceIAP, int64(1700000000)).Return(nil)
			},
			expectedError: false,
		},
		"should downgrade to free on EXPIRATION": {
			body:       `{"api_version":"4.0","event":{"type":"EXPIRATION","app_user_id":"user-123","entitlement_ids":["plus"],"expiration_at_ms":1700000000000,"product_id":"plus_monthly"}}`,
			authHeader: "Bearer test-key",
			mockFSSetup: func(m *MockFirebaseSubGateway) {
				m.On("SetUserSubscription", mock.Anything, "user-123", authentication.PlanFree, "", authentication.SubscriptionSourceIAP, int64(0)).Return(nil)
			},
			expectedError: false,
		},
		"should ignore unknown event types": {
			body:          `{"api_version":"4.0","event":{"type":"TEST","app_user_id":"user-123","entitlement_ids":["plus"]}}`,
			authHeader:    "Bearer test-key",
			mockFSSetup:   func(m *MockFirebaseSubGateway) {},
			expectedError: false,
		},
		"should reject missing app_user_id": {
			body:          `{"api_version":"4.0","event":{"type":"INITIAL_PURCHASE","app_user_id":"","entitlement_ids":["plus"]}}`,
			authHeader:    "Bearer test-key",
			mockFSSetup:   func(m *MockFirebaseSubGateway) {},
			expectedError: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mockMP := new(MockMPGateway)
			mockFS := new(MockFirebaseSubGateway)
			tc.mockFSSetup(mockFS)

			s := NewSubscription(mockMP, mockFS, new(MockSubscriptionPlanRepo), nil, nil)

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

func TestSubscription_HandleWebhook_MirrorsToDB(t *testing.T) {
	mpResponse := gateway.MPSubscription{
		ID:                "sub-123",
		Status:            "authorized",
		ExternalReference: "user-123|plus_monthly",
		DateCreated:       "2026-01-10T12:00:00.000-03:00",
		NextPaymentDate:   "2026-02-10T12:00:00.000-03:00",
		AutoRecurring: gateway.MPAutoRecurring{
			TransactionAmount: 9.90,
			CurrencyID:        "BRL",
		},
	}

	mockMP := new(MockMPGateway)
	mockFS := new(MockFirebaseSubGateway)
	mockSub := new(MockSubscriptionRepo)

	mockMP.On("GetSubscription", mock.Anything, "sub-123").Return(mpResponse, nil)
	mockSub.On("Upsert", mock.Anything, mock.MatchedBy(func(sub domain.Subscription) bool {
		return sub.UserID == "user-123" &&
			sub.Source == domain.SubscriptionSourceMercadoPago &&
			sub.ExternalID == "sub-123" &&
			sub.PlanID == "plus_monthly" &&
			sub.Status == domain.SubscriptionStatusActive &&
			sub.CurrentPrice == 9.90 &&
			sub.Currency == "BRL" &&
			!sub.StartedAt.IsZero() &&
			sub.CurrentPeriodEnd != nil &&
			sub.CancelledAt == nil
	})).Return(domain.Subscription{}, nil)
	mockFS.On("SetUserSubscription", mock.Anything, "user-123", authentication.PlanPlus, "sub-123", authentication.SubscriptionSourceMP, int64(0)).Return(nil)

	s := NewSubscription(mockMP, mockFS, new(MockSubscriptionPlanRepo), mockSub, nil)

	body := []byte(`{"action":"created","type":"subscription_preapproval","data":{"id":"sub-123"}}`)
	err := s.HandleWebhook(context.Background(), "", "", body)
	assert.NoError(t, err)

	mockMP.AssertExpectations(t)
	mockSub.AssertExpectations(t)
	mockFS.AssertExpectations(t)
}

func TestSubscription_HandleWebhook_StampsCancelledAt(t *testing.T) {
	mpResponse := gateway.MPSubscription{
		ID:                "sub-123",
		Status:            "cancelled",
		ExternalReference: "user-123",
		DateCreated:       "2026-01-10T12:00:00.000-03:00",
	}

	mockMP := new(MockMPGateway)
	mockFS := new(MockFirebaseSubGateway)
	mockSub := new(MockSubscriptionRepo)

	mockMP.On("GetSubscription", mock.Anything, "sub-123").Return(mpResponse, nil)
	mockSub.On("Upsert", mock.Anything, mock.MatchedBy(func(sub domain.Subscription) bool {
		return sub.Status == domain.SubscriptionStatusCancelled && sub.CancelledAt != nil
	})).Return(domain.Subscription{}, nil)
	mockFS.On("SetUserSubscription", mock.Anything, "user-123", authentication.PlanFree, "", authentication.SubscriptionSourceMP, int64(0)).Return(nil)

	s := NewSubscription(mockMP, mockFS, new(MockSubscriptionPlanRepo), mockSub, nil)

	body := []byte(`{"action":"updated","type":"subscription_preapproval","data":{"id":"sub-123"}}`)
	err := s.HandleWebhook(context.Background(), "", "", body)
	assert.NoError(t, err)

	mockSub.AssertExpectations(t)
}

func TestSubscription_HandleRevenueCatWebhook_MirrorsToDB(t *testing.T) {
	t.Setenv("REVENUECAT_WEBHOOK_AUTH_KEY", "test-key")

	body := []byte(`{
		"api_version":"4.0",
		"event":{
			"type":"INITIAL_PURCHASE",
			"app_user_id":"user-123",
			"store":"APP_STORE",
			"original_transaction_id":"orig-tx-1",
			"product_id":"plus_monthly",
			"purchased_at_ms":1700000000000,
			"expiration_at_ms":1702678400000,
			"price_in_purchased_currency":9.99,
			"currency":"BRL",
			"entitlement_ids":["plus"]
		}
	}`)

	mockMP := new(MockMPGateway)
	mockFS := new(MockFirebaseSubGateway)
	mockSub := new(MockSubscriptionRepo)
	mockPlan := new(MockSubscriptionPlanRepo)

	mockPlan.On("FindIDByStoreProduct", mock.Anything, "APP_STORE", "plus_monthly").Return("plus_monthly", nil)
	mockSub.On("Upsert", mock.Anything, mock.MatchedBy(func(sub domain.Subscription) bool {
		return sub.UserID == "user-123" &&
			sub.Source == domain.SubscriptionSourceApple &&
			sub.ExternalID == "orig-tx-1" &&
			sub.ExternalProductID == "plus_monthly" &&
			sub.PlanID == "plus_monthly" &&
			sub.Status == domain.SubscriptionStatusActive &&
			sub.CurrentPrice == 9.99 &&
			sub.Currency == "BRL" &&
			!sub.StartedAt.IsZero() &&
			sub.CurrentPeriodEnd != nil &&
			sub.CancelledAt == nil
	})).Return(domain.Subscription{}, nil)
	mockFS.On("SetUserSubscription", mock.Anything, "user-123", authentication.PlanPlus, "", authentication.SubscriptionSourceIAP, int64(0)).Return(nil)

	s := NewSubscription(mockMP, mockFS, mockPlan, mockSub, nil)

	err := s.HandleRevenueCatWebhook(context.Background(), "Bearer test-key", body)
	assert.NoError(t, err)

	mockPlan.AssertExpectations(t)
	mockSub.AssertExpectations(t)
	mockFS.AssertExpectations(t)
}

func TestSubscription_SummarizeSubscriptions(t *testing.T) {
	mockSub := new(MockSubscriptionRepo)
	mockSub.On("List", mock.Anything, repository.SubscriptionListFilter{}).Return([]domain.Subscription{
		{Source: domain.SubscriptionSourceMercadoPago, Status: domain.SubscriptionStatusActive, CurrentPrice: 19.90, Currency: "BRL"},
		{Source: domain.SubscriptionSourceMercadoPago, Status: domain.SubscriptionStatusActive, CurrentPrice: 19.90, Currency: "BRL"},
		{Source: domain.SubscriptionSourceMercadoPago, Status: domain.SubscriptionStatusCancelled, CurrentPrice: 19.90, Currency: "BRL"},
		{Source: domain.SubscriptionSourceApple, Status: domain.SubscriptionStatusActive, CurrentPrice: 9.99, Currency: "USD"},
		{Source: domain.SubscriptionSourceGoogle, Status: domain.SubscriptionStatusExpired, CurrentPrice: 9.99, Currency: "USD"},
	}, nil)

	s := NewSubscription(nil, nil, nil, mockSub, nil)

	summary, err := s.SummarizeSubscriptions(context.Background(), repository.SubscriptionListFilter{})
	assert.NoError(t, err)

	assert.Equal(t, 5, summary.TotalSubscriptions)
	assert.Equal(t, 3, summary.ActiveSubscriptions)
	assert.Equal(t, 3, summary.BySource["mercadopago"])
	assert.Equal(t, 1, summary.BySource["apple"])
	assert.Equal(t, 1, summary.BySource["google"])
	assert.Equal(t, 3, summary.ByStatus["active"])
	assert.Equal(t, 1, summary.ByStatus["cancelled"])
	assert.Equal(t, 1, summary.ByStatus["expired"])
	assert.InDelta(t, 39.80, summary.ActiveRevenueByCurrency["BRL"], 0.001)
	assert.InDelta(t, 9.99, summary.ActiveRevenueByCurrency["USD"], 0.001)
}

func TestSubscription_SummarizeSubscriptions_EmptyWithNilRepo(t *testing.T) {
	s := NewSubscription(nil, nil, nil, nil, nil)

	summary, err := s.SummarizeSubscriptions(context.Background(), repository.SubscriptionListFilter{})
	assert.NoError(t, err)
	assert.Equal(t, 0, summary.TotalSubscriptions)
	assert.NotNil(t, summary.BySource)
	assert.NotNil(t, summary.ByStatus)
	assert.NotNil(t, summary.ActiveRevenueByCurrency)
}

func TestSubscription_HandleRevenueCatWebhook_StampsCancelledAt(t *testing.T) {
	t.Setenv("REVENUECAT_WEBHOOK_AUTH_KEY", "test-key")

	body := []byte(`{
		"api_version":"4.0",
		"event":{
			"type":"CANCELLATION",
			"app_user_id":"user-123",
			"store":"PLAY_STORE",
			"original_transaction_id":"orig-tx-2",
			"product_id":"plus_monthly",
			"purchased_at_ms":1700000000000,
			"expiration_at_ms":1702678400000,
			"price_in_purchased_currency":9.99,
			"currency":"BRL"
		}
	}`)

	mockMP := new(MockMPGateway)
	mockFS := new(MockFirebaseSubGateway)
	mockSub := new(MockSubscriptionRepo)
	mockPlan := new(MockSubscriptionPlanRepo)

	mockPlan.On("FindIDByStoreProduct", mock.Anything, "PLAY_STORE", "plus_monthly").Return("plus_monthly", nil)
	mockSub.On("Upsert", mock.Anything, mock.MatchedBy(func(sub domain.Subscription) bool {
		return sub.UserID == "user-123" &&
			sub.Source == domain.SubscriptionSourceGoogle &&
			sub.Status == domain.SubscriptionStatusCancelled &&
			sub.CancelledAt != nil
	})).Return(domain.Subscription{}, nil)
	mockFS.On("SetUserSubscription", mock.Anything, "user-123", authentication.PlanPlus, "", authentication.SubscriptionSourceIAP, int64(1702678400)).Return(nil)

	s := NewSubscription(mockMP, mockFS, mockPlan, mockSub, nil)

	err := s.HandleRevenueCatWebhook(context.Background(), "Bearer test-key", body)
	assert.NoError(t, err)

	mockPlan.AssertExpectations(t)
	mockSub.AssertExpectations(t)
}
