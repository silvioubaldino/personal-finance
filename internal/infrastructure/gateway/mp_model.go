package gateway

import "net/http"

type MPSubscription struct {
	ID                string `json:"id"`
	Status            string `json:"status"`
	ExternalReference string `json:"external_reference"`
	NextPaymentDate   string `json:"next_payment_date"`
}

type MercadoPagoGateway struct {
	httpClient  *http.Client
	accessToken string
	baseURL     string
	reason      string
	price       float64
	currency    string
	backURL     string
}

type MPAutoRecurring struct {
	Frequency         int     `json:"frequency"`
	FrequencyType     string  `json:"frequency_type"`
	TransactionAmount float64 `json:"transaction_amount"`
	CurrencyID        string  `json:"currency_id"`
	StartDate         string  `json:"start_date,omitempty"`
}

type MPCreateSubscriptionRequest struct {
	PayerEmail        string          `json:"payer_email"`
	Reason            string          `json:"reason"`
	ExternalReference string          `json:"external_reference"`
	BackURL           string          `json:"back_url"`
	AutoRecurring     MPAutoRecurring `json:"auto_recurring"`
	Status            string          `json:"status,omitempty"`
}
