package fixture

import (
	"math"
	"time"

	"personal-finance/internal/domain"

	"github.com/google/uuid"
)

var (
	now = time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC)

	MovementID    = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	CategoryID    = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	WalletID      = uuid.MustParse("33333333-3333-3333-3333-333333333333")
	SubCategoryID = uuid.MustParse("44444444-4444-4444-4444-444444444444")
	RecurrentID   = uuid.MustParse("55555555-5555-5555-5555-555555555555")
)

type MovementMockOption func(m *domain.Movement)

func MovementMock(options ...MovementMockOption) domain.Movement {
	m := domain.Movement{
		ID:            &MovementID,
		Description:   "Movimento de teste",
		Amount:        -100.0,
		UserID:        "user-test-id",
		IsPaid:        true,
		IsRecurrent:   false,
		RecurrentID:   nil,
		WalletID:      &WalletID,
		TypePayment:   domain.TypePaymentDebit,
		CategoryID:    &CategoryID,
		SubCategoryID: nil,
		DateCreate:    now,
		DateUpdate:    now,
	}

	for _, opt := range options {
		opt(&m)
	}

	return m
}

func WithMovementID(id uuid.UUID) MovementMockOption {
	return func(m *domain.Movement) {
		m.ID = &id
	}
}

func WithMovementDescription(description string) MovementMockOption {
	return func(m *domain.Movement) {
		m.Description = description
	}
}

func WithMovementAmount(amount float64) MovementMockOption {
	return func(m *domain.Movement) {
		m.Amount = amount
	}
}

func WithMovementDate(date time.Time) MovementMockOption {
	return func(m *domain.Movement) {
		m.Date = &date
	}
}

func WithMovementUserID(userID string) MovementMockOption {
	return func(m *domain.Movement) {
		m.UserID = userID
	}
}

func WithMovementIsPaid(isPaid bool) MovementMockOption {
	return func(m *domain.Movement) {
		m.IsPaid = isPaid
	}
}

func WithMovementIsRecurrent(isRecurrent bool) MovementMockOption {
	return func(m *domain.Movement) {
		m.IsRecurrent = isRecurrent
	}
}

func WithMovementRecurrentID() MovementMockOption {
	return func(m *domain.Movement) {
		m.RecurrentID = &RecurrentID
	}
}

func WithMovementWalletID(walletID uuid.UUID) MovementMockOption {
	return func(m *domain.Movement) {
		m.WalletID = &walletID
	}
}

func WithoutMovementWallet() MovementMockOption {
	return func(m *domain.Movement) {
		m.WalletID = nil
	}
}

func WithMovementTypePayment(typePayment string) MovementMockOption {
	return func(m *domain.Movement) {
		m.TypePayment = domain.TypePayment(typePayment)
	}
}

func WithMovementCategoryID(categoryID uuid.UUID) MovementMockOption {
	return func(m *domain.Movement) {
		m.CategoryID = &categoryID
	}
}

func WithMovementSubCategoryID(subCategoryID uuid.UUID) MovementMockOption {
	return func(m *domain.Movement) {
		m.SubCategoryID = &subCategoryID
	}
}

func WithMovementDateCreate(dateCreate time.Time) MovementMockOption {
	return func(m *domain.Movement) {
		m.DateCreate = dateCreate
	}
}

func WithMovementDateUpdate(dateUpdate time.Time) MovementMockOption {
	return func(m *domain.Movement) {
		m.DateUpdate = dateUpdate
	}
}

func AsMovementExpense(amount float64) MovementMockOption {
	return func(m *domain.Movement) {
		m.Amount = -math.Abs(amount)
	}
}

func AsMovementIncome(amount float64) MovementMockOption {
	return func(m *domain.Movement) {
		m.Amount = math.Abs(amount)
	}
}

func AsMovementRecurrent() MovementMockOption {
	return func(m *domain.Movement) {
		m.IsRecurrent = true
	}
}

func AsMovementUnpaid() MovementMockOption {
	return func(m *domain.Movement) {
		m.IsPaid = false
	}
}
