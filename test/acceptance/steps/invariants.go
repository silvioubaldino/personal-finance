//go:build acceptance

package steps

import (
	"errors"

	"personal-finance/test/acceptance/world"
)

// invariant is a rule that must hold for the data of every scenario, whatever the
// scenario did. It runs in the After hook, scoped to the scenario's user_id.
type invariant struct {
	name  string
	check func(w *world.World) error
}

var invariants = []invariant{}

func checkInvariants(w *world.World) error {
	var errs []error
	for _, inv := range invariants {
		if err := inv.check(w); err != nil {
			errs = append(errs, errors.New(inv.name+": "+err.Error()))
		}
	}
	return errors.Join(errs...)
}
