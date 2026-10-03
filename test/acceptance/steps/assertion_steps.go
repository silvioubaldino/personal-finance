//go:build acceptance

package steps

import (
	"context"
	"fmt"
	"net/http"

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
