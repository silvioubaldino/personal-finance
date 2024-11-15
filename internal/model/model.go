package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	TransactionStatusPaidID    = 1
	TransactionStatusPlannedID = 2
)

type (
	Wallet struct {
		ID             *uuid.UUID `json:"id,omitempty" gorm:"primaryKey"`
		Description    string     `json:"description,omitempty"`
		Balance        float64    `json:"balance"`
		UserID         string     `json:"user_id"`
		InitialBalance float64    `json:"initial_balance"`
		InitialDate    time.Time  `json:"initial_date"`
		DateCreate     time.Time  `json:"date_create"`
		DateUpdate     time.Time  `json:"date_update"`
	}

	TypePayment struct {
		ID          int       `json:"id,omitempty" gorm:"primaryKey"`
		Description string    `json:"description,omitempty"`
		UserID      string    `json:"user_id"`
		DateCreate  time.Time `json:"date_create"`
		DateUpdate  time.Time `json:"date_update"`
	}

	Category struct {
		ID            *uuid.UUID      `json:"id,omitempty" gorm:"primaryKey"`
		Description   string          `json:"description,omitempty"`
		UserID        string          `json:"user_id"`
		IsIncome      bool            `json:"is_income"`
		SubCategories SubCategoryList `json:"sub_categories"`
		DateCreate    time.Time       `json:"date_create"`
		DateUpdate    time.Time       `json:"date_update"`
	}

	SubCategory struct {
		ID          *uuid.UUID `json:"id,omitempty" gorm:"primaryKey"`
		Description string     `json:"description,omitempty"`
		UserID      string     `json:"user_id"`
		CategoryID  *uuid.UUID `json:"category_id,omitempty"`
		DateCreate  time.Time  `json:"date_create"`
		DateUpdate  time.Time  `json:"date_update"`
	}

	SubCategoryList []SubCategory

	EstimateCategories struct {
		ID               *uuid.UUID `json:"id" gorm:"primaryKey"`
		CategoryID       *uuid.UUID `json:"category_id"`
		CategoryName     string     `json:"category_name"`
		IsCategoryIncome bool       `json:"is_category_income"`
		Month            time.Month `json:"month"`
		Year             int        `json:"year"`
		Amount           float64    `json:"amount"`
		UserID           string     `json:"user_id"`
	}

	EstimateCategoriesList []EstimateCategories

	EstimateSubCategories struct {
		ID                 *uuid.UUID `json:"id" gorm:"primaryKey"`
		SubCategoryID      *uuid.UUID `json:"sub_category_id"`
		SubCategoryName    string     `json:"sub_category_name"`
		EstimateCategoryID *uuid.UUID `json:"estimate_category_id"`
		Month              time.Month `json:"month"`
		Year               int        `json:"year"`
		Amount             float64    `json:"amount"`
		UserID             string     `json:"user_id"`
	}

	TransactionStatus struct {
		ID          int       `json:"id,omitempty" gorm:"primaryKey"`
		Description string    `json:"description,omitempty"`
		DateCreate  time.Time `json:"date_create"`
		DateUpdate  time.Time `json:"date_update"`
	}

	Movement struct {
		ID            *uuid.UUID  `json:"id,omitempty" gorm:"primaryKey"`
		Description   string      `json:"description,omitempty"`
		Amount        float64     `json:"amount"`
		Date          *time.Time  `json:"date"`
		TransactionID *uuid.UUID  `json:"transaction_id,omitempty"`
		UserID        string      `json:"user_id"`
		IsPaid        bool        `json:"is_paid"`
		IsRecurrent   bool        `json:"is_recurrent"`
		RecurrentID   *uuid.UUID  `json:"recurrent_id"`
		StatusID      int         `json:"status_id,omitempty"`
		WalletID      *uuid.UUID  `json:"wallet_id,omitempty"`
		Wallet        Wallet      `json:"wallets,omitempty"`
		TypePaymentID int         `json:"type_payment_id,omitempty"`
		TypePayment   TypePayment `json:"type_payments,omitempty"`
		CategoryID    *uuid.UUID  `json:"category_id,omitempty"`
		Category      Category    `json:"categories,omitempty"`
		SubCategoryID *uuid.UUID  `json:"sub_category_id,omitempty"`
		SubCategory   SubCategory `json:"sub_categories,omitempty"`
		DateCreate    time.Time   `json:"date_create"`
		DateUpdate    time.Time   `json:"date_update"`
	}

	MovementList []Movement

	RecurrentMovement struct {
		ID            *uuid.UUID `json:"id,omitempty" gorm:"primaryKey"`
		Description   string     `json:"description,omitempty"`
		Amount        float64    `json:"amount"`
		InitialDate   *time.Time `json:"initial_date"`
		EndDate       *time.Time `json:"end_date"`
		UserID        string     `json:"user_id"`
		CategoryID    *uuid.UUID `json:"category_id,omitempty"`
		SubCategoryID *uuid.UUID `json:"sub_category_id,omitempty"`
		WalletID      *uuid.UUID `json:"wallet_id,omitempty"`
		TypePaymentID int        `json:"type_payment_id,omitempty"`
	}

	Transaction struct {
		TransactionID *uuid.UUID     `json:"transaction_id"`
		Estimate      *Movement      `json:"estimate,omitempty"`
		Consolidation *Consolidation `json:"consolidation,omitempty"`
		DoneList      MovementList   `json:"done_list"`
	}

	Consolidation struct {
		Estimated float64 `json:"estimated,omitempty"`
		Realized  float64 `json:"realized,omitempty"`
		Remaining float64 `json:"remaining"`
	}

	Balance struct {
		Expense       float64 `json:"expense"`
		Income        float64 `json:"income"`
		PeriodBalance float64 `json:"period_balance"`
	}

	Period struct {
		From time.Time `json:"from"`
		To   time.Time `json:"to"`
	}
)

