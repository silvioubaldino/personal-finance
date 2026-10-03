//go:build acceptance

package steps

import (
	"fmt"
	"strconv"
	"time"

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
