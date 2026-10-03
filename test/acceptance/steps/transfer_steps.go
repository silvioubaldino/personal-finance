//go:build acceptance

package steps

import (
	"context"
	"fmt"
	"net/http"

	"github.com/cucumber/godog"

	"personal-finance/test/acceptance/world"
)

func registerTransferSteps(sc *godog.ScenarioContext) {
	sc.Step(`^(?:I add )?a (pending|paid) internal transfer "([^"]*)" of `+amountRe+` from "([^"]*)" to "([^"]*)" on "(\d{4}-\d{2}-\d{2})"$`, addTransfer)
	sc.Step(`^I pay the internal transfer "([^"]*)"$`, payTransfer)
	sc.Step(`^I revert the payment of the internal transfer "([^"]*)"$`, revertTransfer)
	sc.Step(`^I delete the internal transfer "([^"]*)"$`, deleteTransfer)
	sc.Step(`^I update the internal transfer "([^"]*)" with:$`, updateTransfer)
}

// transferState is what the transfer endpoints answer with.
type transferState struct {
	PairID string             `json:"pair_id"`
	Origin world.MovementView `json:"origin_movement"`
	Dest   world.MovementView `json:"destination_movement"`
}

func addTransfer(ctx context.Context, state, name, amount, from, to, date string) error {
	w := world.From(ctx)
	value, err := money(amount)
	if err != nil {
		return err
	}
	origin, ok := w.Wallets[from]
	if !ok {
		return fmt.Errorf("unknown wallet %q", from)
	}
	destination, ok := w.Wallets[to]
	if !ok {
		return fmt.Errorf("unknown wallet %q", to)
	}

	resp, err := w.Act(http.MethodPost, "/v2/transfers/", map[string]any{
		"origin_wallet_id":      origin.ID,
		"destination_wallet_id": destination.ID,
		"amount":                value,
		"date":                  date,
		"description":           name,
		"is_paid":               state == "paid",
	})
	if err != nil || !resp.OK() {
		return err
	}

	var created struct {
		PairID string `json:"pair_id"`
	}
	if err := resp.Decode(&created); err != nil {
		return err
	}
	pairID, err := parseUUID(created.PairID)
	if err != nil {
		return err
	}
	w.Movements[name] = world.MovementRef{Kind: world.KindTransfer, PairID: pairID}
	return nil
}

func transferPath(w *world.World, name string) (string, error) {
	ref, ok := w.Movements[name]
	if !ok || ref.Kind != world.KindTransfer {
		return "", fmt.Errorf("%q is not an internal transfer", name)
	}
	return "/v2/transfers/" + ref.PairID.String(), nil
}

func payTransfer(ctx context.Context, name string) error {
	w := world.From(ctx)
	path, err := transferPath(w, name)
	if err != nil {
		return err
	}
	_, err = w.Act(http.MethodPost, path+"/pay", nil)
	return err
}

func revertTransfer(ctx context.Context, name string) error {
	w := world.From(ctx)
	path, err := transferPath(w, name)
	if err != nil {
		return err
	}
	_, err = w.Act(http.MethodPost, path+"/revert-pay", nil)
	return err
}

func deleteTransfer(ctx context.Context, name string) error {
	w := world.From(ctx)
	path, err := transferPath(w, name)
	if err != nil {
		return err
	}
	_, err = w.Act(http.MethodDelete, path, nil)
	return err
}

// updateTransfer sends the whole transfer: the current legs read from the API with the
// overrides (origin, destination, amount, date, description) on top.
func updateTransfer(ctx context.Context, name string, table *godog.Table) error {
	w := world.From(ctx)
	path, err := transferPath(w, name)
	if err != nil {
		return err
	}
	overrides, err := tableKV(table)
	if err != nil {
		return err
	}

	ref := w.Movements[name]
	legs, err := w.TransferLegs(ref.PairID)
	if err != nil {
		return err
	}
	out, in := legs[0], legs[1]

	body := map[string]any{
		"origin_wallet_id":      out.Wallet.ID,
		"destination_wallet_id": in.Wallet.ID,
		"amount":                in.Amount,
		"date":                  out.Date.Format("2006-01-02"),
		"description":           name,
	}
	for field, value := range overrides {
		switch field {
		case "origin", "destination":
			wallet, ok := w.Wallets[value]
			if !ok {
				return fmt.Errorf("unknown wallet %q", value)
			}
			key := "origin_wallet_id"
			if field == "destination" {
				key = "destination_wallet_id"
			}
			body[key] = wallet.ID
		case "amount":
			amount, err := money(value)
			if err != nil {
				return err
			}
			body["amount"] = amount
		case "date":
			body["date"] = value
		case "description":
			body["description"] = value
		default:
			return fmt.Errorf("unknown transfer field %q (use origin, destination, amount, date, description)", field)
		}
	}

	_, err = w.Act(http.MethodPut, path, body)
	return err
}
