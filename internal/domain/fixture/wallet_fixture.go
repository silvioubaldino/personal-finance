package fixture

import (
	"time"

	"personal-finance/internal/domain"

	"github.com/google/uuid"
)

var (
	FixtureWalletID = uuid.MustParse("66666666-6666-6666-6666-666666666666")
	fixtureNow      = time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC)
)

type WalletMockOption func(w *domain.Wallet)

func WalletMock(options ...WalletMockOption) domain.Wallet {
	w := domain.Wallet{
		ID:             &FixtureWalletID,
		Description:    "Carteira de teste",
		Balance:        1000.0,
		UserID:         "user-test-id",
		InitialBalance: 1000.0,
		InitialDate:    fixtureNow,
		DateCreate:     fixtureNow,
		DateUpdate:     fixtureNow,
	}

	for _, opt := range options {
		opt(&w)
	}

	return w
}

func WithWalletID(id uuid.UUID) WalletMockOption {
	return func(w *domain.Wallet) {
		w.ID = &id
	}
}

func WithWalletDescription(description string) WalletMockOption {
	return func(w *domain.Wallet) {
		w.Description = description
	}
}

func WithWalletBalance(balance float64) WalletMockOption {
	return func(w *domain.Wallet) {
		w.Balance = balance
	}
}

func WithWalletUserID(userID string) WalletMockOption {
	return func(w *domain.Wallet) {
		w.UserID = userID
	}
}

func WithWalletInitialBalance(initialBalance float64) WalletMockOption {
	return func(w *domain.Wallet) {
		w.InitialBalance = initialBalance
	}
}

func WithWalletInitialDate(initialDate time.Time) WalletMockOption {
	return func(w *domain.Wallet) {
		w.InitialDate = initialDate
	}
}

func WithWalletDateCreate(dateCreate time.Time) WalletMockOption {
	return func(w *domain.Wallet) {
		w.DateCreate = dateCreate
	}
}

func WithWalletDateUpdate(dateUpdate time.Time) WalletMockOption {
	return func(w *domain.Wallet) {
		w.DateUpdate = dateUpdate
	}
}
