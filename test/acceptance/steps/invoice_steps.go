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

const monthRe = `"(\d{4}-\d{2})"`

func registerInvoiceSteps(sc *godog.ScenarioContext) {
	sc.Step(`^I pa(?:y|id) the invoice of card "([^"]*)" due in `+monthRe+` from wallet "([^"]*)"(?: paying `+amountRe+`)?$`, payInvoice)
	sc.Step(`^I revert the payment of the invoice of card "([^"]*)" due in `+monthRe+`$`, revertInvoicePayment)
	sc.Step(`^I recalculate the invoice of card "([^"]*)" due in `+monthRe+`$`, recalculateInvoice)

	sc.Step(`^the invoice of card "([^"]*)" due in `+monthRe+` has amount (-?\d+(?:\.\d+)?)$`, theInvoiceHasAmount)
	sc.Step(`^the invoice of card "([^"]*)" due in `+monthRe+` is (paid|open)$`, theInvoiceIs)
	sc.Step(`^the invoice of card "([^"]*)" due in `+monthRe+` does not exist$`, theInvoiceDoesNotExist)
	sc.Step(`^the invoice of card "([^"]*)" due in `+monthRe+` was paid on "(\d{4}-\d{2}-\d{2})"$`, theInvoiceWasPaidOn)
	sc.Step(`^the invoices of card "([^"]*)" are:$`, theInvoicesAre)
	sc.Step(`^the invoice of card "([^"]*)" due in `+monthRe+` contains:$`, theInvoiceContains)
}

// invoiceOf finds the invoice of a card due in a month; it fails the step when there is none.
func invoiceOf(w *world.World, card, month string) (world.InvoiceView, error) {
	ref, ok := w.Cards[card]
	if !ok {
		return world.InvoiceView{}, fmt.Errorf("unknown credit card %q", card)
	}
	inv, err := w.CardInvoice(ref.ID, month)
	if err != nil {
		return world.InvoiceView{}, err
	}
	if inv == nil {
		return world.InvoiceView{}, fmt.Errorf("card %q has no invoice due in %s", card, month)
	}
	return *inv, nil
}

func payInvoice(ctx context.Context, card, month, wallet, amount string) error {
	w := world.From(ctx)
	inv, err := invoiceOf(w, card, month)
	if err != nil {
		return err
	}
	walletRef, ok := w.Wallets[wallet]
	if !ok {
		return fmt.Errorf("unknown wallet %q", wallet)
	}

	body := map[string]any{"wallet_id": walletRef.ID}
	if amount != "" {
		value, err := money(amount)
		if err != nil {
			return err
		}
		body["amount"] = value
	}
	_, err = w.Act(http.MethodPost, "/v2/invoices/"+inv.ID.String()+"/pay", body)
	return err
}

func revertInvoicePayment(ctx context.Context, card, month string) error {
	w := world.From(ctx)
	inv, err := invoiceOf(w, card, month)
	if err != nil {
		return err
	}
	_, err = w.Act(http.MethodPost, "/v2/invoices/"+inv.ID.String()+"/revert-pay", nil)
	return err
}

func recalculateInvoice(ctx context.Context, card, month string) error {
	w := world.From(ctx)
	inv, err := invoiceOf(w, card, month)
	if err != nil {
		return err
	}
	_, err = w.Act(http.MethodPost, "/v2/invoices/"+inv.ID.String()+"/recalculate", nil)
	return err
}

func theInvoiceHasAmount(ctx context.Context, card, month, expected string) error {
	inv, err := invoiceOf(world.From(ctx), card, month)
	if err != nil {
		return err
	}
	want, err := money(expected)
	if err != nil {
		return err
	}
	if world.Cents(inv.Amount) != world.Cents(want) {
		return fmt.Errorf("invoice of card %q due in %s: expected amount %.2f, got %.2f", card, month, want, inv.Amount)
	}
	return nil
}

