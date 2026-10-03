package clock

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNow(t *testing.T) {
	pinned := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name  string
		ctx   func() context.Context
		check func(t *testing.T, got time.Time)
	}{
		{
			name: "without override returns the real time",
			ctx:  context.Background,
			check: func(t *testing.T, got time.Time) {
				assert.WithinDuration(t, time.Now(), got, time.Second)
			},
		},
		{
			name: "with override returns the pinned time",
			ctx: func() context.Context {
				return WithNow(context.Background(), pinned)
			},
			check: func(t *testing.T, got time.Time) {
				assert.True(t, pinned.Equal(got))
			},
		},
		{
			name: "child context inherits the pinned time",
			ctx: func() context.Context {
				type k struct{}
				return context.WithValue(WithNow(context.Background(), pinned), k{}, "v")
			},
			check: func(t *testing.T, got time.Time) {
				assert.True(t, pinned.Equal(got))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.check(t, Now(tt.ctx()))
		})
	}
}
