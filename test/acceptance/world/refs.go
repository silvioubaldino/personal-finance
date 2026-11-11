//go:build acceptance

package world

import "github.com/google/uuid"

type WalletRef struct {
	ID             uuid.UUID
	InitialBalance float64
}

type CardRef struct {
	ID           uuid.UUID
	InitialLimit float64
}

type CategoryRef struct {
	ID       uuid.UUID
	IsIncome bool
}

type SubCategoryRef struct {
	ID         uuid.UUID
	CategoryID uuid.UUID
}

type MovementKind int

const (
	KindSingle MovementKind = iota
	KindSeries
	KindInstallments
	KindTransfer
)

type MovementRef struct {
	Kind MovementKind

	ID uuid.UUID

	Descriptions []string

	InstallmentGroupID uuid.UUID
	CardAlias          string

	PairID uuid.UUID
}

func (m MovementRef) HasDescription(desc string) bool {
	for _, d := range m.Descriptions {
		if d == desc {
			return true
		}
	}
	return false
}
