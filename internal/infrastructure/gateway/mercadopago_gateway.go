package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"
)

const (
	_createPlanPath = "/preapproval_plan"

	envMPBaseURL      = "MERCADOPAGO_BASE_URL"
	envMPCheckoutBase = "MERCADOPAGO_CHECKOUT_BASE"

	defaultMPBaseURL      = "https://api.mercadopago.com"
	defaultMPCheckoutBase = "https://www.mercadopago.com.br"
)

func NewMercadoPagoGateway() *MercadoPagoGateway {
	return &MercadoPagoGateway{
		httpClient:   &http.Client{Timeout: 10 * time.Second},
		accessToken:  os.Getenv("MERCADOPAGO_ACCESS_TOKEN"),
		baseURL:      getEnv(envMPBaseURL, defaultMPBaseURL),
		checkoutBase: getEnv(envMPCheckoutBase, defaultMPCheckoutBase),
	}
}

func (g *MercadoPagoGateway) CreatePlan(ctx context.Context, plan SubscriptionPlanConfig, reason, backURL string) (string, string, error) {
	req := MPCreatePlanRequest{
		Reason:  reason,
		BackURL: backURL,
		AutoRecurring: MPAutoRecurring{
			Frequency:         plan.Frequency,
			FrequencyType:     plan.FrequencyType,
			TransactionAmount: plan.Price,
			CurrencyID:        plan.Currency,
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", "", fmt.Errorf("error marshaling request: %w", err)
	}

	httpReq, err := g.buildHTTPCreateRequest(ctx, _createPlanPath, body)
	if err != nil {
		return "", "", fmt.Errorf("error creating request: %w", err)
	}

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return "", "", fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		var errorResponse interface{}
		_ = json.NewDecoder(resp.Body).Decode(&errorResponse)
		errJSON, _ := json.Marshal(errorResponse)
		return "", "", fmt.Errorf("mercado pago api returned status: %d error: %s", resp.StatusCode, string(errJSON))
	}

	var res MPCreatePlanResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", "", fmt.Errorf("error decoding response: %w", err)
	}

	return res.ID, res.InitPoint, nil
}

func (g *MercadoPagoGateway) BuildPlanCheckoutURL(planID, externalReference string) string {
	checkoutURL := fmt.Sprintf("%s/subscriptions/checkout?preapproval_plan_id=%s", g.checkoutBase, url.QueryEscape(planID))
	if externalReference != "" {
		checkoutURL += "&external_reference=" + url.QueryEscape(externalReference)
	}
	return checkoutURL
}

func (g *MercadoPagoGateway) buildHTTPCreateRequest(ctx context.Context, path string, body []byte) (*http.Request, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s%s", g.baseURL, path), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", g.accessToken))
	httpReq.Header.Set("Content-Type", "application/json")

	return httpReq, nil
}

func (g *MercadoPagoGateway) GetSubscription(ctx context.Context, id string) (MPSubscription, error) {
	requestURL := fmt.Sprintf("%s/preapproval/%s", g.baseURL, id)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return MPSubscription{}, fmt.Errorf("error creating request: %w", err)
	}

	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", g.accessToken))

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return MPSubscription{}, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorResponse interface{}
		_ = json.NewDecoder(resp.Body).Decode(&errorResponse)
		errJSON, _ := json.Marshal(errorResponse)
		return MPSubscription{}, fmt.Errorf("mercado pago api returned status: %d response: %s", resp.StatusCode, string(errJSON))
	}

	var res MPSubscription
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return MPSubscription{}, fmt.Errorf("error decoding response: %w", err)
	}

	return res, nil
}

func (g *MercadoPagoGateway) CancelSubscription(ctx context.Context, id string) error {
	requestURL := fmt.Sprintf("%s/preapproval/%s", g.baseURL, id)

	req := map[string]string{
		"status": "cancelled",
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("error marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, requestURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", g.accessToken))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorResponse interface{}
		_ = json.NewDecoder(resp.Body).Decode(&errorResponse)
		errJSON, _ := json.Marshal(errorResponse)
		return fmt.Errorf("mercado pago api returned status: %d response: %s", resp.StatusCode, string(errJSON))
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
