//go:build acceptance

package harness

import (
	"net/http"
	"time"

	"personal-finance/pkg/clock"
)

// withClock turns the X-Test-Now header into a per-request clock, so every scenario has
// its own "today" even when scenarios run in parallel.
func withClock(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if raw := r.Header.Get(HeaderNow); raw != "" {
			if now, err := time.Parse(time.RFC3339, raw); err == nil {
				r = r.WithContext(clock.WithNow(r.Context(), now))
			}
		}
		next.ServeHTTP(w, r)
	})
}
