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
	{name: "I1 invoice amount", check: invoiceAmountMatchesItems},
	{name: "I2 credit card limit", check: cardLimitMatchesInvoices},
	{name: "I3 wallet balance", check: walletBalanceMatchesMovements},
	{name: "I4 transfer pairs", check: transferPairsAreIntact},
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

// invoiceAmountMatchesItems (I1): invoices.amount = the sum of the items of the invoice,
// everything but the invoice payment movement (the same rule as /recalculate).
func invoiceAmountMatchesItems(w *world.World) error {
	type row struct {
		DueDate string
		Card    string
		Amount  float64
		Items   float64
	}
	var rows []row

	err := w.DB().Raw(`
		SELECT to_char(i.due_date, 'YYYY-MM') AS due_date,
		       c.name AS card,
		       i.amount AS amount,
		       COALESCE((
		           SELECT SUM(m.amount) FROM movements m
		           WHERE m.invoice_id = i.id AND m.user_id = i.user_id
		             AND COALESCE(m.type_payment, '') <> 'invoice_payment'
		       ), 0) AS items
		FROM invoices i JOIN credit_cards c ON c.id = i.credit_card_id
		WHERE i.user_id = ?`, w.UserID).Scan(&rows).Error
	if err != nil {
		return err
	}

	var errs []error
	for _, r := range rows {
		if world.Cents(r.Amount) != world.Cents(r.Items) {
			errs = append(errs, fmt.Errorf("invoice of card %q due in %s has amount %.2f but its items add up to %.2f",
				r.Card, r.DueDate, r.Amount, r.Items))
		}
	}
	return errors.Join(errs...)
}

// cardLimitMatchesInvoices (I2): the available limit of a card = its initial limit + the
// amount of its unpaid invoices (amounts are negative). A paid invoice, even partially
// paid, has already given its limit back; the unpaid remainder lives in the next invoice.
func cardLimitMatchesInvoices(w *world.World) error {
	var errs []error
	for name, card := range w.Cards {
		var row struct {
			Limit  float64
			Unpaid float64
		}
		err := w.DB().Raw(`
			SELECT COALESCE(c.credit_limit, 0) AS "limit",
			       COALESCE((
			           SELECT SUM(i.amount) FROM invoices i
			           WHERE i.credit_card_id = c.id AND i.user_id = c.user_id AND NOT i.is_paid
			       ), 0) AS unpaid
			FROM credit_cards c
			WHERE c.id = ? AND c.user_id = ?`, card.ID, w.UserID).Scan(&row).Error
		if err != nil {
			return err
		}

		want := card.InitialLimit + row.Unpaid
		if world.Cents(row.Limit) != world.Cents(want) {
			errs = append(errs, fmt.Errorf("card %q has available limit %.2f but initial limit %.2f + unpaid invoices %.2f = %.2f",
				name, row.Limit, card.InitialLimit, row.Unpaid, want))
		}
	}
	return errors.Join(errs...)
}

// transferPairsAreIntact (I4): every pair_id has exactly two legs, with opposite amounts, in
// two different wallets and in the same paid state.
func transferPairsAreIntact(w *world.World) error {
	type row struct {
		PairID  string
		Legs    int
		Sum     float64
		Wallets int
		States  int
	}
	var rows []row

	err := w.DB().Raw(`
		SELECT pair_id::text AS pair_id,
		       COUNT(*) AS legs,
		       COALESCE(SUM(amount), 0) AS sum,
		       COUNT(DISTINCT wallet_id) AS wallets,
		       COUNT(DISTINCT COALESCE(is_paid, false)) AS states
		FROM movements
		WHERE user_id = ? AND pair_id IS NOT NULL
		GROUP BY pair_id`, w.UserID).Scan(&rows).Error
	if err != nil {
		return err
	}

	var errs []error
	for _, r := range rows {
		if r.Legs != 2 || world.Cents(r.Sum) != 0 || r.Wallets != 2 || r.States != 1 {
			errs = append(errs, fmt.Errorf("transfer %s is broken: %d legs, amounts add up to %.2f, %d wallets, %d paid states",
				r.PairID, r.Legs, r.Sum, r.Wallets, r.States))
		}
	}
	return errors.Join(errs...)
}
