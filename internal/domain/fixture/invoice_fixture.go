package fixture

import (
	"time"

	"personal-finance/internal/domain"

	"github.com/google/uuid"
)

var (
	InvoiceID = uuid.MustParse("88888888-8888-8888-8888-888888888888")
)

type InvoiceMockOption func(i *domain.Invoice)

func InvoiceMock(options ...InvoiceMockOption) domain.Invoice {
	i := domain.Invoice{
		ID:           &InvoiceID,
		CreditCardID: &CreditCardID,
		PeriodStart:  time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:    time.Date(2023, 10, 31, 0, 0, 0, 0, time.UTC),
		DueDate:      time.Date(2023, 10, 22, 0, 0, 0, 0, time.UTC),
		Amount:       1500.0,
		IsPaid:       false,
		UserID:       "user-test-id",
		DateCreate:   now,
		DateUpdate:   now,
	}

	for _, option := range options {
		option(&i)
	}

	return i
}

func WithInvoiceCreditCardID(creditCardID uuid.UUID) InvoiceMockOption {
	return func(i *domain.Invoice) {
		i.CreditCardID = &creditCardID
	}
}

func WithInvoicePeriod(start, end time.Time) InvoiceMockOption {
	return func(i *domain.Invoice) {
		i.PeriodStart = start
		i.PeriodEnd = end
	}
}

func WithInvoiceDueDate(dueDate time.Time) InvoiceMockOption {
	return func(i *domain.Invoice) {
		i.DueDate = dueDate
	}
}

func WithInvoiceAmount(amount float64) InvoiceMockOption {
	return func(i *domain.Invoice) {
		i.Amount = amount
	}
}

func WithInvoiceIsPaid(isPaid bool) InvoiceMockOption {
	return func(i *domain.Invoice) {
		i.IsPaid = isPaid
	}
}

func WithInvoicePayment(paymentDate time.Time, walletID uuid.UUID) InvoiceMockOption {
	return func(i *domain.Invoice) {
		i.PaymentDate = &paymentDate
		i.WalletID = &walletID
		i.IsPaid = true
	}
}

func WithInvoiceUserID(userID string) InvoiceMockOption {
	return func(i *domain.Invoice) {
		i.UserID = userID
	}
}

type DetailedInvoiceMockOption func(di *domain.DetailedInvoice)

func DetailedInvoiceMock(options ...DetailedInvoiceMockOption) domain.DetailedInvoice {
	di := domain.DetailedInvoice{
		Invoice: InvoiceMock(),
		Movements: []domain.Movement{
			MovementMock(),
			MovementMock(WithMovementAmount(-200.0)),
		},
	}

	for _, option := range options {
		option(&di)
	}

	return di
}

func WithDetailedInvoiceMovements(movements []domain.Movement) DetailedInvoiceMockOption {
	return func(di *domain.DetailedInvoice) {
		di.Movements = movements
	}
}

func WithDetailedInvoiceInvoice(invoice domain.Invoice) DetailedInvoiceMockOption {
	return func(di *domain.DetailedInvoice) {
		di.Invoice = invoice
	}
}
