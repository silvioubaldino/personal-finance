//go:build acceptance

package steps

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/cucumber/godog"

	"personal-finance/test/acceptance/world"
)

func registerCreditCardSteps(sc *godog.ScenarioContext) {
	sc.Step(`^a credit card "([^"]*)" with limit `+amountRe+`, closing day (\d+) and due day (\d+), paid from wallet "([^"]*)"$`, aCreditCard)
	sc.Step(`^(?:I make )?a credit card purchase "([^"]*)" of `+amountRe+` on "(\d{4}-\d{2}-\d{2})" on card "([^"]*)" under category "([^"]*)"$`, aPurchase)
	sc.Step(`^(?:I make )?a credit card purchase "([^"]*)" in (\d+) installments of `+amountRe+` starting "(\d{4}-\d{2}-\d{2})" on card "([^"]*)" under category "([^"]*)"$`, anInstallmentPurchase)
	sc.Step(`^the available limit of card "([^"]*)" is (-?\d+(?:\.\d+)?)$`, theAvailableLimitIs)
}

func aCreditCard(ctx context.Context, name, limit, closing, due, wallet string) error {
	w := world.From(ctx)
	amount, err := money(limit)
	if err != nil {
		return err
	}
	walletRef, ok := w.Wallets[wallet]
	if !ok {
		return fmt.Errorf("unknown wallet %q", wallet)
	}
	closingDay, _ := strconv.Atoi(closing)
	dueDay, _ := strconv.Atoi(due)

	resp, err := w.Act(http.MethodPost, "/v2/creditcards/", map[string]any{
		"name":              name,
		"credit_limit":      amount,
		"closing_day":       closingDay,
		"due_day":           dueDay,
		"default_wallet_id": walletRef.ID,
	})
	if err != nil || !resp.OK() {
		return err
	}

	id, err := createdID(resp)
	if err != nil {
		return err
	}
	w.Cards[name] = world.CardRef{ID: id, InitialLimit: amount}
	return nil
}

func purchase(ctx context.Context, desc string, amount float64, date, card, category string, installments int) error {
	w := world.From(ctx)
	cardRef, ok := w.Cards[card]
	if !ok {
		return fmt.Errorf("unknown credit card %q", card)
	}
	categoryRef, ok := w.Categories[category]
	if !ok {
		return fmt.Errorf("unknown category %q", category)
	}
	day, err := dateOf(date)
	if err != nil {
		return err
	}

	info := map[string]any{"credit_card_id": cardRef.ID}
	if installments > 0 {
		info["installment_number"] = 1
		info["total_installments"] = installments
	}

	resp, err := w.Act(http.MethodPost, "/v2/movements/", map[string]any{
		"description":      desc,
		"amount":           -amount,
		"date":             apiDate(day),
		"category_id":      categoryRef.ID,
		"type_payment":     "credit_card",
		"is_paid":          false,
		"credit_card_info": info,
	})
	if err != nil || !resp.OK() {
		return err
	}

	var created world.MovementView
	if err := resp.Decode(&created); err != nil {
		return err
	}
	if created.ID == nil {
		return fmt.Errorf("created purchase came without id: %s", resp)
	}

	if installments > 0 {
		if created.CreditCardInfo == nil || created.CreditCardInfo.InstallmentGroupID == nil {
			return fmt.Errorf("installment purchase came without installment_group_id: %s", resp)
		}
		w.Movements[desc] = world.MovementRef{
			Kind:               world.KindInstallments,
			InstallmentGroupID: *created.CreditCardInfo.InstallmentGroupID,
			CardAlias:          card,
		}
		return nil
	}
	w.Movements[desc] = world.MovementRef{Kind: world.KindSingle, ID: *created.ID}
	return nil
}

func aPurchase(ctx context.Context, desc, amount, date, card, category string) error {
	value, err := money(amount)
	if err != nil {
		return err
	}
	return purchase(ctx, desc, value, date, card, category, 0)
}

func anInstallmentPurchase(ctx context.Context, desc, count, amount, date, card, category string) error {
	value, err := money(amount)
	if err != nil {
		return err
	}
	n, _ := strconv.Atoi(count)
	return purchase(ctx, desc, value, date, card, category, n)
}

func theAvailableLimitIs(ctx context.Context, card, expected string) error {
	w := world.From(ctx)
	ref, ok := w.Cards[card]
	if !ok {
		return fmt.Errorf("unknown credit card %q", card)
	}
	want, err := money(expected)
	if err != nil {
		return err
	}
	view, err := w.Card(ref.ID)
	if err != nil {
		return err
	}
	if world.Cents(view.CreditLimit) != world.Cents(want) {
		return fmt.Errorf("available limit of card %q: expected %.2f, got %.2f", card, want, view.CreditLimit)
	}
	return nil
}
