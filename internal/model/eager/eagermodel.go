package eager

import (
	"time"

	"personal-finance/internal/model"
)

type Transaction struct {
	ID            int               `json:"id,omitempty"`
	Description   string            `json:"description,omitempty"`
	Amount        float64           `json:"amount"`
	Date          time.Time         `json:"date"`
	WalletID      int               `json:"-"`
	Wallet        model.Wallet      `json:"wallets,omitempty"`
	TypePaymentID int               `json:"-"`
	TypePayment   model.TypePayment `json:"type_payments,omitempty"`
	CategoryID    int               `json:"-"`
	Category      model.Category    `json:"categories,omitempty"`
	DateCreate    time.Time         `json:"date_create"`
	DateUpdate    time.Time         `json:"date_update"`
}
