//go:build acceptance

package steps

import (
	"context"
	"time"

	"github.com/cucumber/godog"

	"personal-finance/test/acceptance/world"
)

func registerContextSteps(sc *godog.ScenarioContext) {
	sc.Step(`^today is "(\d{4}-\d{2}-\d{2})"$`, setToday)
	sc.Step(`^it is now "(\d{4}-\d{2}-\d{2})"$`, setToday)
	sc.Step(`^I am on the free plan$`, iAmOnTheFreePlan)
}

func setToday(ctx context.Context, date string) error {
	day, err := time.Parse("2006-01-02", date)
	if err != nil {
		return err
	}
	world.From(ctx).Now = time.Date(day.Year(), day.Month(), day.Day(), 12, 0, 0, 0, time.UTC)
	return nil
}

func iAmOnTheFreePlan(ctx context.Context) error {
	world.From(ctx).Plan = "free"
	return nil
}
