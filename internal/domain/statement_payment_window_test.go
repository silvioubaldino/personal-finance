package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func d(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func TestInvoicePaymentWindow_Classify(t *testing.T) {
	window := InvoicePaymentWindow{PreviousDueDate: d(2026, 1, 4), PeriodEnd: d(2026, 1, 29)}

	tests := map[string]struct {
		window   InvoicePaymentWindow
		date     time.Time
		expected string
	}{
		"payment before the previous due date is the previous invoice":   {window: window, date: d(2025, 12, 30), expected: ExclusionReasonInvoicePayment},
		"payment on the previous due date is the previous invoice":       {window: window, date: d(2026, 1, 4), expected: ExclusionReasonInvoicePayment},
		"payment the day after the previous due date is an advance":      {window: window, date: d(2026, 1, 5), expected: ExclusionReasonAdvancePayment},
		"payment in the middle of the cycle is an advance":               {window: window, date: d(2026, 1, 11), expected: ExclusionReasonAdvancePayment},
		"payment on the period end is an advance":                        {window: window, date: d(2026, 1, 29), expected: ExclusionReasonAdvancePayment},
		"payment after the period end is not an advance of this invoice": {window: window, date: d(2026, 1, 30), expected: ExclusionReasonInvoicePayment},
		"unknown window classifies everything as invoice_payment":        {window: InvoicePaymentWindow{}, date: d(2026, 1, 11), expected: ExclusionReasonInvoicePayment},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.window.Classify(tc.date))
		})
	}
}

func TestInvoicePaymentWindowFromCreditCard(t *testing.T) {
	tests := map[string]struct {
		card     CreditCard
		dates    []time.Time
		expected InvoicePaymentWindow
		ok       bool
	}{
		"closing 29 due 4 (Mercado Pago): previous due 04/01, period end 29/01": {
			card:     CreditCard{ClosingDay: 29, DueDay: 4},
			dates:    []time.Time{d(2025, 12, 31), d(2026, 1, 20)},
			expected: InvoicePaymentWindow{PreviousDueDate: d(2026, 1, 4), PeriodEnd: d(2026, 1, 29)},
			ok:       true,
		},
		"closing 3 due 10: previous due is the 10th of the closing month": {
			card:     CreditCard{ClosingDay: 3, DueDay: 10},
			dates:    []time.Time{d(2026, 5, 12), d(2026, 5, 20)},
			expected: InvoicePaymentWindow{PreviousDueDate: d(2026, 5, 10), PeriodEnd: d(2026, 6, 3)},
			ok:       true,
		},
		"no reference dates gives no window": {
			card:  CreditCard{ClosingDay: 29, DueDay: 4},
			dates: nil,
			ok:    false,
		},
		"invalid closing day gives no window": {
			card:  CreditCard{ClosingDay: 0, DueDay: 4},
			dates: []time.Time{d(2026, 1, 20)},
			ok:    false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			window, ok := InvoicePaymentWindowFromCreditCard(tc.card, tc.dates)

			assert.Equal(t, tc.ok, ok)
			assert.Equal(t, tc.expected, window)
		})
	}
}

func TestInvoicePaymentWindowFromMeta(t *testing.T) {
	str := func(s string) *string { return &s }

	tests := map[string]struct {
		meta     *InvoiceMeta
		expected InvoicePaymentWindow
		ok       bool
	}{
		"uses closing date and due date minus one month": {
			meta:     &InvoiceMeta{ClosingDate: str("2026-01-29"), DueDate: str("2026-02-04")},
			expected: InvoicePaymentWindow{PreviousDueDate: d(2026, 1, 4), PeriodEnd: d(2026, 1, 29)},
			ok:       true,
		},
		"nil meta gives no window":       {meta: nil},
		"missing closing date":           {meta: &InvoiceMeta{DueDate: str("2026-02-04")}},
		"missing due date":               {meta: &InvoiceMeta{ClosingDate: str("2026-01-29")}},
		"unparseable dates give nothing": {meta: &InvoiceMeta{ClosingDate: str("29/01"), DueDate: str("04/02")}},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			window, ok := InvoicePaymentWindowFromMeta(tc.meta)

			assert.Equal(t, tc.ok, ok)
			assert.Equal(t, tc.expected, window)
		})
	}
}

func TestInvoiceTotalReconciles(t *testing.T) {
	tests := map[string]struct {
		sum      float64
		declared float64
		expected bool
	}{
		"equal values":                         {sum: -1304.90, declared: -1304.90, expected: true},
		"opposite sign is compared by modulus": {sum: -1304.90, declared: 1304.90, expected: true},
		"within the tolerance":                 {sum: -1304.90, declared: -1304.91, expected: true},
		"beyond the tolerance":                 {sum: -1304.90, declared: -1304.95, expected: false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, InvoiceTotalReconciles(tc.sum, tc.declared))
		})
	}
}