func theInvoiceIs(ctx context.Context, card, month, state string) error {
	inv, err := invoiceOf(world.From(ctx), card, month)
	if err != nil {
		return err
	}
	if inv.IsPaid != (state == "paid") {
		return fmt.Errorf("invoice of card %q due in %s: expected it to be %s, but paid=%s", card, month, state, yesNoText(inv.IsPaid))
	}
	return nil
}

func theInvoiceDoesNotExist(ctx context.Context, card, month string) error {
	w := world.From(ctx)
	ref, ok := w.Cards[card]
	if !ok {
		return fmt.Errorf("unknown credit card %q", card)
	}
	inv, err := w.CardInvoice(ref.ID, month)
	if err != nil {
		return err
	}
	if inv != nil {
		return fmt.Errorf("card %q should have no invoice due in %s, but has one with amount %.2f", card, month, inv.Amount)
	}
	return nil
}

func theInvoiceWasPaidOn(ctx context.Context, card, month, date string) error {
	inv, err := invoiceOf(world.From(ctx), card, month)
	if err != nil {
		return err
	}
	if inv.PaymentDate == nil {
		return fmt.Errorf("invoice of card %q due in %s has no payment date", card, month)
	}
	if got := inv.PaymentDate.Format("2006-01-02"); got != date {
		return fmt.Errorf("invoice of card %q due in %s was paid on %s, expected %s", card, month, got, date)
	}
	return nil
}

// theInvoicesAre checks the exact set of invoices of a card (due | amount | paid).
func theInvoicesAre(ctx context.Context, card string, table *godog.Table) error {
	w := world.From(ctx)
	ref, ok := w.Cards[card]
	if !ok {
		return fmt.Errorf("unknown credit card %q", card)
	}
	invoices, err := w.CardInvoices(ref.ID)
	if err != nil {
		return err
	}

	var got, want []string
	for _, inv := range invoices {
		got = append(got, invoiceLine(inv.DueDate.Format("2006-01"), inv.Amount, inv.IsPaid))
	}
	for _, row := range tableRows(table) {
		amount, err := money(row["amount"])
		if err != nil {
			return err
		}
		paid, err := yesNo(row["paid"])
		if err != nil {
			return err
		}
		want = append(want, invoiceLine(row["due"], amount, paid))
	}

	sort.Strings(got)
	sort.Strings(want)
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		return fmt.Errorf("invoices of card %q:\nexpected:\n  %s\ngot:\n  %s", card, strings.Join(want, "\n  "), strings.Join(got, "\n  "))
	}
	return nil
}

func invoiceLine(due string, amount float64, paid bool) string {
	return fmt.Sprintf("%s | %.2f | paid=%s", due, amount, yesNoText(paid))
}

// theInvoiceContains checks the exact set of items that make up an invoice
// (description | amount | installment, installment written as "k/n" or left empty).
func theInvoiceContains(ctx context.Context, card, month string, table *godog.Table) error {
	inv, err := invoiceOf(world.From(ctx), card, month)
	if err != nil {
		return err
	}

	var got, want []string
	for _, m := range inv.Movements {
		installment := ""
		if info := m.CreditCardInfo; info != nil && info.InstallmentNumber != nil && info.TotalInstallments != nil {
			installment = fmt.Sprintf("%d/%d", *info.InstallmentNumber, *info.TotalInstallments)
		}
		got = append(got, itemLine(m.Description, m.Amount, installment))
	}
	for _, row := range tableRows(table) {
		amount, err := money(row["amount"])
		if err != nil {
			return err
		}
		want = append(want, itemLine(row["description"], amount, row["installment"]))
	}

	sort.Strings(got)
	sort.Strings(want)
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		return fmt.Errorf("items of the invoice of card %q due in %s:\nexpected:\n  %s\ngot:\n  %s",
			card, month, strings.Join(want, "\n  "), strings.Join(got, "\n  "))
	}
	return nil
}

func itemLine(description string, amount float64, installment string) string {
	return fmt.Sprintf("%s | %.2f | %s", description, amount, installment)
}
