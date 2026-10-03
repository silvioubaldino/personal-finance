//go:build acceptance

package steps

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/cucumber/godog"

	"personal-finance/test/acceptance/world"
)

// defaultTypePayment is what the suite sends for plain wallet movements.
const defaultTypePayment = "debit_card"

const amountRe = `(\d+(?:\.\d+)?)`

func registerMovementSteps(sc *godog.ScenarioContext) {
	sc.Step(`^(?:I add )?a (pending|paid) (expense|income) "([^"]*)" of `+amountRe+` on "(\d{4}-\d{2}-\d{2})" in wallet "([^"]*)" under category "([^"]*)"$`, addMovement)
	sc.Step(`^(?:I add )?a (pending|paid) monthly recurrent (expense|income) "([^"]*)" of `+amountRe+` starting "(\d{4}-\d{2}-\d{2})" in wallet "([^"]*)" under category "([^"]*)"$`, addRecurrent)

	sc.Step(`^I pa(?:y|id) `+refRe+`(?: on "(\d{4}-\d{2}-\d{2})")?$`, payMovement)
	sc.Step(`^I revert the payment of `+refRe+`$`, revertPayment)

	sc.Step(`^I update only `+refRe+` with:$`, updateOneWithTable)
	sc.Step(`^I update `+refRe+` and all next with:$`, updateAllNextWithTable)
	sc.Step(`^I update only `+refRe+` setting amount to (-?\d+(?:\.\d+)?)$`, updateOneSettingAmount)
	sc.Step(`^I update `+refRe+` and all next setting amount to (-?\d+(?:\.\d+)?)$`, updateAllNextSettingAmount)

	sc.Step(`^I delete only `+refRe+`$`, deleteOne)
	sc.Step(`^I delete `+refRe+` and all next$`, deleteAllNext)
}

func apiDate(d time.Time) string { return d.Format(time.RFC3339) }

func signed(kind string, amount float64) float64 {
	if kind == "expense" {
		return -amount
	}
	return amount
}

func addMovement(ctx context.Context, state, kind, desc, amount, date, wallet, category string) error {
	return createMovement(ctx, state, kind, desc, amount, date, wallet, category, false)
}

func addRecurrent(ctx context.Context, state, kind, desc, amount, date, wallet, category string) error {
	return createMovement(ctx, state, kind, desc, amount, date, wallet, category, true)
}

func createMovement(ctx context.Context, state, kind, desc, rawAmount, rawDate, wallet, category string, recurrent bool) error {
	w := world.From(ctx)

	amount, err := money(rawAmount)
	if err != nil {
		return err
	}
	date, err := dateOf(rawDate)
	if err != nil {
		return err
	}
	walletRef, ok := w.Wallets[wallet]
	if !ok {
		return fmt.Errorf("unknown wallet %q", wallet)
	}
	categoryRef, ok := w.Categories[category]
	if !ok {
		return fmt.Errorf("unknown category %q", category)
	}

	resp, err := w.Act(http.MethodPost, "/v2/movements/", map[string]any{
		"description":  desc,
		"amount":       signed(kind, amount),
		"date":         apiDate(date),
		"wallet_id":    walletRef.ID,
		"category_id":  categoryRef.ID,
		"type_payment": defaultTypePayment,
		"is_paid":      state == "paid",
		"is_recurrent": recurrent,
	})
	if err != nil || !resp.OK() {
		return err
	}

	var created world.MovementView
	if err := resp.Decode(&created); err != nil {
		return err
	}
	if created.ID == nil {
		return fmt.Errorf("created movement came without id: %s", resp)
	}

	if recurrent {
		w.Movements[desc] = world.MovementRef{Kind: world.KindSeries, Descriptions: []string{desc}}
	} else {
		w.Movements[desc] = world.MovementRef{Kind: world.KindSingle, ID: *created.ID}
	}
	return nil
}

func payMovement(ctx context.Context, ref, on string) error {
	w := world.From(ctx)
	t, err := resolve(w, ref)
	if err != nil {
		return err
	}

	date := t.view.Date
	if on != "" {
		if date, err = dateOf(on); err != nil {
			return err
		}
	}

	_, err = w.Act(http.MethodPost, "/v2/movements/"+t.id.String()+"/pay?"+url.Values{"date": {date.Format("2006-01-02")}}.Encode(), nil)
	return err
}

