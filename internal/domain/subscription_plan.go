package domain

type SubscriptionPlan struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Price         float64 `json:"price"`
	Currency      string  `json:"currency"`
	Frequency     int     `json:"frequency"`
	FrequencyType string  `json:"frequency_type"`
	IsActive      bool    `json:"is_active"`
	StripePriceID string  `json:"-"`
}
