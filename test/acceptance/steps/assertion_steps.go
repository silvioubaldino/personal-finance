//go:build acceptance

package steps

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/cucumber/godog"

	"personal-finance/test/acceptance/world"
)

// reasons maps the "rejected as" vocabulary to HTTP statuses.
var reasons = map[string]int{
	"invalid input":        http.StatusBadRequest,
	"not found":            http.StatusNotFound,
	"forbidden":            http.StatusForbidden,
	"conflict":             http.StatusConflict,
	"insufficient balance": http.StatusUnprocessableEntity,
	"insufficient limit":   http.StatusUnprocessableEntity,
}

func registerAssertionSteps(sc *godog.ScenarioContext) {
	sc.Step(`^the operation succeeds$`, theOperationSucceeds)
	sc.Step(`^the operation is rejected as "([^"]*)"$`, theOperationIsRejectedAs)
	sc.Step(`^the balance of wallet "([^"]*)" is (-?\d+(?:\.\d+)?)$`, theBalanceOfWalletIs)
	sc.Step(`^the occurrences of "([^"]*)" are:$`, theOccurrencesAre)
	sc.Step(`^there is no occurrence of "([^"]*)" in "(\d{4}-\d{2})"$`, thereIsNoOccurrence)
	sc.Step(`^`+refRe+` is (paid|pending)$`, theMovementIs)
	sc.Step(`^the movements of wallet "([^"]*)" in "(\d{4}-\d{2})" are:$`, theMovementsOfWalletAre)
}

func theOperationSucceeds(ctx context.Context) error {
	w := world.From(ctx)
	last := w.Last()
	if last == nil {
		return fmt.Errorf("no operation has been performed yet")
	}
	w.MarkAsserted()
	if !last.OK() {
		return fmt.Errorf("expected the operation to succeed, got %s", last)
	}
	return nil
}

func theOperationIsRejectedAs(ctx context.Context, reason string) error {
	w := world.From(ctx)
	want, ok := reasons[reason]
	if !ok {
		return fmt.Errorf("unknown rejection reason %q", reason)
	}
	last := w.Last()
	if last == nil {
		return fmt.Errorf("no operation has been performed yet")
	}
	w.MarkAsserted()
	if last.Status != want {
		return fmt.Errorf("expected the operation to be rejected as %q (HTTP %d), got %s", reason, want, last)
	}
	return nil
}

func theBalanceOfWalletIs(ctx context.Context, name, expected string) error {
	w := world.From(ctx)
	ref, ok := w.Wallets[name]
	if !ok {
		return fmt.Errorf("unknown wallet %q", name)
	}
	want, err := money(expected)
	if err != nil {
		return err
	}

	resp, err := w.Query(http.MethodGet, "/v2/wallets/"+ref.ID.String())
	if err != nil {
		return err
	}
	var got struct {
		Balance float64 `json:"balance"`
	}
	if err := resp.Decode(&got); err != nil {
		return err
	}
	if world.Cents(got.Balance) != world.Cents(want) {
		return fmt.Errorf("balance of wallet %q: expected %.2f, got %.2f", name, want, got.Balance)
	}
	return nil
}

// theOccurrencesAre checks the listed months of a series, one row each: amount, paid and,
// optionally, description. Months not listed are not checked.
func theOccurrencesAre(ctx context.Context, alias string, table *godog.Table) error {
	w := world.From(ctx)
	for _, row := range tableRows(table) {
		month := row["month"]
		t, err := resolveOccurrence(w, alias, month)
		if err != nil {
			return err
		}

		if want, ok := row["amount"]; ok {
			amount, err := money(want)
			if err != nil {
				return err
			}
			if world.Cents(t.view.Amount) != world.Cents(amount) {
				return fmt.Errorf("occurrence of %q in %s: expected amount %.2f, got %.2f", alias, month, amount, t.view.Amount)
			}
		}
		if want, ok := row["paid"]; ok {
			paid, err := yesNo(want)
			if err != nil {
				return err
			}
			if t.view.IsPaid != paid {
				return fmt.Errorf("occurrence of %q in %s: expected paid=%s, got paid=%s", alias, month, want, yesNoText(t.view.IsPaid))
			}
		}
		if want, ok := row["description"]; ok && t.view.Description != want {
			return fmt.Errorf("occurrence of %q in %s: expected description %q, got %q", alias, month, want, t.view.Description)
		}
	}
	return nil
}

func thereIsNoOccurrence(ctx context.Context, alias, month string) error {
	w := world.From(ctx)
	t, err := resolveOccurrence(w, alias, month)
	if err == nil {
		return fmt.Errorf("expected no occurrence of %q in %s, found %q of %.2f (paid=%s)",
			alias, month, t.view.Description, t.view.Amount, yesNoText(t.view.IsPaid))
	}
	if strings.HasPrefix(err.Error(), "there is no occurrence") {
		return nil
	}
	return err
}

func theMovementIs(ctx context.Context, ref, state string) error {
	t, err := resolve(world.From(ctx), ref)
	if err != nil {
		return err
	}
	if t.view.IsPaid != (state == "paid") {
		return fmt.Errorf("%s: expected it to be %s, but paid=%s", ref, state, yesNoText(t.view.IsPaid))
	}
	return nil
}

// theMovementsOfWalletAre checks the exact set of wallet movements of a month (credit card
// purchases are not wallet movements; they show up in invoices).
func theMovementsOfWalletAre(ctx context.Context, wallet, month string, table *godog.Table) error {
	w := world.From(ctx)
	ref, ok := w.Wallets[wallet]
	if !ok {
		return fmt.Errorf("unknown wallet %q", wallet)
	}
	view, err := w.Month(month)
	if err != nil {
		return err
	}

	var got []string
	for _, m := range view.Movements {
		if m.Wallet.ID != nil && *m.Wallet.ID == ref.ID {
			got = append(got, movementLine(m.Description, m.Amount, m.IsPaid))
		}
	}

	var want []string
	for _, row := range tableRows(table) {
		amount, err := money(row["amount"])
		if err != nil {
			return err
		}
		paid, err := yesNo(row["paid"])
		if err != nil {
			return err
		}
		want = append(want, movementLine(row["description"], amount, paid))
	}

	sort.Strings(got)
	sort.Strings(want)
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		return fmt.Errorf("movements of wallet %q in %s:\nexpected:\n  %s\ngot:\n  %s",
			wallet, month, strings.Join(want, "\n  "), strings.Join(got, "\n  "))
	}
	return nil
}

func movementLine(description string, amount float64, paid bool) string {
	return fmt.Sprintf("%s | %.2f | paid=%s", description, amount, yesNoText(paid))
}
