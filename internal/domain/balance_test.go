package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPeriod_ValidateAt(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	day := func(d int) time.Time { return time.Date(2026, 3, d, 0, 0, 0, 0, time.UTC) }

	tests := []struct {
		name     string
		period   Period
		wantFrom time.Time
		wantTo   time.Time
		wantErr  string
	}{
		{
			name:     "valid period is kept",
			period:   Period{From: day(1), To: day(31)},
			wantFrom: day(1),
			wantTo:   day(31),
		},
		{
			name:     "unset bounds default to now",
			period:   Period{From: day(1)},
			wantFrom: day(1),
			wantTo:   now,
		},
		{
			name:    "equal bounds are rejected",
			period:  Period{},
			wantErr: "date must be informed",
		},
		{
			name:    "from after to is rejected",
			period:  Period{From: day(20), To: day(10)},
			wantErr: "'from' must be before 'to'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.period

			err := p.ValidateAt(now)

			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				return
			}
			assert.NoError(t, err)
			assert.True(t, tt.wantFrom.Equal(p.From))
			assert.True(t, tt.wantTo.Equal(p.To))
		})
	}
}
