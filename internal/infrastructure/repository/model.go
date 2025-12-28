package repository

import (
	"time"

	"personal-finance/internal/domain"

	"github.com/google/uuid"
)

type MovementDB struct {
	ID                 *uuid.UUID    `gorm:"primaryKey"`
	Description        string        `gorm:"description"`
	Amount             float64       `gorm:"amount"`
	Date               *time.Time    `gorm:"date"`
	UserID             string        `gorm:"user_id"`
	IsPaid             bool          `gorm:"is_paid"`
	RecurrentID        *uuid.UUID    `gorm:"recurrent_id"`
	PairID             *uuid.UUID    `gorm:"pair_id"`
	InvoiceID          *uuid.UUID    `gorm:"invoice_id"`
	Invoice            InvoiceDB     `gorm:"foreignKey:InvoiceID"`
	InstallmentGroupID *uuid.UUID    `gorm:"installment_group_id"`
	InstallmentNumber  *int          `gorm:"installment_number"`
	TotalInstallments  *int          `gorm:"total_installments"`
	WalletID           *uuid.UUID    `gorm:"wallet_id"`
	Wallet             WalletDB      `gorm:"wallets"`
	TypePayment        string        `gorm:"type_payment"`
	CategoryID         *uuid.UUID    `gorm:"category_id"`
	Category           CategoryDB    `gorm:"categories"`
	SubCategoryID      *uuid.UUID    `gorm:"sub_category_id"`
	SubCategory        SubCategoryDB `gorm:"sub_categories"`
	DateCreate         time.Time     `gorm:"date_create"`
	DateUpdate         time.Time     `gorm:"date_update"`
}

func (MovementDB) TableName() string {
	return "movements"
}

