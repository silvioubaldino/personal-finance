//go:build acceptance

package world

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
)

// The structs below mirror the JSON contract of the /v2 API on purpose: the suite does not
// import internal/domain/output, so a broken contract breaks the suite.

type IDRef struct {
	ID *uuid.UUID `json:"id"`
}

type CreditCardInfoView struct {
	InvoiceID          *uuid.UUID `json:"invoice_id"`
	CreditCardID       *uuid.UUID `json:"credit_card_id"`
	InstallmentGroupID *uuid.UUID `json:"installment_group_id"`
	InstallmentNumber  *int       `json:"installment_number"`
	TotalInstallments  *int       `json:"total_installments"`
}

type MovementView struct {
	ID             *uuid.UUID          `json:"id"`
	Description    string              `json:"description"`
	Amount         float64             `json:"amount"`
	Date           time.Time           `json:"date"`
	IsPaid         bool                `json:"is_paid"`
	IsRecurrent    bool                `json:"is_recurrent"`
	RecurrentID    *uuid.UUID          `json:"recurrent_id"`
	PairID         *uuid.UUID          `json:"pair_id"`
	TypePayment    string              `json:"type_payment"`
	Wallet         IDRef               `json:"wallet"`
	Category       IDRef               `json:"category"`
	SubCategory    IDRef               `json:"sub_category"`
	CreditCardInfo *CreditCardInfoView `json:"credit_card_info"`
}

type InvoiceView struct {
	ID          *uuid.UUID     `json:"id"`
	CreditCard  IDRef          `json:"credit_card"`
	PeriodStart time.Time      `json:"period_start"`
	PeriodEnd   time.Time      `json:"period_end"`
	DueDate     time.Time      `json:"due_date"`
	PaymentDate *time.Time     `json:"payment_date"`
	Amount      float64        `json:"amount"`
	IsPaid      bool           `json:"is_paid"`
	Wallet      IDRef          `json:"wallet"`
	Movements   []MovementView `json:"movements"`
}

// PeriodView is the answer of GET /v2/movements.
type PeriodView struct {
	Movements []MovementView `json:"movements"`
	Invoices  []InvoiceView  `json:"invoices"`
}

const (
	wideFrom = "2000-01-01"
	wideTo   = "2100-12-31"
)

// MonthBounds returns the first and last day of a yyyy-mm month, as yyyy-mm-dd.
func MonthBounds(month string) (string, string, error) {
	first, err := time.Parse("2006-01", month)
	if err != nil {
		return "", "", fmt.Errorf("invalid month %q: %w", month, err)
	}
	last := first.AddDate(0, 1, -1)
	return first.Format("2006-01-02"), last.Format("2006-01-02"), nil
}

// Period reads GET /v2/movements for [from, to] (yyyy-mm-dd).
func (w *World) Period(from, to string) (PeriodView, error) {
	path := "/v2/movements/?" + url.Values{"from": {from}, "to": {to}}.Encode()
	resp, err := w.Query(http.MethodGet, path)
	if err != nil {
		return PeriodView{}, err
	}
	var out PeriodView
	if err := resp.Decode(&out); err != nil {
		return PeriodView{}, err
	}
	return out, nil
}

// Month reads GET /v2/movements for a whole yyyy-mm month. Recurrent occurrences not yet
// materialized are projected for the month of the "to" date, so this is how they are seen.
func (w *World) Month(month string) (PeriodView, error) {
	from, to, err := MonthBounds(month)
	if err != nil {
		return PeriodView{}, err
	}
	return w.Period(from, to)
}

// FindMovement looks a materialized movement (wallet movement or credit card item) up by
// id across a wide window. It returns nil when the movement does not exist.
func (w *World) FindMovement(id uuid.UUID) (*MovementView, error) {
	view, err := w.Period(wideFrom, wideTo)
	if err != nil {
		return nil, err
	}
	for i := range view.Movements {
		if view.Movements[i].ID != nil && *view.Movements[i].ID == id {
			return &view.Movements[i], nil
		}
	}
	for _, inv := range view.Invoices {
		for i := range inv.Movements {
			if inv.Movements[i].ID != nil && *inv.Movements[i].ID == id {
				return &inv.Movements[i], nil
			}
		}
	}
	return nil, nil
}
