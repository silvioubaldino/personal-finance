package domain

import "time"

type UserDataExport struct {
	ExportedAt    time.Time               `json:"exported_at"`
	UserID        string                  `json:"user_id"`
	Preferences   *UserPreferences        `json:"preferences,omitempty"`
	Consents      []UserConsent           `json:"consents,omitempty"`
	Wallets       []Wallet                `json:"wallets,omitempty"`
	Categories    []Category              `json:"categories,omitempty"`
	SubCategories []SubCategory           `json:"sub_categories,omitempty"`
	Movements     []Movement              `json:"movements,omitempty"`
	Recurrents    []RecurrentMovement     `json:"recurrent_movements,omitempty"`
	CreditCards   []CreditCard            `json:"credit_cards,omitempty"`
	Invoices      []Invoice               `json:"invoices,omitempty"`
	Estimates     UserDataExportEstimates `json:"estimates,omitempty"`
}

type UserDataExportEstimates struct {
	Categories    []EstimateCategories    `json:"categories,omitempty"`
	SubCategories []EstimateSubCategories `json:"sub_categories,omitempty"`
}
