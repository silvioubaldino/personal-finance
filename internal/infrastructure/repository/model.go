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
	IdempotencyHash    *string       `gorm:"idempotency_hash"`
	DateCreate         time.Time     `gorm:"date_create"`
	DateUpdate         time.Time     `gorm:"date_update"`
}

func (MovementDB) TableName() string {
	return "movements"
}

func (m MovementDB) ToDomain() domain.Movement {
	movement := domain.Movement{
		ID:              m.ID,
		Description:     m.Description,
		Amount:          m.Amount,
		Date:            m.Date,
		UserID:          m.UserID,
		IsPaid:          m.IsPaid,
		IsRecurrent:     m.RecurrentID != nil,
		RecurrentID:     m.RecurrentID,
		PairID:          m.PairID,
		WalletID:        m.WalletID,
		Wallet:          m.Wallet.ToDomain(),
		TypePayment:     domain.TypePayment(m.TypePayment),
		CategoryID:      m.CategoryID,
		Category:        m.Category.ToDomain(),
		SubCategoryID:   m.SubCategoryID,
		SubCategory:     m.SubCategory.ToDomain(),
		IdempotencyHash: m.IdempotencyHash,
		DateCreate:      m.DateCreate,
		DateUpdate:      m.DateUpdate,
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
		ID:              d.ID,
		Description:     d.Description,
		Amount:          d.Amount,
		Date:            d.Date,
		UserID:          d.UserID,
		IsPaid:          d.IsPaid,
		RecurrentID:     d.RecurrentID,
		PairID:          d.PairID,
		WalletID:        d.WalletID,
		TypePayment:     string(d.TypePayment),
		CategoryID:      d.CategoryID,
		SubCategoryID:   d.SubCategoryID,
		IdempotencyHash: d.IdempotencyHash,
		DateCreate:      d.DateCreate,
		DateUpdate:      d.DateUpdate,
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
	ID            *uuid.UUID      `gorm:"primaryKey"`
	Description   string          `gorm:"description,omitempty"`
	UserID        string          `gorm:"user_id"`
	Color         string          `gorm:"color"`
	IsIncome      bool            `gorm:"is_income"`
	SubCategories []SubCategoryDB `gorm:"foreignKey:CategoryID"`
	DateCreate    time.Time       `gorm:"date_create"`
	DateUpdate    time.Time       `gorm:"date_update"`
}

func (CategoryDB) TableName() string {
	return "categories"
}

func (c CategoryDB) ToDomain() domain.Category {
	return domain.Category{
		ID:          c.ID,
		Description: c.Description,
		UserID:      c.UserID,
		Color:       c.Color,
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
		Color:       d.Color,
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
	Color           string
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
		Color:           c.Color,
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
		Color:           creditCard.Color,
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

type UserDB struct {
	ID        string    `gorm:"primaryKey;column:id"`
	Language  string    `gorm:"column:language"`
	Currency  string    `gorm:"column:currency"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (UserDB) TableName() string {
	return "users"
}

func (u UserDB) ToDomain() domain.User {
	return domain.User{
		ID:        u.ID,
		Language:  u.Language,
		Currency:  u.Currency,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

func FromUserDomain(d domain.User) UserDB {
	return UserDB{
		ID:        d.ID,
		Language:  d.Language,
		Currency:  d.Currency,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}

type UserConsentDB struct {
	ID          *uuid.UUID `gorm:"primaryKey"`
	UserID      string     `gorm:"user_id"`
	TermVersion string     `gorm:"term_version"`
	AgreedAt    time.Time  `gorm:"agreed_at"`
	IPAddress   string     `gorm:"ip_address"`
	UserAgent   string     `gorm:"user_agent"`
}

func (UserConsentDB) TableName() string {
	return "user_consents"
}

func (u UserConsentDB) ToDomain() domain.UserConsent {
	id := uuid.Nil
	if u.ID != nil {
		id = *u.ID
	}
	return domain.UserConsent{
		ID:          id,
		UserID:      u.UserID,
		TermVersion: u.TermVersion,
		AgreedAt:    u.AgreedAt,
		IPAddress:   u.IPAddress,
		UserAgent:   u.UserAgent,
	}
}

func FromUserConsentDomain(d domain.UserConsent) UserConsentDB {
	return UserConsentDB{
		ID:          &d.ID,
		UserID:      d.UserID,
		TermVersion: d.TermVersion,
		AgreedAt:    d.AgreedAt,
		IPAddress:   d.IPAddress,
		UserAgent:   d.UserAgent,
	}
}

type UserDeviceDB struct {
	ID            uuid.UUID  `gorm:"primaryKey"`
	UserID        string     `gorm:"user_id"`
	ExpoPushToken string     `gorm:"expo_push_token"`
	Platform      string     `gorm:"platform"`
	DateCreate    time.Time  `gorm:"date_create"`
	DateUpdate    time.Time  `gorm:"date_update"`
	LastSeenAt    *time.Time `gorm:"last_seen_at"`
}

func (UserDeviceDB) TableName() string {
	return "user_devices"
}

func (d UserDeviceDB) ToDomain() domain.Device {
	return domain.Device{
		ID:            d.ID,
		UserID:        d.UserID,
		ExpoPushToken: d.ExpoPushToken,
		Platform:      domain.Platform(d.Platform),
		DateCreate:    d.DateCreate,
		DateUpdate:    d.DateUpdate,
		LastSeenAt:    d.LastSeenAt,
	}
}

func FromDeviceDomain(d domain.Device) UserDeviceDB {
	return UserDeviceDB{
		ID:            d.ID,
		UserID:        d.UserID,
		ExpoPushToken: d.ExpoPushToken,
		Platform:      string(d.Platform),
		DateCreate:    d.DateCreate,
		DateUpdate:    d.DateUpdate,
		LastSeenAt:    d.LastSeenAt,
	}
}

type SubscriptionPlanDB struct {
	ID                  string `gorm:"primaryKey"`
	Name                string
	Price               float64
	Currency            string
	Frequency           int
	FrequencyType       string
	IsActive            bool
	IsPublic            bool    `gorm:"column:is_public"`
	MPPreapprovalPlanID *string `gorm:"column:mp_preapproval_plan_id"`
	AppleProductID      *string `gorm:"column:apple_product_id"`
	GoogleProductID     *string `gorm:"column:google_product_id"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func (SubscriptionPlanDB) TableName() string {
	return "subscription_plans"
}

func (p SubscriptionPlanDB) ToDomain() domain.SubscriptionPlan {
	mpPlanID := ""
	if p.MPPreapprovalPlanID != nil {
		mpPlanID = *p.MPPreapprovalPlanID
	}
	return domain.SubscriptionPlan{
		ID:                  p.ID,
		Name:                p.Name,
		Price:               p.Price,
		Currency:            p.Currency,
		Frequency:           p.Frequency,
		FrequencyType:       p.FrequencyType,
		IsActive:            p.IsActive,
		IsPublic:            p.IsPublic,
		MPPreapprovalPlanID: mpPlanID,
	}
}

type SubscriptionDB struct {
	ID                uuid.UUID  `gorm:"primaryKey;column:id"`
	UserID            string     `gorm:"column:user_id"`
	Source            string     `gorm:"column:source;uniqueIndex:idx_subscriptions_source_external_id"`
	ExternalID        string     `gorm:"column:external_id;uniqueIndex:idx_subscriptions_source_external_id"`
	ExternalProductID string     `gorm:"column:external_product_id"`
	PlanID            *string    `gorm:"column:plan_id"`
	Status            string     `gorm:"column:status"`
	CurrentPrice      float64    `gorm:"column:current_price"`
	Currency          string     `gorm:"column:currency"`
	StartedAt         time.Time  `gorm:"column:started_at"`
	CurrentPeriodEnd  *time.Time `gorm:"column:current_period_end"`
	CancelledAt       *time.Time `gorm:"column:cancelled_at"`
	CreatedAt         time.Time  `gorm:"column:created_at"`
	UpdatedAt         time.Time  `gorm:"column:updated_at"`
}

func (SubscriptionDB) TableName() string {
	return "subscriptions"
}

func (s SubscriptionDB) ToDomain() domain.Subscription {
	planID := ""
	if s.PlanID != nil {
		planID = *s.PlanID
	}
	return domain.Subscription{
		ID:                s.ID,
		UserID:            s.UserID,
		Source:            domain.SubscriptionSource(s.Source),
		ExternalID:        s.ExternalID,
		ExternalProductID: s.ExternalProductID,
		PlanID:            planID,
		Status:            domain.SubscriptionStatus(s.Status),
		CurrentPrice:      s.CurrentPrice,
		Currency:          s.Currency,
		StartedAt:         s.StartedAt,
		CurrentPeriodEnd:  s.CurrentPeriodEnd,
		CancelledAt:       s.CancelledAt,
		CreatedAt:         s.CreatedAt,
		UpdatedAt:         s.UpdatedAt,
	}
}

type CouponDB struct {
	ID                string    `gorm:"primaryKey;column:id"`
	Code              string    `gorm:"column:code;uniqueIndex:idx_coupons_code"`
	Description       string    `gorm:"column:description"`
	DiscountType      string    `gorm:"column:discount_type"`
	DiscountValue     float64   `gorm:"column:discount_value"`
	ValidFrom         time.Time `gorm:"column:valid_from"`
	ValidUntil        time.Time `gorm:"column:valid_until"`
	MaxRedemptions    *int      `gorm:"column:max_redemptions"`
	RedemptionCount   int       `gorm:"column:redemption_count"`
	ApplicablePlanIDs string    `gorm:"column:applicable_plan_ids"`
	TargetPlanID      *string   `gorm:"column:target_plan_id"`
	IsActive          bool      `gorm:"column:is_active"`
	CreatedAt         time.Time `gorm:"column:created_at"`
	UpdatedAt         time.Time `gorm:"column:updated_at"`
}

func (CouponDB) TableName() string {
	return "coupons"
}

func (c CouponDB) ToDomain() domain.Coupon {
	targetPlanID := ""
	if c.TargetPlanID != nil {
		targetPlanID = *c.TargetPlanID
	}
	return domain.Coupon{
		ID:                c.ID,
		Code:              c.Code,
		Description:       c.Description,
		DiscountType:      domain.CouponDiscountType(c.DiscountType),
		DiscountValue:     c.DiscountValue,
		ValidFrom:         c.ValidFrom,
		ValidUntil:        c.ValidUntil,
		MaxRedemptions:    c.MaxRedemptions,
		RedemptionCount:   c.RedemptionCount,
		ApplicablePlanIDs: decodeStringSlice(c.ApplicablePlanIDs),
		TargetPlanID:      targetPlanID,
		IsActive:          c.IsActive,
		CreatedAt:         c.CreatedAt,
		UpdatedAt:         c.UpdatedAt,
	}
}

func FromCouponDomain(d domain.Coupon) CouponDB {
	var targetPlanID *string
	if d.TargetPlanID != "" {
		tp := d.TargetPlanID
		targetPlanID = &tp
	}
	return CouponDB{
		ID:                d.ID,
		Code:              d.Code,
		Description:       d.Description,
		DiscountType:      string(d.DiscountType),
		DiscountValue:     d.DiscountValue,
		ValidFrom:         d.ValidFrom,
		ValidUntil:        d.ValidUntil,
		MaxRedemptions:    d.MaxRedemptions,
		RedemptionCount:   d.RedemptionCount,
		ApplicablePlanIDs: encodeStringSlice(d.ApplicablePlanIDs),
		TargetPlanID:      targetPlanID,
		IsActive:          d.IsActive,
		CreatedAt:         d.CreatedAt,
		UpdatedAt:         d.UpdatedAt,
	}
}

type CouponRedemptionDB struct {
	ID             uuid.UUID  `gorm:"primaryKey;column:id"`
	UserID         string     `gorm:"column:user_id;uniqueIndex:idx_coupon_redemptions_user_coupon"`
	CouponID       string     `gorm:"column:coupon_id;uniqueIndex:idx_coupon_redemptions_user_coupon"`
	PlanID         string     `gorm:"column:plan_id"`
	SubscriptionID *uuid.UUID `gorm:"column:subscription_id"`
	OriginalPrice  float64    `gorm:"column:original_price"`
	LockedPrice    float64    `gorm:"column:locked_price"`
	Status         string     `gorm:"column:status"`
	RedeemedAt     time.Time  `gorm:"column:redeemed_at"`
	CancelledAt    *time.Time `gorm:"column:cancelled_at"`
}

func (CouponRedemptionDB) TableName() string {
	return "coupon_redemptions"
}

func (r CouponRedemptionDB) ToDomain() domain.CouponRedemption {
	return domain.CouponRedemption{
		ID:             r.ID,
		UserID:         r.UserID,
		CouponID:       r.CouponID,
		PlanID:         r.PlanID,
		SubscriptionID: r.SubscriptionID,
		OriginalPrice:  r.OriginalPrice,
		LockedPrice:    r.LockedPrice,
		Status:         domain.CouponRedemptionStatus(r.Status),
		RedeemedAt:     r.RedeemedAt,
		CancelledAt:    r.CancelledAt,
	}
}

func FromCouponRedemptionDomain(d domain.CouponRedemption) CouponRedemptionDB {
	return CouponRedemptionDB{
		ID:             d.ID,
		UserID:         d.UserID,
		CouponID:       d.CouponID,
		PlanID:         d.PlanID,
		SubscriptionID: d.SubscriptionID,
		OriginalPrice:  d.OriginalPrice,
		LockedPrice:    d.LockedPrice,
		Status:         string(d.Status),
		RedeemedAt:     d.RedeemedAt,
		CancelledAt:    d.CancelledAt,
	}
}

func FromSubscriptionDomain(d domain.Subscription) SubscriptionDB {
	var planID *string
	if d.PlanID != "" {
		planID = &d.PlanID
	}
	return SubscriptionDB{
		ID:                d.ID,
		UserID:            d.UserID,
		Source:            string(d.Source),
		ExternalID:        d.ExternalID,
		ExternalProductID: d.ExternalProductID,
		PlanID:            planID,
		Status:            string(d.Status),
		CurrentPrice:      d.CurrentPrice,
		Currency:          d.Currency,
		StartedAt:         d.StartedAt,
		CurrentPeriodEnd:  d.CurrentPeriodEnd,
		CancelledAt:       d.CancelledAt,
		CreatedAt:         d.CreatedAt,
		UpdatedAt:         d.UpdatedAt,
	}
}
