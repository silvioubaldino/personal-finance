package tests

import (
	"context"
	"testing"

	"personal-finance/tests/steps"
	"personal-finance/tests/suite"

	"github.com/cucumber/godog"
)

func TestFeatures(t *testing.T) {
	testSuite := &suite.TestSuite{}

	opts := godog.Options{
		Format:   "pretty",
		Paths:    []string{"features"},
		TestingT: t,
	}

	godog.TestSuite{
		Name:                 "Integration Tests",
		TestSuiteInitializer: testSuite.InitializeTestSuite,
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			ctx.Before(func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
				testSuite.BeforeScenario(sc)
				return ctx, nil
			})

			recurrentMovementSteps := steps.NewRecurrentMovementSteps(testSuite)
			recurrentMovementSteps.RegisterSteps(ctx)
		},
		Options: &opts,
	}.Run()
}
