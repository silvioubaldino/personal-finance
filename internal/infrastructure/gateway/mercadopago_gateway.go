package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

const (
	_createSubscriptionPath = "/preapproval"

	envMPBaseURL  = "MERCADOPAGO_BASE_URL"
	envMPReason   = "MERCADOPAGO_REASON"
	envMPBackURL  = "MERCADOPAGO_BACK_URL"
	envMPCurrency = "MERCADOPAGO_CURRENCY"

	defaultMPBaseURL  = "https://api.mercadopago.com"
	defaultMPReason   = "Personal Finance Subscription"
	defaultMPBackURL  = "https://personal-finance-frontend-v2.vercel.app/"
	defaultMPCurrency = "BRL"
)

func NewMercadoPagoGateway() *MercadoPagoGateway {
	mpBaseURL := getEnv(envMPBaseURL, defaultMPBaseURL)
	mpReason := getEnv(envMPReason, defaultMPReason)
	mpBackURL := getEnv(envMPBackURL, defaultMPBackURL)
	mpCurrency := getEnv(envMPCurrency, defaultMPCurrency)

	return &MercadoPagoGateway{
		httpClient:  &http.Client{Timeout: 10 * time.Second},
		accessToken: os.Getenv("MERCADOPAGO_ACCESS_TOKEN"),
		baseURL:     mpBaseURL,
		reason:      mpReason,
		backURL:     mpBackURL,
		currency:    mpCurrency,
	}
}

type MPCreateSubscriptionResponse struct {
	ID               string `json:"id"`
	InitPoint        string `json:"init_point"`
	SandboxInitPoint string `json:"sandbox_init_point"`
	Status           string `json:"status"`
}

func (g *MercadoPagoGateway) CreateSubscriptionURL(ctx context.Context, payerEmail, externalReference, backURL string, plan SubscriptionPlanConfig) (string, error) {
	req := g.buildMPRequest(payerEmail, externalReference, backURL, plan)

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("error marshaling request: %w", err)
	}

	httpReq, err := g.buildHTTPCreateRequest(ctx, _createSubscriptionPath, body)
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		var errorResponse interface{}
		_ = json.NewDecoder(resp.Body).Decode(&errorResponse)
		errJSON, _ := json.Marshal(errorResponse)
		return "", fmt.Errorf("mercado pago api returned status: %d error: %s", resp.StatusCode, string(errJSON))
	}

	var res MPCreateSubscriptionResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", fmt.Errorf("error decoding response: %w", err)
	}

	return res.InitPoint, nil
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
	url := fmt.Sprintf("%s/preapproval/%s", g.baseURL, id)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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
	url := fmt.Sprintf("%s/preapproval/%s", g.baseURL, id)

	req := map[string]string{
		"status": "cancelled",
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("error marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))
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

func (g *MercadoPagoGateway) buildMPRequest(payerEmail, externalReference, backURL string, plan SubscriptionPlanConfig) MPCreateSubscriptionRequest {
	startDate := time.Now().Add(1 * time.Hour).Format("2006-01-02T15:04:05.000-07:00")

	resolvedBackURL := backURL
	if resolvedBackURL == "" {
		resolvedBackURL = g.backURL
	}

	currency := plan.Currency
	if currency == "" {
		currency = g.currency
	}

	return MPCreateSubscriptionRequest{
		PayerEmail:        payerEmail,
		Reason:            g.reason,
		ExternalReference: externalReference,
		BackURL:           resolvedBackURL,
		AutoRecurring: MPAutoRecurring{
			Frequency:         plan.Frequency,
			FrequencyType:     plan.FrequencyType,
			TransactionAmount: plan.Price,
			CurrencyID:        currency,
			StartDate:         startDate,
		},
	}
}
