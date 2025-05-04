package repository

import (
	"time"

	"personal-finance/internal/domain"

	"github.com/google/uuid"
)

type MovementDB struct {
	ID            *uuid.UUID    `gorm:"primaryKey"`
	Description   string        `gorm:"description"`
	Amount        float64       `gorm:"amount"`
	Date          *time.Time    `gorm:"date"`
	UserID        string        `gorm:"user_id"`
	IsPaid        bool          `gorm:"is_paid"`
	RecurrentID   *uuid.UUID    `gorm:"recurrent_id"`
	WalletID      *uuid.UUID    `gorm:"wallet_id"`
	Wallet        WalletDB      `gorm:"wallets"`
	TypePayment   string        `gorm:"type_payment"`
	CategoryID    *uuid.UUID    `gorm:"category_id"`
	Category      CategoryDB    `gorm:"categories"`
	SubCategoryID *uuid.UUID    `gorm:"sub_category_id"`
	SubCategory   SubCategoryDB `gorm:"sub_categories"`
	DateCreate    time.Time     `gorm:"date_create"`
	DateUpdate    time.Time     `gorm:"date_update"`
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
		IsRecurrent:   m.RecurrentID != nil,
		RecurrentID:   m.RecurrentID,
		WalletID:      m.WalletID,
		Wallet:        m.Wallet.ToDomain(),
		TypePayment:   domain.TypePayment(m.TypePayment),
		CategoryID:    m.CategoryID,
		Category:      m.Category.ToDomain(),
		SubCategoryID: m.SubCategoryID,
		SubCategory:   m.SubCategory.ToDomain(),
		DateCreate:    m.DateCreate,
		DateUpdate:    m.DateUpdate,
	}
}

func FromMovementDomain(d domain.Movement) MovementDB {
	return MovementDB{
		ID:            d.ID,
		Description:   d.Description,
		Amount:        d.Amount,
		Date:          d.Date,
		UserID:        d.UserID,
		IsPaid:        d.IsPaid,
		RecurrentID:   d.RecurrentID,
		WalletID:      d.WalletID,
		TypePayment:   string(d.TypePayment),
		CategoryID:    d.CategoryID,
		SubCategoryID: d.SubCategoryID,
		DateCreate:    d.DateCreate,
		DateUpdate:    d.DateUpdate,
	}
}

type CategoryDB struct {
	ID          *uuid.UUID `gorm:"primaryKey"`
	Description string     `gorm:"description,omitempty"`
	UserID      string     `gorm:"user_id"`
	IsIncome    bool       `gorm:"is_income"`
	DateCreate  time.Time  `gorm:"date_create"`
	DateUpdate  time.Time  `gorm:"date_update"`
}

func (CategoryDB) TableName() string {
	return "categories"
}

func (c CategoryDB) ToDomain() domain.Category {
	return domain.Category{
		ID:          c.ID,
		Description: c.Description,
		UserID:      c.UserID,
		IsIncome:    c.IsIncome,
		DateCreate:  c.DateCreate,
		DateUpdate:  c.DateUpdate,
	}
}

func FromCategoryDomain(d domain.Category) CategoryDB {
	return CategoryDB{
		ID:          d.ID,
		Description: d.Description,
		UserID:      d.UserID,
		IsIncome:    d.IsIncome,
		DateCreate:  d.DateCreate,
		DateUpdate:  d.DateUpdate,
	}
}

type SubCategoryDB struct {
	ID          *uuid.UUID `gorm:"primaryKey"`
	Description string     `gorm:"description"`
	UserID      string     `gorm:"user_id"`
	CategoryID  *uuid.UUID `gorm:"category_id"`
	DateCreate  time.Time  `gorm:"date_create"`
	DateUpdate  time.Time  `gorm:"date_update"`
}

func (SubCategoryDB) TableName() string {
	return "sub_categories"
}

func (s SubCategoryDB) ToDomain() domain.SubCategory {
	return domain.SubCategory{
		ID:          s.ID,
		Description: s.Description,
		UserID:      s.UserID,
		CategoryID:  s.CategoryID,
		DateCreate:  s.DateCreate,
		DateUpdate:  s.DateUpdate,
	}
}

func FromSubCategoryDomain(d domain.SubCategory) SubCategoryDB {
	return SubCategoryDB{
		ID:          d.ID,
		Description: d.Description,
		UserID:      d.UserID,
		CategoryID:  d.CategoryID,
		DateCreate:  d.DateCreate,
		DateUpdate:  d.DateUpdate,
	}
}

type RecurrentMovementDB struct {
	ID            *uuid.UUID    `gorm:"primaryKey"`
	Description   string        `gorm:"description"`
	Amount        float64       `gorm:"amount"`
	InitialDate   *time.Time    `gorm:"initial_date"`
	EndDate       *time.Time    `gorm:"end_date"`
	UserID        string        `gorm:"user_id"`
	WalletID      *uuid.UUID    `gorm:"wallet_id"`
	Wallet        WalletDB      `gorm:"wallets"`
	CategoryID    *uuid.UUID    `gorm:"category_id"`
	Category      CategoryDB    `gorm:"categories"`
	SubCategoryID *uuid.UUID    `gorm:"sub_category_id"`
	SubCategory   SubCategoryDB `gorm:"sub_categories"`
	TypePayment   string        `gorm:"type_payment"`
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
		TypePayment:   domain.TypePayment(r.TypePayment),
		CategoryID:    r.CategoryID,
		SubCategoryID: r.SubCategoryID,
	}
}

func FromRecurrentMovementDomain(d domain.RecurrentMovement) RecurrentMovementDB {
	return RecurrentMovementDB{
		ID:            d.ID,
		Description:   d.Description,
		Amount:        d.Amount,
		InitialDate:   d.InitialDate,
		EndDate:       d.EndDate,
		UserID:        d.UserID,
		WalletID:      d.WalletID,
		TypePayment:   string(d.TypePayment),
		CategoryID:    d.CategoryID,
		SubCategoryID: d.SubCategoryID,
	}
}

type WalletDB struct {
	ID             *uuid.UUID `gorm:"primaryKey"`
	Description    string     `gorm:"description"`
	Balance        float64    `gorm:"balance"`
	UserID         string     `gorm:"user_id"`
	InitialBalance float64    `gorm:"initial_balance"`
	InitialDate    time.Time  `gorm:"initial_date"`
	DateCreate     time.Time  `gorm:"date_create"`
	DateUpdate     time.Time  `gorm:"date_update"`
}

func (WalletDB) TableName() string {
	return "wallets"
}

func (w WalletDB) ToDomain() domain.Wallet {
	return domain.Wallet{
		ID:             w.ID,
		Description:    w.Description,
		Balance:        w.Balance,
		UserID:         w.UserID,
		InitialBalance: w.InitialBalance,
		InitialDate:    w.InitialDate,
		DateCreate:     w.DateCreate,
		DateUpdate:     w.DateUpdate,
	}
}

func FromWalletDomain(d domain.Wallet) WalletDB {
	return WalletDB{
		ID:             d.ID,
		Description:    d.Description,
		Balance:        d.Balance,
		UserID:         d.UserID,
		InitialBalance: d.InitialBalance,
		InitialDate:    d.InitialDate,
		DateCreate:     d.DateCreate,
		DateUpdate:     d.DateUpdate,
	}
}
