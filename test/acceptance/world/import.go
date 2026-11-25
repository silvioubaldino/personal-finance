//go:build acceptance

package world

import (
	"net/http"

	"github.com/google/uuid"

	"personal-finance/internal/domain"
	"personal-finance/test/acceptance/harness"
)

type ImportState struct {
	Kind        string // "invoice" | "statement"
	CardAlias   string
	WalletAlias string
	TargetID    uuid.UUID

	Extraction domain.StatementExtractResult
	Payload    []domain.ExtractedMovement
	Result     *domain.StatementConfirmResult
}

func (w *World) Upload(path, fileName string, file []byte, fields map[string]string) (harness.Response, error) {
	resp, err := w.client.DoMultipart(w.identity(), path, fileName, file, fields)
	if err != nil {
		return resp, err
	}
	w.last = &resp
	w.asserted = false
	return resp, nil
}

func (w *World) RegisterReading(reading domain.StatementExtractResult) []byte {
	id := "inline-" + uuid.NewString()
	w.vision.Register(id, reading)
	return harness.MinimalPDF(id)
}

func (w *World) SuggestCategory(description string, categoryID uuid.UUID) {
	w.classifier.Suggest(w.UserID, description, domain.CategorySuggestion{CategoryID: &categoryID, Confidence: 0.9})
}

func (w *World) AllMovements() ([]MovementView, error) {
	view, err := w.Period(wideFrom, wideTo)
	if err != nil {
		return nil, err
	}
	out := append([]MovementView{}, view.Movements...)
	resp, err := w.Query(http.MethodGet, "/v2/invoices/detailed?from="+wideFrom+"&to="+wideTo)
	if err != nil {
		return nil, err
	}
	var invoices []InvoiceView
	if err := resp.Decode(&invoices); err != nil {
		return nil, err
	}
	seen := map[uuid.UUID]bool{}
	for _, m := range out {
		if m.ID != nil {
			seen[*m.ID] = true
		}
	}
	for _, inv := range invoices {
		for _, m := range inv.Movements {
			if m.ID != nil && seen[*m.ID] {
				continue
			}
			if m.CreditCardInfo == nil {
				m.CreditCardInfo = &CreditCardInfoView{}
			}
			if m.CreditCardInfo.CreditCardID == nil {
				m.CreditCardInfo.CreditCardID = inv.CreditCard.ID
			}
			if m.CreditCardInfo.InvoiceID == nil {
				m.CreditCardInfo.InvoiceID = inv.ID
			}
			out = append(out, m)
		}
	}
	return out, nil
}