func revertPayment(ctx context.Context, ref string) error {
	w := world.From(ctx)
	t, err := resolve(w, ref)
	if err != nil {
		return err
	}
	_, err = w.Act(http.MethodPost, "/v2/movements/"+t.id.String()+"/revert-pay", nil)
	return err
}

// updateBody is the full movement the clients send on PUT: the state read from the API
// with the scenario's overrides on top (the API replaces the fields, it does not merge).
func updateBody(w *world.World, cur world.MovementView, overrides map[string]string) (map[string]any, error) {
	body := map[string]any{
		"description":  cur.Description,
		"amount":       cur.Amount,
		"date":         apiDate(cur.Date),
		"type_payment": cur.TypePayment,
		"is_paid":      cur.IsPaid,
		"is_recurrent": cur.IsRecurrent,
	}
	if cur.Wallet.ID != nil {
		body["wallet_id"] = cur.Wallet.ID
	}
	if cur.Category.ID != nil {
		body["category_id"] = cur.Category.ID
	}
	if cur.SubCategory.ID != nil {
		body["sub_category_id"] = cur.SubCategory.ID
	}
	// A physical occurrence of a series keeps pointing at its series.
	if cur.RecurrentID != nil && cur.ID != nil && *cur.RecurrentID != *cur.ID {
		body["recurrent_id"] = cur.RecurrentID
	}

	for field, value := range overrides {
		switch field {
		case "description":
			body["description"] = value
		case "amount":
			amount, err := money(value)
			if err != nil {
				return nil, err
			}
			body["amount"] = amount
		case "date":
			date, err := dateOf(value)
			if err != nil {
				return nil, err
			}
			body["date"] = apiDate(date)
		case "wallet":
			ref, ok := w.Wallets[value]
			if !ok {
				return nil, fmt.Errorf("unknown wallet %q", value)
			}
			body["wallet_id"] = ref.ID
		case "category":
			ref, ok := w.Categories[value]
			if !ok {
				return nil, fmt.Errorf("unknown category %q", value)
			}
			body["category_id"] = ref.ID
		case "subcategory":
			ref, ok := w.SubCats[value]
			if !ok {
				return nil, fmt.Errorf("unknown subcategory %q", value)
			}
			body["sub_category_id"] = ref.ID
		default:
			return nil, fmt.Errorf("unknown update field %q (use description, amount, date, wallet, category, subcategory)", field)
		}
	}
	return body, nil
}

func update(ctx context.Context, ref string, allNext bool, overrides map[string]string) error {
	w := world.From(ctx)
	t, err := resolve(w, ref)
	if err != nil {
		return err
	}
	body, err := updateBody(w, t.view, overrides)
	if err != nil {
		return err
	}

	path := "/v2/movements/" + t.id.String()
	if allNext {
		path += "/all-next"
	}
	resp, err := w.Act(http.MethodPut, path, body)
	if err != nil {
		return err
	}

	// The series is followed by its descriptions (see world.MovementRef).
	if desc, ok := overrides["description"]; ok && resp.OK() && t.series != "" {
		series := w.Movements[t.series]
		if !series.HasDescription(desc) {
			series.Descriptions = append(series.Descriptions, desc)
			w.Movements[t.series] = series
		}
	}
	return nil
}

func updateOneWithTable(ctx context.Context, ref string, table *godog.Table) error {
	overrides, err := tableKV(table)
	if err != nil {
		return err
	}
	return update(ctx, ref, false, overrides)
}

func updateAllNextWithTable(ctx context.Context, ref string, table *godog.Table) error {
	overrides, err := tableKV(table)
	if err != nil {
		return err
	}
	return update(ctx, ref, true, overrides)
}

func updateOneSettingAmount(ctx context.Context, ref, amount string) error {
	return update(ctx, ref, false, map[string]string{"amount": amount})
}

func updateAllNextSettingAmount(ctx context.Context, ref, amount string) error {
	return update(ctx, ref, true, map[string]string{"amount": amount})
}

func deleteMovement(ctx context.Context, ref string, allNext bool) error {
	w := world.From(ctx)
	t, err := resolve(w, ref)
	if err != nil {
		return err
	}

	path := "/v2/movements/" + t.id.String()
	if allNext {
		path += "/all-next"
	}
	_, err = w.Act(http.MethodDelete, path+"?"+url.Values{"date": {t.view.Date.Format("2006-01-02")}}.Encode(), nil)
	return err
}

func deleteOne(ctx context.Context, ref string) error { return deleteMovement(ctx, ref, false) }

func deleteAllNext(ctx context.Context, ref string) error { return deleteMovement(ctx, ref, true) }
