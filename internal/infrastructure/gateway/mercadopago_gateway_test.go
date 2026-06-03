package gateway

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestGateway(baseURL string) *MercadoPagoGateway {
	return &MercadoPagoGateway{
		httpClient:  &http.Client{Timeout: 5 * time.Second},
		accessToken: "TEST-TOKEN",
		baseURL:     baseURL,
		reason:      "Personal Finance Subscription",
		backURL:     "https://default.back.url",
		currency:    "BRL",
	}
}

func TestCreateSubscriptionURL_SendsPendingStatusAndReturnsInitPoint(t *testing.T) {
	var captured map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/preapproval", r.URL.Path)
		assert.Equal(t, "Bearer TEST-TOKEN", r.Header.Get("Authorization"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(body, &captured))

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"sub-123","init_point":"https://mp.com/checkout/sub-123","status":"pending"}`))
	}))
	defer server.Close()

	g := newTestGateway(server.URL)

	plan := SubscriptionPlanConfig{
		Price:         9.90,
		Currency:      "BRL",
		Frequency:     1,
		FrequencyType: "months",
	}

	url, err := g.CreateSubscriptionURL(
		context.Background(),
		"user@app.com",
		"user-123|plus_monthly|red-uuid",
		"https://app.com/return",
		plan,
	)
	require.NoError(t, err)

	// init_point from the response is returned to the caller.
	assert.Equal(t, "https://mp.com/checkout/sub-123", url)

	// Core of the fix: the no-plan + pending-payment flow.
	assert.Equal(t, "pending", captured["status"])
	assert.Equal(t, "user@app.com", captured["payer_email"])
	assert.Equal(t, "user-123|plus_monthly|red-uuid", captured["external_reference"])
	assert.Equal(t, "https://app.com/return", captured["back_url"])

	// Must NOT use the associated-plan / authorized-card flow.
	_, hasPlan := captured["preapproval_plan_id"]
	assert.False(t, hasPlan, "must not send preapproval_plan_id in the pending flow")
	_, hasCardToken := captured["card_token_id"]
	assert.False(t, hasCardToken, "must not send card_token_id in the pending flow")

	autoRecurring, ok := captured["auto_recurring"].(map[string]any)
	require.True(t, ok, "auto_recurring must be present")
	assert.Equal(t, 9.90, autoRecurring["transaction_amount"])
	assert.Equal(t, "BRL", autoRecurring["currency_id"])
	assert.Equal(t, float64(1), autoRecurring["frequency"])
	assert.Equal(t, "months", autoRecurring["frequency_type"])
}
