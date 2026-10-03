//go:build acceptance

package steps

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/google/uuid"

	"personal-finance/test/acceptance/world"
)

// refRe matches a movement reference in a step: an alias, one occurrence of a recurrent
// series, or one installment of a credit card purchase. It has ONE capture group (the
// whole reference), parsed by resolve.
const refRe = `("[^"]*"|the occurrence of "[^"]*" in "\d{4}-\d{2}"|installment \d+ of "[^"]*")`

var (
	aliasRef      = regexp.MustCompile(`^"([^"]*)"$`)
	occurrenceRef = regexp.MustCompile(`^the occurrence of "([^"]*)" in "(\d{4}-\d{2})"$`)
	installRef    = regexp.MustCompile(`^installment (\d+) of "([^"]*)"$`)
)

// target is a reference resolved against the current state of the API.
type target struct {
	// id goes in the URL: the movement id, or the recurrent id for a virtual occurrence.
	id   uuid.UUID
	view world.MovementView
	// series is the alias of the recurrent series this target is an occurrence of, if any.
	series string
}

// virtual is true for an occurrence that is only projected from its recurrent series.
func (t target) virtual() bool {
	return t.view.RecurrentID != nil && t.view.ID != nil && *t.view.ID == *t.view.RecurrentID
}

// resolve turns the text of a reference into the movement as the API shows it right now.
func resolve(w *world.World, raw string) (target, error) {
	if m := occurrenceRef.FindStringSubmatch(raw); m != nil {
		return resolveOccurrence(w, m[1], m[2])
	}

	if m := installRef.FindStringSubmatch(raw); m != nil {
		k, _ := strconv.Atoi(m[1])
		return resolveInstallment(w, m[2], k)
	}

	if m := aliasRef.FindStringSubmatch(raw); m != nil {
		ref, ok := w.Movements[m[1]]
		if !ok {
			return target{}, fmt.Errorf("unknown movement %q", m[1])
		}
		switch ref.Kind {
		case world.KindSingle:
			view, err := w.FindMovement(ref.ID)
			if err != nil {
				return target{}, err
			}
			if view == nil {
				return target{}, fmt.Errorf("movement %q does not exist (anymore)", m[1])
			}
			return target{id: ref.ID, view: *view}, nil
		case world.KindSeries:
			return target{}, fmt.Errorf("%q is a recurrent series: use `the occurrence of %q in \"yyyy-mm\"`", m[1], m[1])
		case world.KindInstallments:
			return target{}, fmt.Errorf("%q is an installment purchase: use `installment <k> of %q`", m[1], m[1])
		default:
			return target{}, fmt.Errorf("%q cannot be used as a movement here", m[1])
		}
	}

	return target{}, fmt.Errorf("cannot understand reference %q", raw)
}

func resolveOccurrence(w *world.World, alias, month string) (target, error) {
	ref, ok := w.Movements[alias]
	if !ok || ref.Kind != world.KindSeries {
		return target{}, fmt.Errorf("%q is not a recurrent series", alias)
	}

	view, err := w.Month(month)
	if err != nil {
		return target{}, err
	}

	var found []world.MovementView
	for _, m := range view.Movements {
		if (m.IsRecurrent || m.RecurrentID != nil) && ref.HasDescription(m.Description) {
			found = append(found, m)
		}
	}
	switch len(found) {
	case 0:
		return target{}, fmt.Errorf("there is no occurrence of %q in %s", alias, month)
	case 1:
		m := found[0]
		return target{id: *m.ID, view: m, series: alias}, nil
	default:
		return target{}, fmt.Errorf("found %d occurrences of %q in %s, expected one: %v", len(found), alias, month, found)
	}
}

func resolveInstallment(w *world.World, alias string, k int) (target, error) {
	ref, ok := w.Movements[alias]
	if !ok || ref.Kind != world.KindInstallments {
		return target{}, fmt.Errorf("%q is not an installment purchase", alias)
	}

	view, err := w.Period("2000-01-01", "2100-12-31")
	if err != nil {
		return target{}, err
	}
	for _, inv := range view.Invoices {
		for _, m := range inv.Movements {
			info := m.CreditCardInfo
			if info == nil || info.InstallmentGroupID == nil || *info.InstallmentGroupID != ref.InstallmentGroupID {
				continue
			}
			if info.InstallmentNumber != nil && *info.InstallmentNumber == k {
				return target{id: *m.ID, view: m}, nil
			}
		}
	}
	return target{}, fmt.Errorf("installment %d of %q does not exist (anymore)", k, alias)
}
