package domain

import (
	"time"

	"github.com/google/uuid"
)

type Movement struct {
	ID            *uuid.UUID  `json:"id,omitempty" gorm:"primaryKey"`
	Description   string      `json:"description,omitempty"`
	Amount        float64     `json:"amount"`
	Date          *time.Time  `json:"date"`
	UserID        string      `json:"user_id"`
	IsPaid        bool        `json:"is_paid"`
	IsRecurrent   bool        `json:"is_recurrent"`
	RecurrentID   *uuid.UUID  `json:"recurrent_id"`
	WalletID      *uuid.UUID  `json:"wallet_id,omitempty"`
	Wallet        Wallet      `json:"wallets,omitempty"`
	TypePayment   TypePayment `json:"type_payment,omitempty"`
	CategoryID    *uuid.UUID  `json:"category_id,omitempty"`
	Category      Category    `json:"categories,omitempty"`
	SubCategoryID *uuid.UUID  `json:"sub_category_id,omitempty"`
	SubCategory   SubCategory `json:"sub_categories,omitempty"`
	DateCreate    time.Time   `json:"date_create"`
	DateUpdate    time.Time   `json:"date_update"`
}

func (m Movement) ShouldCreateRecurrent() bool {
	return m.IsRecurrent && m.RecurrentID == nil
}

type MovementList []Movement

func (ml MovementList) GetPaidMovements() MovementList {
	var paidList MovementList
	for _, movement := range ml {
		if movement.IsPaid {
			paidList = append(paidList, movement)
		}
	}
	return paidList
}

func (ml MovementList) GetExpenseMovements() MovementList {
	var expenseList MovementList
	for _, movement := range ml {
		if movement.Amount < 0 {
			expenseList = append(expenseList, movement)
		}
	}
	return expenseList
}

func (ml MovementList) GetIncomeMovements() MovementList {
	var expenseList MovementList
	for _, movement := range ml {
		if movement.Amount > 0 {
			expenseList = append(expenseList, movement)
		}
	}
	return expenseList
}

func (ml MovementList) GetSumByCategory() map[*uuid.UUID]float64 {
	m := make(map[*uuid.UUID]float64)
	for _, movement := range ml {
		if _, ok := m[movement.CategoryID]; !ok {
			m[movement.Category.ID] = movement.Amount
		} else {
			m[movement.Category.ID] += movement.Amount
		}
	}
	return m
}

func (m Movement) ReverseAmount() float64 {
	return -m.Amount
}
