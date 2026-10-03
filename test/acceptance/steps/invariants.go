//go:build acceptance

package steps

import (
	"errors"
	"fmt"

	"personal-finance/test/acceptance/world"
)

// invariant is a rule that must hold for the data of every scenario, whatever the
// scenario did. It runs in the After hook, scoped to the scenario's user_id.
type invariant struct {
	name  string
	check func(w *world.World) error
}

var invariants = []invariant{
	{name: "I3 wallet balance", check: walletBalanceMatchesMovements},
}

func checkInvariants(w *world.World) error {
	var errs []error
	for _, inv := range invariants {
		if err := inv.check(w); err != nil {
			errs = append(errs, errors.New(inv.name+": "+err.Error()))
		}
	}
	return errors.Join(errs...)
}

// walletBalanceMatchesMovements (I3): wallets.balance = initial_balance + the sum of the
// paid movements of the wallet, except credit card purchases and invoice remainders (which
// only reach the wallet through the invoice payment movement).
func walletBalanceMatchesMovements(w *world.World) error {
	type row struct {
		Description    string
		Balance        float64
		InitialBalance float64
		PaidSum        float64
	}
	var rows []row

	err := w.DB().Raw(`
		SELECT w.description AS description,
		       COALESCE(w.balance, 0) AS balance,
		       w.initial_balance AS initial_balance,
		       COALESCE((
		           SELECT SUM(m.amount) FROM movements m
		           WHERE m.wallet_id = w.id AND m.user_id = w.user_id AND m.is_paid
		             AND COALESCE(m.type_payment, '') NOT IN ('credit_card', 'invoice_remainder')
		       ), 0) AS paid_sum
		FROM wallets w
		WHERE w.user_id = ?`, w.UserID).Scan(&rows).Error
	if err != nil {
		return err
	}

	var errs []error
	for _, r := range rows {
		want := r.InitialBalance + r.PaidSum
		if world.Cents(r.Balance) != world.Cents(want) {
			errs = append(errs, fmt.Errorf("wallet %q has balance %.2f but initial balance %.2f + paid movements %.2f = %.2f",
				r.Description, r.Balance, r.InitialBalance, r.PaidSum, want))
		}
	}
	return errors.Join(errs...)
}
