//go:build acceptance

package steps

import (
	"context"
	"net/http"

	"github.com/cucumber/godog"

	"personal-finance/test/acceptance/world"
)

func registerWalletSteps(sc *godog.ScenarioContext) {
	sc.Step(`^(?:I add )?a wallet "([^"]*)" with balance (-?\d+(?:\.\d+)?)$`, aWalletWithBalance)
}

func aWalletWithBalance(ctx context.Context, name, balance string) error {
	w := world.From(ctx)
	amount, err := money(balance)
	if err != nil {
		return err
	}

	resp, err := w.Act(http.MethodPost, "/v2/wallets/", map[string]any{
		"description":     name,
		"initial_balance": amount,
	})
	if err != nil || !resp.OK() {
		return err
	}

	id, err := createdID(resp)
	if err != nil {
		return err
	}
	w.Wallets[name] = world.WalletRef{ID: id, InitialBalance: amount}
	return nil
}
