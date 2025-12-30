package domain

import "time"

type UserPreferences struct {
	UserID     string    `json:"user_id"`
	Language   string    `json:"language"`
	Currency   string    `json:"currency"`
	DateCreate time.Time `json:"date_create"`
	DateUpdate time.Time `json:"date_update"`
}

const (
	DefaultLanguage = "pt-BR"
	DefaultCurrency = "BRL"
)
