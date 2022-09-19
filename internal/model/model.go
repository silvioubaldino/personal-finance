package model

import "time"

type (
	Wallet struct {
		ID          int       `json:"id,omitempty" gorm:"primaryKey"`
		Description string    `json:"description,omitempty"`
		Balance     float64   `json:"balance"`
		DateCreate  time.Time `json:"date_create"`
		DateUpdate  time.Time `json:"date_update"`
	}

	TypePayment struct {
		ID          int       `json:"ID,omitempty" gorm:"primaryKey"`
		Description string    `json:"description,omitempty"`
		DateCreate  time.Time `json:"date_create"`
		DateUpdate  time.Time `json:"date_update"`
	}

	Category struct {
		ID          int       `json:"id,omitempty" gorm:"primaryKey"`
		Description string    `json:"description,omitempty"`
		DateCreate  time.Time `json:"date_create"`
		DateUpdate  time.Time `json:"date_update"`
	}

	Transaction struct {
		ID          int         `json:"ID,omitempty" gorm:"primaryKey"`
		Description string      `json:"description,omitempty"`
		Amount      float64     `json:"amount"`
		Wallet      Wallet      `json:"wallet,omitempty"`
		TypePayment TypePayment `json:"typepayment"`
		Category    Category    `json:"categories,omitempty"`
	}
)
