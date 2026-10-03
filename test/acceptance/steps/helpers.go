//go:build acceptance

package steps

import (
	"fmt"
	"strconv"
	"time"

	"github.com/cucumber/godog"
	"github.com/google/uuid"

	"personal-finance/test/acceptance/harness"
)

// money parses an amount captured by a step regexp.
func money(raw string) (float64, error) {
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount %q: %w", raw, err)
	}
	return v, nil
}

// dateOf parses a yyyy-mm-dd step argument.
func dateOf(raw string) (time.Time, error) {
	d, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date %q: %w", raw, err)
	}
	return d, nil
}

// createdID decodes the id of a 201 response.
func createdID(resp harness.Response) (uuid.UUID, error) {
	var out struct {
		ID uuid.UUID `json:"id"`
	}
	if err := resp.Decode(&out); err != nil {
		return uuid.Nil, err
	}
	return out.ID, nil
}

// tableRows turns a Gherkin table with a header row into one map per data row.
func tableRows(t *godog.Table) []map[string]string {
	if t == nil || len(t.Rows) < 2 {
		return nil
	}
	header := t.Rows[0].Cells
	rows := make([]map[string]string, 0, len(t.Rows)-1)
	for _, r := range t.Rows[1:] {
		row := make(map[string]string, len(header))
		for i, h := range header {
			row[h.Value] = r.Cells[i].Value
		}
		rows = append(rows, row)
	}
	return rows
}

// tableKV reads a two-column table without header (field | value).
func tableKV(t *godog.Table) (map[string]string, error) {
	kv := make(map[string]string)
	if t == nil {
		return kv, nil
	}
	for _, r := range t.Rows {
		if len(r.Cells) != 2 {
			return nil, fmt.Errorf("expected a two-column table (field | value)")
		}
		kv[r.Cells[0].Value] = r.Cells[1].Value
	}
	return kv, nil
}

// yesNo parses the yes/no cells of the assertion tables.
func yesNo(raw string) (bool, error) {
	switch raw {
	case "yes":
		return true, nil
	case "no":
		return false, nil
	}
	return false, fmt.Errorf("expected yes or no, got %q", raw)
}

func yesNoText(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}
