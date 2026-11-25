//go:build acceptance

package steps

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/cucumber/godog"
	"github.com/google/uuid"

	"personal-finance/internal/domain"
	"personal-finance/test/acceptance/harness"
	"personal-finance/test/acceptance/world"
)

const importTargetRe = `(?:card|wallet) "([^"]*)"`

func registerImportSteps(sc *godog.ScenarioContext) {
	sc.Step(`^I (import|extract) the (protected )?(invoice|statement) "([^"]*)" (?:into|for) `+importTargetRe+`(?: with password "([^"]*)")?$`, importFixture)
	sc.Step(`^I (import|extract) an? (invoice|statement) (?:into|for) `+importTargetRe+`(?: declaring total (-?\d+(?:\.\d+)?))? read as:$`, importInline)
	sc.Step(`^I confirm the import$`, confirmImport)
	sc.Step(`^the AI suggests category "([^"]*)" for "([^"]*)"$`, theAISuggestsCategory)

	sc.Step(`^I accept the suggested link for "([^"]*)"$`, acceptSuggestedLink)
	sc.Step(`^I reject the link for "([^"]*)"$`, rejectLink)
	sc.Step(`^I link "([^"]*)" to the purchase "([^"]*)"$`, linkToPurchase)
	sc.Step(`^I re-include "([^"]*)"$`, reInclude)
	sc.Step(`^I link the line "([^"]*)" to the recurrent (?:expense|income) "([^"]*)"$`, linkToRecurrence)

	sc.Step(`^the extraction marks:$`, theExtractionMarks)
	sc.Step(`^the import created (\d+) and skipped (\d+)$`, theImportCreatedAndSkipped)
	sc.Step(`^the invoice of card "([^"]*)" due in `+monthRe+` matches the document total$`, theInvoiceMatchesTheDocumentTotal)
	sc.Step(`^every item of the document is in the app$`, everyItemIsInTheApp)
	sc.Step(`^the installments of "([^"]*)" on card "([^"]*)" are:$`, theInstallmentsAre)
	sc.Step(`^"([^"]*)" is (uncategorized income|uncategorized|in category "([^"]*)")$`, theItemIsInCategory)
}

func importFixture(ctx context.Context, mode, protected, kind, fixture, target, password string) error {
	name := "document.pdf"
	if protected != "" {
		name = "document-protegido.pdf"
	}
	pdf, err := os.ReadFile(filepath.Join(harness.FixturesDir(), fixture, name))
	if err != nil {
		return fmt.Errorf("fixture %q: %w", fixture, err)
	}
	return runImport(ctx, mode, kind, target, pdf, password)
}

func importInline(ctx context.Context, mode, kind, target, total string, table *godog.Table) error {
	w := world.From(ctx)
	reading := domain.StatementExtractResult{DocumentType: domain.DocumentType(kind), Confidence: 0.97}
	for _, row := range tableRows(table) {
		amount, err := money(row["amount"])
		if err != nil {
			return err
		}
		m := domain.ExtractedMovement{Date: row["date"], Description: row["description"], Amount: amount}
		if kind == "invoice" {
			m.TypePayment = domain.TypePaymentCreditCard
		}
		if inst := row["installment"]; inst != "" {
			n, t, err := installmentOf(inst)
			if err != nil {
				return err
			}
			m.InstallmentNumber, m.TotalInstallments = &n, &t
		}
		reading.Movements = append(reading.Movements, m)
	}
	if total != "" {
		v, err := money(total)
		if err != nil {
			return err
		}
		reading.InvoiceMeta = &domain.InvoiceMeta{TotalAmount: &v}
	}
	return runImport(ctx, mode, kind, target, w.RegisterReading(reading), "")
}

