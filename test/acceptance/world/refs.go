//go:build acceptance

package world

import "github.com/google/uuid"

// WalletRef is a wallet created by the scenario. InitialBalance feeds invariant I3.
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

// MovementRef is something the scenario can point at by alias: a single movement, a
// recurrent series, an installment purchase or a transfer. Filled in by the movement steps.
type MovementRef struct {
	ID uuid.UUID
}
