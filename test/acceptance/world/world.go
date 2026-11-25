//go:build acceptance

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

var DefaultToday = time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)

type World struct {
	UserID string
	Plan   string
	Now    time.Time

	Wallets    map[string]WalletRef
	Cards      map[string]CardRef
	Categories map[string]CategoryRef
	SubCats    map[string]SubCategoryRef
	Movements  map[string]MovementRef

	client *harness.Client
	db     *gorm.DB

	last     *harness.Response
	asserted bool
}

func New(env *harness.Env) *World {
	return &World{
		UserID:     uuid.NewString(),
		Plan:       "plus",
		Now:        DefaultToday,
		Wallets:    map[string]WalletRef{},
		Cards:      map[string]CardRef{},
		Categories: map[string]CategoryRef{},
		SubCats:    map[string]SubCategoryRef{},
		Movements:  map[string]MovementRef{},
		client:     env.Client,
		db:         env.DB,
	}
}

type worldKey struct{}

func Into(ctx context.Context, w *World) context.Context {
	return context.WithValue(ctx, worldKey{}, w)
}

func From(ctx context.Context) *World {
	w, _ := ctx.Value(worldKey{}).(*World)
	return w
}

func (w *World) DB() *gorm.DB { return w.db }

func (w *World) identity() harness.Identity {
	return harness.Identity{UserID: w.UserID, Plan: w.Plan, Now: w.Now}
}

func (w *World) Act(method, path string, body any) (harness.Response, error) {
	resp, err := w.client.Do(w.identity(), method, path, body)
	if err != nil {
		return resp, err
	}
	w.last = &resp
	w.asserted = false
	return resp, nil
}

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

func (w *World) Last() *harness.Response { return w.last }

func (w *World) MarkAsserted() { w.asserted = true }

func (w *World) CheckPending() error {
	if w.last == nil || w.asserted || w.last.OK() {
		return nil
	}
	resp := *w.last
	w.asserted = true
	return fmt.Errorf("previous step got an unexpected response: %s", resp)
}

func Cents(v float64) int64 { return int64(math.Round(v * 100)) }