func runImport(ctx context.Context, mode, kind, target string, pdf []byte, password string) error {
	w := world.From(ctx)
	state := &world.ImportState{Kind: kind}
	fields := map[string]string{"source_type": kind}

	if kind == "invoice" {
		card, ok := w.Cards[target]
		if !ok {
			return fmt.Errorf("unknown credit card %q", target)
		}
		state.CardAlias, state.TargetID = target, card.ID
		fields["credit_card_id"] = card.ID.String()
	} else {
		wallet, ok := w.Wallets[target]
		if !ok {
			return fmt.Errorf("unknown wallet %q", target)
		}
		state.WalletAlias, state.TargetID = target, wallet.ID
	}
	if password != "" {
		fields["password"] = password
	}
	w.Import = state

	resp, err := w.Upload("/v2/statements/extract", "document.pdf", pdf, fields)
	if err != nil || !resp.OK() {
		return err // resposta não-2xx fica pendente para "the operation is rejected as"
	}
	if err := resp.Decode(&state.Extraction); err != nil {
		return err
	}
	state.Payload = append([]domain.ExtractedMovement{}, state.Extraction.Movements...)

	if err := classify(w); err != nil {
		return err
	}
	if mode == "import" {
		return confirm(w)
	}
	return nil
}

func classify(w *world.World) error {
	if len(w.Import.Payload) == 0 {
		return nil
	}
	resp, err := w.Act(http.MethodPost, "/v2/statements/classify", map[string]any{"movements": w.Import.Payload})
	if err != nil {
		return err
	}
	if !resp.OK() {
		return fmt.Errorf("classify failed: %s", resp)
	}
	var out domain.StatementClassifyResult
	if err := resp.Decode(&out); err != nil {
		return err
	}
	for i, s := range out.Suggestions {
		if i < len(w.Import.Payload) && s.CategoryID != nil {
			w.Import.Payload[i].CategoryID = s.CategoryID
			w.Import.Payload[i].SubCategoryID = s.SubCategoryID
		}
	}
	return nil
}

func confirm(w *world.World) error {
	state := w.Import
	var (
		path string
		body map[string]any
	)
	if state.Kind == "invoice" {
		path = "/v2/statements/confirm-invoice"
		body = map[string]any{"credit_card_id": state.TargetID, "movements": state.Payload}
	} else {
		path = "/v2/statements/confirm"
		body = map[string]any{"wallet_id": state.TargetID, "movements": state.Payload}
	}
	resp, err := w.Act(http.MethodPost, path, body)
	if err != nil || !resp.OK() {
		return err
	}
	var result domain.StatementConfirmResult
	if err := resp.Decode(&result); err != nil {
		return err
	}
	state.Result = &result
	return nil
}

func confirmImport(ctx context.Context) error {
	w := world.From(ctx)
	if w.Import == nil {
		return errors.New("no import in progress")
	}
	return confirm(w)
}

func theAISuggestsCategory(ctx context.Context, category, description string) error {
	w := world.From(ctx)
	ref, ok := w.Categories[category]
	if !ok {
		return fmt.Errorf("unknown category %q", category)
	}
	w.SuggestCategory(description, ref.ID)
	return nil
}

func payloadItem(w *world.World, description string) (*domain.ExtractedMovement, error) {
	if w.Import == nil {
		return nil, errors.New("no import in progress")
	}
	for i := range w.Import.Payload {
		if w.Import.Payload[i].Description == description {
			return &w.Import.Payload[i], nil
		}
	}
	return nil, fmt.Errorf("the document has no item %q", description)
}

func acceptSuggestedLink(ctx context.Context, description string) error {
	item, err := payloadItem(world.From(ctx), description)
	if err != nil {
		return err
	}
	if item.InstallmentMatch == nil {
		return fmt.Errorf("item %q has no suggested link", description)
	}
	group := item.InstallmentMatch.InstallmentGroupID
	item.InstallmentGroupID = &group
	return nil
}

func rejectLink(ctx context.Context, description string) error {
	item, err := payloadItem(world.From(ctx), description)
	if err != nil {
		return err
	}
	item.InstallmentGroupID = nil
	return nil
}

