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
	httpClient  *http.Client
	accessToken string
	baseURL     string
	reason      string
	currency    string
	backURL     string
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

type MPCreateSubscriptionRequest struct {
	Reason            string          `json:"reason"`
	ExternalReference string          `json:"external_reference"`
	BackURL           string          `json:"back_url"`
	AutoRecurring     MPAutoRecurring `json:"auto_recurring"`
	Status            string          `json:"status,omitempty"`
}
