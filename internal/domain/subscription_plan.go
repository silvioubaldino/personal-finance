package domain

type SubscriptionPlan struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Price         float64 `json:"price"`
	Currency      string  `json:"currency"`
	Frequency     int     `json:"frequency"`
	FrequencyType string  `json:"frequency_type"`
	IsActive      bool    `json:"is_active"`
	// StripePriceID is the Stripe price (price_...) used to open the web Checkout Session.
	// Hidden from API responses.
	StripePriceID string `json:"-"`
}