func linkToPurchase(ctx context.Context, description, purchase string) error {
	w := world.From(ctx)
	item, err := payloadItem(w, description)
	if err != nil {
		return err
	}
	ref, ok := w.Movements[purchase]
	if !ok || ref.Kind != world.KindInstallments {
		return fmt.Errorf("unknown installment purchase %q", purchase)
	}
	group := ref.InstallmentGroupID
	item.InstallmentGroupID = &group
	return nil
}

func reInclude(ctx context.Context, description string) error {
	item, err := payloadItem(world.From(ctx), description)
	if err != nil {
		return err
	}
	item.Excluded, item.ExclusionReason = false, ""
	return nil
}

func linkToRecurrence(ctx context.Context, description, recurrent string) error {
	w := world.From(ctx)
	item, err := payloadItem(w, description)
	if err != nil {
		return err
	}
	ref, ok := w.Movements[recurrent]
	if !ok || ref.Kind != world.KindSeries || ref.RecurrentID == uuid.Nil {
		return fmt.Errorf("unknown recurrent movement %q", recurrent)
	}
	id := ref.RecurrentID
	item.RecurrenceID = &id
	return nil
}

func theExtractionMarks(ctx context.Context, table *godog.Table) error {
	w := world.From(ctx)
	if w.Import == nil {
		return errors.New("no import in progress")
	}
	var errs []error
	for _, row := range tableRows(table) {
		desc, as := row["description"], row["as"]
		item, err := extractedItem(w, desc)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if err := checkMark(w, *item, as); err != nil {
			errs = append(errs, fmt.Errorf("%q: %w", desc, err))
		}
	}
	return errors.Join(errs...)
}

func extractedItem(w *world.World, description string) (*domain.ExtractedMovement, error) {
	for i := range w.Import.Extraction.Movements {
		if w.Import.Extraction.Movements[i].Description == description {
			return &w.Import.Extraction.Movements[i], nil
		}
	}
	return nil, fmt.Errorf("the extraction has no item %q", description)
}

func checkMark(w *world.World, item domain.ExtractedMovement, as string) error {
	describe := func() string {
		group := "none"
		if item.InstallmentGroupID != nil {
			group = item.InstallmentGroupID.String()
		}
		return fmt.Sprintf("excluded=%v reason=%q match=%v installment_group_id=%s",
			item.Excluded, item.ExclusionReason, item.InstallmentMatch != nil, group)
	}
	switch {
	case as == "invoice payment":
		if !item.Excluded || item.ExclusionReason != domain.ExclusionReasonInvoicePayment {
			return fmt.Errorf("expected invoice payment, got %s", describe())
		}
	case as == "future installment":
		if !item.Excluded || item.ExclusionReason != domain.ExclusionReasonFutureInstallment {
			return fmt.Errorf("expected future installment, got %s", describe())
		}
	case as == "new":
		if item.Excluded || item.InstallmentMatch != nil || item.InstallmentGroupID != nil {
			return fmt.Errorf("expected a new item, got %s", describe())
		}
	case strings.HasPrefix(as, "linked to "):
		group, err := purchaseGroup(w, strings.Trim(strings.TrimPrefix(as, "linked to "), `"`))
		if err != nil {
			return err
		}
		if item.InstallmentGroupID == nil || *item.InstallmentGroupID != group {
			return fmt.Errorf("expected to come linked to group %s, got %s", group, describe())
		}
	case strings.HasPrefix(as, "suggested "):
		group, err := purchaseGroup(w, strings.Trim(strings.TrimPrefix(as, "suggested "), `"`))
		if err != nil {
			return err
		}
		if item.InstallmentMatch == nil || item.InstallmentMatch.InstallmentGroupID != group || item.InstallmentGroupID != nil {
			return fmt.Errorf("expected a suggestion (not a link) to group %s, got %s", group, describe())
		}
	default:
		return fmt.Errorf("unknown mark %q", as)
	}
	return nil
}

