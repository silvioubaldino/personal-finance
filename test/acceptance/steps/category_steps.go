//go:build acceptance

package steps

import (
	"context"
	"net/http"

	"github.com/cucumber/godog"

	"personal-finance/test/acceptance/world"
)

func registerCategorySteps(sc *godog.ScenarioContext) {
	sc.Step(`^an (expense|income) category "([^"]*)"$`, aCategory)
}

func aCategory(ctx context.Context, kind, name string) error {
	w := world.From(ctx)
	isIncome := kind == "income"

	resp, err := w.Act(http.MethodPost, "/v2/categories/", map[string]any{
		"description": name,
		"is_income":   isIncome,
	})
	if err != nil || !resp.OK() {
		return err
	}

	id, err := createdID(resp)
	if err != nil {
		return err
	}
	w.Categories[name] = world.CategoryRef{ID: id, IsIncome: isIncome}
	return nil
}
