//go:build acceptance

// Package world holds the per-scenario state: who the scenario acts as, what day it is,
// the aliases used in the Gherkin text and the last API response.
package world

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"personal-finance/test/acceptance/harness"
)

// DefaultToday is the "today" of a scenario that does not declare its own.
var DefaultToday = time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)

// World is the state of one scenario. It travels in the context.Context godog passes from
// step to step, so nothing is global and scenarios can run in parallel.
type World struct {
	UserID string
	Plan   string    // "plus" (default) | "free"
	Now    time.Time // becomes X-Test-Now on every request

	Wallets    map[string]WalletRef
	Cards      map[string]CardRef
	Categories map[string]CategoryRef
	Movements  map[string]MovementRef

	client *harness.Client
	db     *gorm.DB

	last     *harness.Response
	asserted bool
}

// New returns the world of a fresh scenario, acting as a brand new user.
func New(env *harness.Env) *World {
	return &World{
		UserID:     uuid.NewString(),
		Plan:       "plus",
		Now:        DefaultToday,
		Wallets:    map[string]WalletRef{},
		Cards:      map[string]CardRef{},
		Categories: map[string]CategoryRef{},
		Movements:  map[string]MovementRef{},
		client:     env.Client,
		db:         env.DB,
	}
}

type worldKey struct{}

// Into stores the world in the context.
func Into(ctx context.Context, w *World) context.Context {
	return context.WithValue(ctx, worldKey{}, w)
}

// From fetches the world of the running scenario.
func From(ctx context.Context) *World {
	w, _ := ctx.Value(worldKey{}).(*World)
	return w
}

// DB gives invariants read access to the database.
func (w *World) DB() *gorm.DB { return w.db }

func (w *World) identity() harness.Identity {
	return harness.Identity{UserID: w.UserID, Plan: w.Plan, Now: w.Now}
}

// Act performs an action step. It never fails on the HTTP status: the response is kept and
// the next step decides (see CheckPending).
func (w *World) Act(method, path string, body any) (harness.Response, error) {
	resp, err := w.client.Do(w.identity(), method, path, body)
	if err != nil {
		return resp, err
	}
	w.last = &resp
	w.asserted = false
	return resp, nil
}

// Query performs a read that backs an assertion. It does not touch the last action
// response, and a non-2xx answer is an error of the suite, not of the scenario.
func (w *World) Query(method, path string) (harness.Response, error) {
	resp, err := w.client.Do(w.identity(), method, path, nil)
	if err != nil {
		return resp, err
	}
	if !resp.OK() {
		return resp, fmt.Errorf("%s %s: %s", method, path, resp)
	}
	return resp, nil
}

// Last returns the response of the last action, or nil.
func (w *World) Last() *harness.Response { return w.last }

// MarkAsserted records that the last action response has been checked by a step.
func (w *World) MarkAsserted() { w.asserted = true }

// CheckPending fails when the last action got a non-2xx answer nobody asserted on, so a
// failure never passes silently. The failure is reported once.
func (w *World) CheckPending() error {
	if w.last == nil || w.asserted || w.last.OK() {
		return nil
	}
	resp := *w.last
	w.asserted = true
	return fmt.Errorf("previous step got an unexpected response: %s", resp)
}

// Cents converts a money amount to integer cents for comparisons.
func Cents(v float64) int64 { return int64(math.Round(v * 100)) }