func purchaseGroup(w *world.World, purchase string) (uuid.UUID, error) {
	ref, ok := w.Movements[purchase]
	if !ok || ref.Kind != world.KindInstallments {
		return uuid.Nil, fmt.Errorf("unknown installment purchase %q", purchase)
	}
	return ref.InstallmentGroupID, nil
}

func theImportCreatedAndSkipped(ctx context.Context, created, skipped string) error {
	w := world.From(ctx)
	if w.Import == nil || w.Import.Result == nil {
		return errors.New("no confirmed import")
	}
	c, _ := strconv.Atoi(created)
	s, _ := strconv.Atoi(skipped)
	if r := w.Import.Result; r.Created != c || r.Skipped != s {
		return fmt.Errorf("expected created=%d skipped=%d, got created=%d skipped=%d errors=%v", c, s, r.Created, r.Skipped, r.Errors)
	}
	return nil
}

func theInvoiceMatchesTheDocumentTotal(ctx context.Context, card, month string) error {
	w := world.From(ctx)
	if w.Import == nil || w.Import.Extraction.InvoiceMeta == nil || w.Import.Extraction.InvoiceMeta.TotalAmount == nil {
		return errors.New("the imported document declares no total")
	}
	ref, ok := w.Cards[card]
	if !ok {
		return fmt.Errorf("unknown credit card %q", card)
	}
	inv, err := w.CardInvoice(ref.ID, month)
	if err != nil {
		return err
	}
	if inv == nil {
		return fmt.Errorf("card %q has no invoice due in %s", card, month)
	}
	want := -abs(*w.Import.Extraction.InvoiceMeta.TotalAmount)
	if world.Cents(inv.Amount) != world.Cents(want) {
		return fmt.Errorf("invoice of card %q due in %s: document total %.2f, app has %.2f\n%s",
			card, month, want, inv.Amount, describeInvoice(*inv))
	}
	return nil
}