func (m MovementDB) ToDomain() domain.Movement {
	movement := domain.Movement{
		ID:            m.ID,
		Description:   m.Description,
		Amount:        m.Amount,
		Date:          m.Date,
		UserID:        m.UserID,
		IsPaid:        m.IsPaid,
		IsRecurrent:   m.RecurrentID != nil,
		RecurrentID:   m.RecurrentID,
		PairID:        m.PairID,
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

	if m.InvoiceID != nil || m.InstallmentGroupID != nil {
		creditCardInfo := &domain.CreditCardMovement{
			InvoiceID:          m.InvoiceID,
			InstallmentGroupID: m.InstallmentGroupID,
			InstallmentNumber:  m.InstallmentNumber,
			TotalInstallments:  m.TotalInstallments,
		}

		if m.InvoiceID != nil && m.Invoice.ID != nil {
			creditCardInfo.CreditCardID = m.Invoice.CreditCardID
		}

		movement.CreditCardInfo = creditCardInfo
	}

	return movement
}

func FromMovementDomain(d domain.Movement) MovementDB {
	movementDB := MovementDB{
		ID:            d.ID,
		Description:   d.Description,
		Amount:        d.Amount,
		Date:          d.Date,
		UserID:        d.UserID,
		IsPaid:        d.IsPaid,
		RecurrentID:   d.RecurrentID,
		PairID:        d.PairID,
		WalletID:      d.WalletID,
		TypePayment:   string(d.TypePayment),
		CategoryID:    d.CategoryID,
		SubCategoryID: d.SubCategoryID,
		DateCreate:    d.DateCreate,
		DateUpdate:    d.DateUpdate,
	}

	if d.CreditCardInfo != nil {
		movementDB.InvoiceID = d.CreditCardInfo.InvoiceID
		movementDB.InstallmentGroupID = d.CreditCardInfo.InstallmentGroupID
		movementDB.InstallmentNumber = d.CreditCardInfo.InstallmentNumber
		movementDB.TotalInstallments = d.CreditCardInfo.TotalInstallments
	}

	return movementDB
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

type CreditCardDB struct {
	ID              *uuid.UUID `gorm:"primaryKey"`
	Name            string
	CreditLimit     float64
	ClosingDay      int
	DueDay          int
	DefaultWalletID *uuid.UUID
	DefaultWallet   WalletDB `gorm:"foreignKey:DefaultWalletID"`
	UserID          string
	DateCreate      time.Time
	DateUpdate      time.Time
}

func (CreditCardDB) TableName() string {
	return "credit_cards"
}

func (c CreditCardDB) ToDomain() domain.CreditCard {
	return domain.CreditCard{
		ID:              c.ID,
		Name:            c.Name,
		CreditLimit:     c.CreditLimit,
		ClosingDay:      c.ClosingDay,
		DueDay:          c.DueDay,
		DefaultWalletID: c.DefaultWalletID,
		DefaultWallet:   c.DefaultWallet.ToDomain(),
		UserID:          c.UserID,
		DateCreate:      c.DateCreate,
		DateUpdate:      c.DateUpdate,
	}
}

func FromCreditCardDomain(creditCard domain.CreditCard) CreditCardDB {
	return CreditCardDB{
		ID:              creditCard.ID,
		Name:            creditCard.Name,
		CreditLimit:     creditCard.CreditLimit,
		ClosingDay:      creditCard.ClosingDay,
		DueDay:          creditCard.DueDay,
		DefaultWalletID: creditCard.DefaultWalletID,
		UserID:          creditCard.UserID,
		DateCreate:      creditCard.DateCreate,
		DateUpdate:      creditCard.DateUpdate,
	}
}

type InvoiceDB struct {
	ID           *uuid.UUID `gorm:"primaryKey"`
	CreditCardID *uuid.UUID
	CreditCard   CreditCardDB `gorm:"foreignKey:CreditCardID"`
	PeriodStart  time.Time
	PeriodEnd    time.Time
	DueDate      time.Time
	PaymentDate  *time.Time
	Amount       float64
	IsPaid       bool
	WalletID     *uuid.UUID
	Wallet       WalletDB `gorm:"foreignKey:WalletID"`
	UserID       string
	DateCreate   time.Time
	DateUpdate   time.Time
}

func (InvoiceDB) TableName() string {
	return "invoices"
}

func (i InvoiceDB) ToDomain() domain.Invoice {
	return domain.Invoice{
		ID:           i.ID,
		CreditCardID: i.CreditCardID,
		CreditCard:   i.CreditCard.ToDomain(),
		PeriodStart:  i.PeriodStart,
		PeriodEnd:    i.PeriodEnd,
		DueDate:      i.DueDate,
		PaymentDate:  i.PaymentDate,
		Amount:       i.Amount,
		IsPaid:       i.IsPaid,
		WalletID:     i.WalletID,
		Wallet:       i.Wallet.ToDomain(),
		UserID:       i.UserID,
		DateCreate:   i.DateCreate,
		DateUpdate:   i.DateUpdate,
	}
}

func FromInvoiceDomain(invoice domain.Invoice) InvoiceDB {
	return InvoiceDB{
		ID:           invoice.ID,
		CreditCardID: invoice.CreditCardID,
		CreditCard:   FromCreditCardDomain(invoice.CreditCard),
		PeriodStart:  invoice.PeriodStart,
		PeriodEnd:    invoice.PeriodEnd,
		DueDate:      invoice.DueDate,
		PaymentDate:  invoice.PaymentDate,
		Amount:       invoice.Amount,
		IsPaid:       invoice.IsPaid,
		WalletID:     invoice.WalletID,
		UserID:       invoice.UserID,
		DateCreate:   invoice.DateCreate,
		DateUpdate:   invoice.DateUpdate,
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
		Wallet:        r.Wallet.ToDomain(),
		TypePayment:   domain.TypePayment(r.TypePayment),
		CategoryID:    r.CategoryID,
		Category:      r.Category.ToDomain(),
		SubCategoryID: r.SubCategoryID,
		SubCategory:   r.SubCategory.ToDomain(),
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

type UserPreferencesDB struct {
	UserID     string    `gorm:"primaryKey"`
	Language   string    `gorm:"language"`
	Currency   string    `gorm:"currency"`
	DateCreate time.Time `gorm:"date_create"`
	DateUpdate time.Time `gorm:"date_update"`
}

func (UserPreferencesDB) TableName() string {
	return "user_preferences"
}

func (u UserPreferencesDB) ToDomain() domain.UserPreferences {
	return domain.UserPreferences{
		UserID:     u.UserID,
		Language:   u.Language,
		Currency:   u.Currency,
		DateCreate: u.DateCreate,
		DateUpdate: u.DateUpdate,
	}
}

func FromUserPreferencesDomain(d domain.UserPreferences) UserPreferencesDB {
	return UserPreferencesDB{
		UserID:     d.UserID,
		Language:   d.Language,
		Currency:   d.Currency,
		DateCreate: d.DateCreate,
		DateUpdate: d.DateUpdate,
	}
}