func (t TransactionStatus) TableName() string {
	return "movement_status"
}

func (p *Period) Validate() error {
	now := time.Now()
	if p.From == p.To {
		return errors.New("date must be informed")
	}

	if p.From.IsZero() {
		p.From = now
	}
	if p.To.IsZero() {
		p.To = now
	}

	if p.From.After(p.To) {
		return errors.New("'from' must be before 'to'")
	}

	return nil
}

func BuildTransaction(estimate Movement, doneList MovementList) Transaction {
	pt := Transaction{
		TransactionID: estimate.ID,
		Estimate:      &estimate,
		Consolidation: &Consolidation{},
		DoneList:      doneList,
	}
	pt.Consolidate()
	return pt
}

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

func (el EstimateCategoriesList) GetEstimateByCategory() map[*uuid.UUID]float64 {
	m := make(map[*uuid.UUID]float64)
	for _, estimate := range el {
		if _, ok := m[estimate.CategoryID]; !ok {
			m[estimate.CategoryID] = estimate.Amount
		} else {
			m[estimate.CategoryID] += estimate.Amount
		}
	}
	return m
}

func (el EstimateCategoriesList) GetExpenseEstimates() EstimateCategoriesList {
	var expenseList EstimateCategoriesList
	for _, estimate := range el {
		if estimate.Amount < 0 {
			expenseList = append(expenseList, estimate)
		}
	}
	return expenseList
}

func (el EstimateCategoriesList) GetIncomeEstimates() EstimateCategoriesList {
	var expenseList EstimateCategoriesList
	for _, estimate := range el {
		if estimate.Amount > 0 {
			expenseList = append(expenseList, estimate)
		}
	}
	return expenseList
}

func (pt *Transaction) Consolidate() {
	if *pt.Estimate.ID == uuid.Nil {
		return
	}

	var realized float64
	for _, transaction := range pt.DoneList {
		realized += transaction.Amount
	}
	pt.Consolidation.Estimated = pt.Estimate.Amount
	pt.Consolidation.Realized = realized
	pt.Consolidation.Remaining = pt.Estimate.Amount - realized
}

func (b *Balance) Consolidate() {
	b.PeriodBalance = b.Income + b.Expense
}

func ToRecurrentMovement(movement Movement) RecurrentMovement {
	return RecurrentMovement{
		Description:   movement.Description,
		Amount:        movement.Amount,
		InitialDate:   movement.Date,
		UserID:        movement.UserID,
		CategoryID:    movement.CategoryID,
		SubCategoryID: movement.SubCategoryID,
		WalletID:      movement.WalletID,
		TypePaymentID: movement.TypePaymentID,
	}
}
