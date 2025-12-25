package domain

import (
	"time"

	"github.com/google/uuid"
)

type RecurrentMovement struct {
	ID            *uuid.UUID  `json:"id,omitempty" gorm:"primaryKey"`
	Description   string      `json:"description,omitempty"`
	Amount        float64     `json:"amount"`
	InitialDate   *time.Time  `json:"initial_date"`
	EndDate       *time.Time  `json:"end_date"`
	UserID        string      `json:"user_id"`
	CategoryID    *uuid.UUID  `json:"category_id,omitempty"`
	Category      Category    `json:"categories,omitempty"`
	SubCategoryID *uuid.UUID  `json:"sub_category_id,omitempty"`
	SubCategory   SubCategory `json:"sub_categories,omitempty"`
	WalletID      *uuid.UUID  `json:"wallet_id,omitempty"`
	Wallet        Wallet      `json:"wallets,omitempty"`
	TypePayment   TypePayment `json:"type_payment,omitempty"`
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
		TypePayment:   movement.TypePayment,
	}
}

func FromRecurrentMovement(recurrent RecurrentMovement, date time.Time) Movement {
	monthDate := SetMonthYear(*recurrent.InitialDate, date.Month(), date.Year())

	return Movement{
		Description:   recurrent.Description,
		Amount:        recurrent.Amount,
		Date:          &monthDate,
		UserID:        recurrent.UserID,
		IsRecurrent:   true,
		RecurrentID:   recurrent.ID,
		CategoryID:    recurrent.CategoryID,
		Category:      recurrent.Category,
		SubCategoryID: recurrent.SubCategoryID,
		SubCategory:   recurrent.SubCategory,
		WalletID:      recurrent.WalletID,
		Wallet:        recurrent.Wallet,
		TypePayment:   recurrent.TypePayment,
	}
}

func SetMonthYear(date time.Time, month time.Month, year int) time.Time {
	if month > 12 {
		month = month - 12
		year++
	}

	return time.Date(
		year,
		month,
		date.Day(),
		date.Hour(),
		date.Minute(),
		date.Second(),
		date.Nanosecond(),
		date.Location(),
	)
}

func SetMonthYearClamped(date time.Time, month time.Month, year int) time.Time {
	if month < 1 {
		month = month + 12
		year--
	}
	if month > 12 {
		month = month - 12
		year++
	}

	lastDayOfMonth := lastDay(year, month)
	day := date.Day()
	if day > lastDayOfMonth {
		day = lastDayOfMonth
	}

	return time.Date(
		year,
		month,
		day,
		date.Hour(),
		date.Minute(),
		date.Second(),
		date.Nanosecond(),
		date.Location(),
	)
}

func lastDay(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}
