package domain

type SubscriptionPlan struct {
	ID            string
	Name          string
	Price         float64
	Currency      string
	Frequency     int
	FrequencyType string
	IsActive      bool
}
