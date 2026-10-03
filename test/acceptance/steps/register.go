//go:build acceptance

// Package steps is the step library of the suite: the stable vocabulary scenarios are
// written with. Add new phrases to the group of the capability they belong to.
package steps

import (
	"context"
	"fmt"
	"regexp"

	"github.com/cucumber/godog"

	"personal-finance/test/acceptance/harness"
	"personal-finance/test/acceptance/world"
)

var rejectedStep = regexp.MustCompile(`^the operation is rejected as `)

// Initialize returns the godog ScenarioInitializer bound to the suite environment.
func Initialize(env *harness.Env) func(*godog.ScenarioContext) {
	return func(sc *godog.ScenarioContext) {
		sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
			return world.Into(ctx, world.New(env)), nil
		})

		// A non-2xx answer must be asserted by the very next step.
		sc.StepContext().Before(func(ctx context.Context, st *godog.Step) (context.Context, error) {
			if rejectedStep.MatchString(st.Text) {
				return ctx, nil
			}
			if err := world.From(ctx).CheckPending(); err != nil {
				return ctx, err
			}
			return ctx, nil
		})

		sc.After(func(ctx context.Context, _ *godog.Scenario, scenarioErr error) (context.Context, error) {
			w := world.From(ctx)
			if w == nil {
				return ctx, nil
			}
			if scenarioErr != nil {
				return ctx, nil
			}
			if err := w.CheckPending(); err != nil {
				return ctx, err
			}
			if err := checkInvariants(w); err != nil {
				return ctx, fmt.Errorf("global invariant violated: %w", err)
			}
			return ctx, nil
		})

		registerContextSteps(sc)
		registerWalletSteps(sc)
		registerCategorySteps(sc)
		registerMovementSteps(sc)
		registerAssertionSteps(sc)
	}
}
