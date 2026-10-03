//go:build acceptance

package world

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
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

// CardView is the answer of GET /v2/creditcards/:id.
type CardView struct {
	ID          *uuid.UUID `json:"id"`
	Name        string     `json:"name"`
	CreditLimit float64    `json:"credit_limit"`
}

// Card reads a credit card; CreditLimit is the limit still available.
func (w *World) Card(id uuid.UUID) (CardView, error) {
	resp, err := w.Query(http.MethodGet, "/v2/creditcards/"+id.String())
	if err != nil {
		return CardView{}, err
	}
	var out CardView
	if err := resp.Decode(&out); err != nil {
		return CardView{}, err
	}
	return out, nil
}

// CardInvoices lists every invoice of a card (whatever its due date), oldest first.
func (w *World) CardInvoices(cardID uuid.UUID) ([]InvoiceView, error) {
	path := "/v2/invoices/detailed?" + url.Values{"from": {wideFrom}, "to": {wideTo}}.Encode()
	resp, err := w.Query(http.MethodGet, path)
	if err != nil {
		return nil, err
	}
	var all []InvoiceView
	if err := resp.Decode(&all); err != nil {
		return nil, err
	}

	var out []InvoiceView
	for _, inv := range all {
		if inv.CreditCard.ID != nil && *inv.CreditCard.ID == cardID {
			out = append(out, inv)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DueDate.Before(out[j].DueDate) })
	return out, nil
}

// CardInvoice returns the invoice of the card that is due in the yyyy-mm month, or nil.
func (w *World) CardInvoice(cardID uuid.UUID, month string) (*InvoiceView, error) {
	invoices, err := w.CardInvoices(cardID)
	if err != nil {
		return nil, err
	}
	var found []InvoiceView
	for _, inv := range invoices {
		if inv.DueDate.Format("2006-01") == month {
			found = append(found, inv)
		}
	}
	switch len(found) {
	case 0:
		return nil, nil
	case 1:
		return &found[0], nil
	default:
		return nil, fmt.Errorf("found %d invoices due in %s for the same card, expected one", len(found), month)
	}
}

// TransferLegs returns the two legs of a transfer: the outgoing one first, then the
// incoming one. It fails when the transfer does not have exactly two legs.
func (w *World) TransferLegs(pairID uuid.UUID) ([2]MovementView, error) {
	view, err := w.Period(wideFrom, wideTo)
	if err != nil {
		return [2]MovementView{}, err
	}
	var legs []MovementView
	for _, m := range view.Movements {
		if m.PairID != nil && *m.PairID == pairID {
			legs = append(legs, m)
		}
	}
	if len(legs) != 2 {
		return [2]MovementView{}, fmt.Errorf("transfer %s has %d legs, expected 2", pairID, len(legs))
	}
	if legs[0].Amount > 0 {
		legs[0], legs[1] = legs[1], legs[0]
	}
	return [2]MovementView{legs[0], legs[1]}, nil
}
