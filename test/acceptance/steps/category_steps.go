//go:build acceptance

package steps

import (
	"context"
	"fmt"
	"net/http"

	"github.com/cucumber/godog"

	"personal-finance/test/acceptance/world"
)

func registerCategorySteps(sc *godog.ScenarioContext) {
	sc.Step(`^an (expense|income) category "([^"]*)"$`, aCategory)
	sc.Step(`^a subcategory "([^"]*)" of "([^"]*)"$`, aSubcategory)
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

func aSubcategory(ctx context.Context, name, category string) error {
	w := world.From(ctx)
	parent, ok := w.Categories[category]
	if !ok {
		return fmt.Errorf("unknown category %q", category)
	}

	resp, err := w.Act(http.MethodPost, "/v2/subcategories/", map[string]any{
		"description": name,
		"category_id": parent.ID,
	})
	if err != nil || !resp.OK() {
		return err
	}

	id, err := createdID(resp)
	if err != nil {
		return err
	}
	w.SubCats[name] = world.SubCategoryRef{ID: id, CategoryID: parent.ID}
	return nil
}
