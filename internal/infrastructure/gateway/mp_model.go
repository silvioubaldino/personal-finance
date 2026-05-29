package gateway

import "net/http"

type MPSubscription struct {
	ID                string          `json:"id"`
	Status            string          `json:"status"`
	ExternalReference string          `json:"external_reference"`
	NextPaymentDate   string          `json:"next_payment_date"`
	DateCreated       string          `json:"date_created"`
	AutoRecurring     MPAutoRecurring `json:"auto_recurring"`
}

type MercadoPagoGateway struct {
	httpClient   *http.Client
	accessToken  string
	baseURL      string
	checkoutBase string
}

type SubscriptionPlanConfig struct {
	Price         float64
	Currency      string
	Frequency     int
	FrequencyType string
}

type MPAutoRecurring struct {
	Frequency         int     `json:"frequency"`
	FrequencyType     string  `json:"frequency_type"`
	TransactionAmount float64 `json:"transaction_amount"`
	CurrencyID        string  `json:"currency_id"`
	StartDate         string  `json:"start_date,omitempty"`
}

// MPCreatePlanRequest is the body for POST /preapproval_plan. It intentionally has
// no payer_email: the subscriber's account (and email) is captured at the plan's
// checkout, so there is no email-mismatch rejection.
type MPCreatePlanRequest struct {
	Reason            string          `json:"reason"`
	BackURL           string          `json:"back_url"`
	ExternalReference string          `json:"external_reference,omitempty"`
	AutoRecurring     MPAutoRecurring `json:"auto_recurring"`
}

type MPCreatePlanResponse struct {
	ID        string `json:"id"`
	InitPoint string `json:"init_point"`
}
