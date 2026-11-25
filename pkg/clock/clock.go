package clock

import (
	"context"
	"time"
)

type nowKey struct{}

func WithNow(ctx context.Context, now time.Time) context.Context {
	return context.WithValue(ctx, nowKey{}, now)
}

func Now(ctx context.Context) time.Time {
	if now, ok := ctx.Value(nowKey{}).(time.Time); ok {
		return now
	}
	return time.Now()
}