func describeInvoice(inv world.InvoiceView) string {
	var b strings.Builder
	for _, m := range inv.Movements {
		fmt.Fprintf(&b, "  %s  %-45s %10.2f\n", m.Date.Format("2006-01-02"), m.Description, m.Amount)
	}
	return b.String()
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func everyItemIsInTheApp(ctx context.Context) error {
	w := world.From(ctx)
	if w.Import == nil {
		return errors.New("no import in progress")
	}
	all, err := w.AllMovements()
	if err != nil {
		return err
	}
	var pool []world.MovementView
	for _, m := range all {
		if w.Import.Kind == "invoice" {
			if m.CreditCardInfo != nil && m.CreditCardInfo.CreditCardID != nil && *m.CreditCardInfo.CreditCardID == w.Import.TargetID {
				pool = append(pool, m)
			}
		} else if m.Wallet.ID != nil && *m.Wallet.ID == w.Import.TargetID && m.TypePayment != "credit_card" {
			pool = append(pool, m)
		}
	}

	used := make([]bool, len(pool))
	var missing []string
	for _, item := range w.Import.Payload {
		if item.Excluded || (w.Import.Kind == "invoice" && domain.IsInvoicePaymentDescription(item.Description)) {
			continue
		}
		found := -1
		for i, m := range pool {
			if used[i] || world.Cents(m.Amount) != world.Cents(item.Amount) {
				continue
			}
			if item.InstallmentGroupID != nil {
				info := m.CreditCardInfo
				if info != nil && info.InstallmentGroupID != nil && *info.InstallmentGroupID == *item.InstallmentGroupID &&
					info.InstallmentNumber != nil && item.InstallmentNumber != nil && *info.InstallmentNumber == *item.InstallmentNumber {
					found = i
					break
				}
				continue
			}
			if m.Description == item.Description {
				found = i
				break
			}
		}
		if found < 0 {
			missing = append(missing, fmt.Sprintf("%s %q %.2f", item.Date, item.Description, item.Amount))
			continue
		}
		used[found] = true
	}
	if len(missing) > 0 {
		return fmt.Errorf("%d item(s) of the document are not in the app:\n  %s", len(missing), strings.Join(missing, "\n  "))
	}
	return nil
}

func theInstallmentsAre(ctx context.Context, purchase, card string, table *godog.Table) error {
	w := world.From(ctx)
	ref, ok := w.Cards[card]
	if !ok {
		return fmt.Errorf("unknown credit card %q", card)
	}
	invoices, err := w.CardInvoices(ref.ID)
	if err != nil {
		return err
	}
	var group *uuid.UUID
	if m, ok := w.Movements[purchase]; ok && m.Kind == world.KindInstallments {
		group = &m.InstallmentGroupID
	}
	root := descriptionRoot(purchase)

	type inst struct {
		n      int
		due    string
		amount int64
	}
	var got []inst
	for _, inv := range invoices {
		for _, m := range inv.Movements {
			info := m.CreditCardInfo
			if info == nil || info.InstallmentNumber == nil {
				continue
			}
			inGroup := group != nil && info.InstallmentGroupID != nil && *info.InstallmentGroupID == *group
			if !inGroup && descriptionRoot(m.Description) != root {
				continue
			}
			got = append(got, inst{*info.InstallmentNumber, inv.DueDate.Format("2006-01"), world.Cents(m.Amount)})
		}
	}
	sort.Slice(got, func(i, j int) bool {
		if got[i].n != got[j].n {
			return got[i].n < got[j].n
		}
		return got[i].due < got[j].due
	})

	var want []inst
	for _, row := range tableRows(table) {
		n, err := strconv.Atoi(row["installment"])
		if err != nil {
			return fmt.Errorf("invalid installment %q", row["installment"])
		}
		amount, err := money(row["amount"])
		if err != nil {
			return err
		}
		want = append(want, inst{n, row["due"], world.Cents(amount)})
	}

	format := func(list []inst) string {
		var b strings.Builder
		for _, i := range list {
			fmt.Fprintf(&b, "  %2d  %s  %.2f\n", i.n, i.due, float64(i.amount)/100)
		}
		return b.String()
	}
	if format(got) != format(want) {
		return fmt.Errorf("installments of %q on card %q:\nexpected:\n%sgot:\n%s", purchase, card, format(want), format(got))
	}
	return nil
}

func descriptionRoot(desc string) string {
	return strings.TrimSpace(domain.NormalizeDescription(domain.StripInstallmentSuffix(desc)))
}

func theItemIsInCategory(ctx context.Context, description, kind, category string) error {
	w := world.From(ctx)
	all, err := w.AllMovements()
	if err != nil {
		return err
	}
	var want uuid.UUID
	switch kind {
	case "uncategorized":
		want = uuid.MustParse(domain.UncategorizedCategoryID)
	case "uncategorized income":
		want = uuid.MustParse(domain.UncategorizedIncomeCategoryID)
	default:
		ref, ok := w.Categories[category]
		if !ok {
			return fmt.Errorf("unknown category %q", category)
		}
		want = ref.ID
	}
	for _, m := range all {
		if m.Description != description {
			continue
		}
		if m.Category.ID == nil || *m.Category.ID != want {
			got := "none"
			if m.Category.ID != nil {
				got = m.Category.ID.String()
			}
			return fmt.Errorf("%q: expected category %s (%s), got %s", description, want, kind, got)
		}
		return nil
	}
	return fmt.Errorf("no movement %q in the app", description)
}

func installmentOf(raw string) (int, int, error) {
	a, b, ok := strings.Cut(raw, "/")
	n, err1 := strconv.Atoi(strings.TrimSpace(a))
	t, err2 := strconv.Atoi(strings.TrimSpace(b))
	if !ok || err1 != nil || err2 != nil {
		return 0, 0, fmt.Errorf("invalid installment %q, expected n/total", raw)
	}
	return n, t, nil
}
