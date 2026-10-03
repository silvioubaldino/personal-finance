package domain

import (
	"errors"
	"time"
)

type Balance struct {
	Expense       float64 `json:"expense"`
	Income        float64 `json:"income"`
	PeriodBalance float64 `json:"period_balance"`
}

type Period struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

func (b *Balance) Consolidate() {
	b.PeriodBalance = b.Income + b.Expense
}

// Validate checks the period using the real clock. Legacy callers use it; /v2 callers
// use ValidateAt with clock.Now(ctx).
func (p *Period) Validate() error {
	return p.ValidateAt(time.Now())
}

// ValidateAt checks the period, defaulting unset bounds to now.
func (p *Period) ValidateAt(now time.Time) error {
	if p.From == p.To {
		return errors.New("date must be informed")
	}

	if p.From.IsZero() {
		p.From = now
	}
	if p.To.IsZero() {
		p.To = now
	}

	if p.From.After(p.To) {
		return errors.New("'from' must be before 'to'")
	}

	return nil
}
