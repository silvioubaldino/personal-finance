//go:build acceptance

package acceptance

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/cucumber/godog"

	"personal-finance/test/acceptance/harness"
	"personal-finance/test/acceptance/steps"
)

var env *harness.Env

func TestMain(m *testing.M) {
	var err error
	if env, err = harness.Start(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, "acceptance: cannot start harness:", err)
		os.Exit(1)
	}
	code := m.Run()
	env.Close()
	os.Exit(code)
}

func TestFeatures(t *testing.T) {
	if err := os.MkdirAll("reports", 0o755); err != nil {
		t.Fatal(err)
	}

	suite := godog.TestSuite{
		Name:                "acceptance",
		ScenarioInitializer: steps.Initialize(env),
		Options: &godog.Options{
			Paths:       []string{"features"},
			Format:      "pretty,junit:reports/acceptance-junit.xml,cucumber:reports/acceptance-cucumber.json",
			Tags:        envOr("GODOG_TAGS", "~@known-bug && ~@wip"),
			Concurrency: envIntOr("GODOG_CONCURRENCY", 4),
			Randomize:   -1,   // random order: exposes hidden coupling between scenarios
			Strict:      true, // undefined or pending steps fail
			TestingT:    t,
		},
	}

	if suite.Run() != 0 {
		t.Fatal("acceptance scenarios failed")
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envIntOr(key string, fallback int) int {
	if v, err := strconv.Atoi(os.Getenv(key)); err == nil && v > 0 {
		return v
	}
	return fallback
}
