package fixture

import (
	"personal-finance/internal/domain"

	"github.com/google/uuid"
)

var (
	CreditCardID    = uuid.MustParse("66666666-6666-6666-6666-666666666666")
	DefaultWalletID = uuid.MustParse("77777777-7777-7777-7777-777777777777")
)

type CreditCardMockOption func(c *domain.CreditCard)

func CreditCardMock(options ...CreditCardMockOption) domain.CreditCard {
	c := domain.CreditCard{
		ID:              &CreditCardID,
		Name:            "Cartão de Teste",
		CreditLimit:     5000.0,
		ClosingDay:      15,
		DueDay:          22,
		DefaultWalletID: &DefaultWalletID,
		UserID:          "user-test-id",
		DateCreate:      now,
		DateUpdate:      now,
	}

	for _, option := range options {
		option(&c)
	}

	return c
}

func WithCreditCardName(name string) CreditCardMockOption {
	return func(c *domain.CreditCard) {
		c.Name = name
	}
}

func WithCreditCardLimit(limit float64) CreditCardMockOption {
	return func(c *domain.CreditCard) {
		c.CreditLimit = limit
	}
}

func WithCreditCardClosingDay(day int) CreditCardMockOption {
	return func(c *domain.CreditCard) {
		c.ClosingDay = day
	}
}

func WithCreditCardDueDay(day int) CreditCardMockOption {
	return func(c *domain.CreditCard) {
		c.DueDay = day
	}
}

func WithCreditCardUserID(userID string) CreditCardMockOption {
	return func(c *domain.CreditCard) {
		c.UserID = userID
	}
}

func WithCreditCardDefaultWalletID(walletID uuid.UUID) CreditCardMockOption {
	return func(c *domain.CreditCard) {
		c.DefaultWalletID = &walletID
	}
}

func CreditCardWithOpenInvoicesMock(options ...CreditCardMockOption) domain.CreditCardWithOpenInvoices {
	return domain.CreditCardWithOpenInvoices{
		CreditCard: CreditCardMock(options...),
		OpenInvoices: []domain.Invoice{
			InvoiceMock(),
			InvoiceMock(WithInvoiceAmount(800.0)),
		},
	}
}

func WithOpenInvoices(invoices []domain.Invoice) func(*domain.CreditCardWithOpenInvoices) {
	return func(c *domain.CreditCardWithOpenInvoices) {
		c.OpenInvoices = invoices
	}
}
