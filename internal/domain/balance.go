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

func (p *Period) Validate() error {
	now := time.Now()
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
