package domain

import "time"

type User struct {
	ID        string    `json:"id"`
	Language  string    `json:"language"`
	Currency  string    `json:"currency"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

const (
	DefaultLanguage = "pt-BR"
	DefaultCurrency = "BRL"
)
