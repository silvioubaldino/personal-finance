package repository

import (
	"time"

	"personal-finance/internal/domain"

	"github.com/google/uuid"
)

type MovementDB struct {
	ID            *uuid.UUID `gorm:"primaryKey"`
	Description   string     `gorm:"description"`
	Amount        float64    `gorm:"amount"`
	Date          *time.Time `gorm:"date"`
	UserID        string     `gorm:"user_id"`
	IsPaid        bool       `gorm:"is_paid"`
	IsRecurrent   bool       `gorm:"is_recurrent"`
	RecurrentID   *uuid.UUID `gorm:"recurrent_id"`
	WalletID      *uuid.UUID `gorm:"wallet_id"`
	TypePaymentID int        `gorm:"type_payment_id"`
	CategoryID    *uuid.UUID `gorm:"category_id"`
	SubCategoryID *uuid.UUID `gorm:"sub_category_id"`
	DateCreate    time.Time  `gorm:"date_create"`
	DateUpdate    time.Time  `gorm:"date_update"`
}

func (MovementDB) TableName() string {
	return "movements"
}

func (m MovementDB) ToDomain() domain.Movement {
	return domain.Movement{
		ID:            m.ID,
		Description:   m.Description,
		Amount:        m.Amount,
		Date:          m.Date,
		UserID:        m.UserID,
		IsPaid:        m.IsPaid,
		IsRecurrent:   m.IsRecurrent,
		RecurrentID:   m.RecurrentID,
		WalletID:      m.WalletID,
		TypePaymentID: m.TypePaymentID,
		CategoryID:    m.CategoryID,
		SubCategoryID: m.SubCategoryID,
		DateCreate:    m.DateCreate,
		DateUpdate:    m.DateUpdate,
	}
}

func ToMovementModel(d domain.Movement) MovementDB {
	return MovementDB{
		ID:            d.ID,
		Description:   d.Description,
		Amount:        d.Amount,
		Date:          d.Date,
		UserID:        d.UserID,
		IsPaid:        d.IsPaid,
		IsRecurrent:   d.IsRecurrent,
		RecurrentID:   d.RecurrentID,
		WalletID:      d.WalletID,
		TypePaymentID: d.TypePaymentID,
		CategoryID:    d.CategoryID,
		SubCategoryID: d.SubCategoryID,
		DateCreate:    d.DateCreate,
		DateUpdate:    d.DateUpdate,
	}
}

type RecurrentMovementDB struct {
	ID            *uuid.UUID `gorm:"primaryKey"`
	Description   string     `gorm:"description"`
	Amount        float64    `gorm:"amount"`
	InitialDate   *time.Time `gorm:"initial_date"`
	EndDate       *time.Time `gorm:"end_date"`
	UserID        string     `gorm:"user_id"`
	WalletID      *uuid.UUID `gorm:"wallet_id"`
	TypePaymentID int        `gorm:"type_payment_id"`
	CategoryID    *uuid.UUID `gorm:"category_id"`
	SubCategoryID *uuid.UUID `gorm:"sub_category_id"`
}

func (RecurrentMovementDB) TableName() string {
	return "recurrent_movements"
}

func (r RecurrentMovementDB) ToDomain() domain.RecurrentMovement {
	return domain.RecurrentMovement{
		ID:            r.ID,
		Description:   r.Description,
		Amount:        r.Amount,
		InitialDate:   r.InitialDate,
		EndDate:       r.EndDate,
		UserID:        r.UserID,
		WalletID:      r.WalletID,
		TypePaymentID: r.TypePaymentID,
		CategoryID:    r.CategoryID,
		SubCategoryID: r.SubCategoryID,
	}
}

func ToRecurrentMovementModel(d domain.RecurrentMovement) RecurrentMovementDB {
	return RecurrentMovementDB{
		ID:            d.ID,
		Description:   d.Description,
		Amount:        d.Amount,
		InitialDate:   d.InitialDate,
		EndDate:       d.EndDate,
		UserID:        d.UserID,
		WalletID:      d.WalletID,
		TypePaymentID: d.TypePaymentID,
		CategoryID:    d.CategoryID,
		SubCategoryID: d.SubCategoryID,
	}
}
