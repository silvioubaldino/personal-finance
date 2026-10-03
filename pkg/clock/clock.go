// Package clock gives business rules a "now" that can be pinned per context.
//
// Production code never calls WithNow, so Now behaves exactly like time.Now. The
// acceptance suite pins the clock per request so each scenario has its own "today".
package clock

import (
	"context"
	"time"
)

type nowKey struct{}

// WithNow pins the "now" observed by Now for this context.
func WithNow(ctx context.Context, now time.Time) context.Context {
	return context.WithValue(ctx, nowKey{}, now)
}

// Now returns the pinned time of the context, or time.Now() when there is no override.
func Now(ctx context.Context) time.Time {
	if now, ok := ctx.Value(nowKey{}).(time.Time); ok {
		return now
	}
	return time.Now()
}
