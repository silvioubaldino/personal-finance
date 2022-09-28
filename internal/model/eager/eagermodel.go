package eager

import (
	"personal-finance/internal/model"
	"time"
)

type Transaction struct {
	ID            int               `json:"id,omitempty"`
	Description   string            `json:"description,omitempty"`
	Amount        float64           `json:"amount"`
	WalletID      int               `json:"-"`
	Wallet        model.Wallet      `json:"wallets,omitempty"`
	TypePaymentID int               `json:"-"`
	TypePayment   model.TypePayment `json:"type_payments,omitempty"`
	CategoryID    int               `json:"-"`
	Category      model.Category    `json:"categories,omitempty"`
	DateCreate    time.Time         `json:"date_create"`
	DateUpdate    time.Time         `json:"date_update"`
}
