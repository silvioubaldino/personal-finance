//go:build acceptance

package world

import "github.com/google/uuid"

// WalletRef is a wallet created by the scenario.
type WalletRef struct {
	ID             uuid.UUID
	InitialBalance float64
}

// CardRef is a credit card created by the scenario. InitialLimit feeds invariant I2.
type CardRef struct {
	ID           uuid.UUID
	InitialLimit float64
}

// CategoryRef is a category created by the scenario.
type CategoryRef struct {
	ID       uuid.UUID
	IsIncome bool
}

// SubCategoryRef is a subcategory created by the scenario.
type SubCategoryRef struct {
	ID         uuid.UUID
	CategoryID uuid.UUID
}

// MovementKind says what an alias points at.
type MovementKind int

const (
	// KindSingle is one materialized movement (or credit card purchase item).
	KindSingle MovementKind = iota
	// KindSeries is a monthly recurrent series.
	KindSeries
	// KindInstallments is an installment purchase on a credit card.
	KindInstallments
	// KindTransfer is an internal transfer (the pair).
	KindTransfer
)

// MovementRef is something the scenario can point at by alias.
type MovementRef struct {
	Kind MovementKind

	// KindSingle: the movement id. Updates never change it.
	ID uuid.UUID

	// KindSeries: every description the series has had. Updates and deletes split the
	// recurrent chain and create new recurrent ids, so a series is followed by its
	// description(s) — scenarios keep descriptions of different series distinct.
	Descriptions []string

	// KindInstallments: the installment group returned on creation.
	InstallmentGroupID uuid.UUID
	CardAlias          string

	// KindTransfer: the pair id.
	PairID uuid.UUID
}

// HasDescription reports whether a series has used the description.
func (m MovementRef) HasDescription(desc string) bool {
	for _, d := range m.Descriptions {
		if d == desc {
			return true
		}
	}
	return false
}
