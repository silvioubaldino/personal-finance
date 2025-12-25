package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSetMonthYearClamped(t *testing.T) {
	loc := time.UTC

	tests := []struct {
		name     string
		date     time.Time
		month    time.Month
		year     int
		expected time.Time
	}{
		{
			name:     "normal day in target month",
			date:     time.Date(2025, 1, 15, 10, 30, 0, 0, loc),
			month:    time.March,
			year:     2025,
			expected: time.Date(2025, 3, 15, 10, 30, 0, 0, loc),
		},
		{
			name:     "day 31 to February (non-leap year) clamps to 28",
			date:     time.Date(2025, 1, 31, 10, 30, 0, 0, loc),
			month:    time.February,
			year:     2025,
			expected: time.Date(2025, 2, 28, 10, 30, 0, 0, loc),
		},
		{
			name:     "day 31 to February (leap year) clamps to 29",
			date:     time.Date(2024, 1, 31, 10, 30, 0, 0, loc),
			month:    time.February,
			year:     2024,
			expected: time.Date(2024, 2, 29, 10, 30, 0, 0, loc),
		},
		{
			name:     "day 31 to April clamps to 30",
			date:     time.Date(2025, 1, 31, 10, 30, 0, 0, loc),
			month:    time.April,
			year:     2025,
			expected: time.Date(2025, 4, 30, 10, 30, 0, 0, loc),
		},
		{
			name:     "day 30 to February clamps to 28",
			date:     time.Date(2025, 1, 30, 10, 30, 0, 0, loc),
			month:    time.February,
			year:     2025,
			expected: time.Date(2025, 2, 28, 10, 30, 0, 0, loc),
		},
		{
			name:     "day 29 to February (non-leap) clamps to 28",
			date:     time.Date(2025, 1, 29, 10, 30, 0, 0, loc),
			month:    time.February,
			year:     2025,
			expected: time.Date(2025, 2, 28, 10, 30, 0, 0, loc),
		},
		{
			name:     "day 29 to February (leap year) stays 29",
			date:     time.Date(2024, 1, 29, 10, 30, 0, 0, loc),
			month:    time.February,
			year:     2024,
			expected: time.Date(2024, 2, 29, 10, 30, 0, 0, loc),
		},
		{
			name:     "preserves hour minute second nanosecond",
			date:     time.Date(2025, 1, 15, 14, 45, 30, 123456789, loc),
			month:    time.June,
			year:     2025,
			expected: time.Date(2025, 6, 15, 14, 45, 30, 123456789, loc),
		},
		{
			name:     "month overflow (13) wraps to next year",
			date:     time.Date(2025, 1, 15, 10, 30, 0, 0, loc),
			month:    13,
			year:     2025,
			expected: time.Date(2026, 1, 15, 10, 30, 0, 0, loc),
		},
		{
			name:     "month underflow (0) wraps to previous year December",
			date:     time.Date(2025, 1, 15, 10, 30, 0, 0, loc),
			month:    0,
			year:     2025,
			expected: time.Date(2024, 12, 15, 10, 30, 0, 0, loc),
		},
		{
			name:     "month underflow (-1) wraps to previous year November",
			date:     time.Date(2025, 1, 15, 10, 30, 0, 0, loc),
			month:    -1,
			year:     2025,
			expected: time.Date(2024, 11, 15, 10, 30, 0, 0, loc),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SetMonthYearClamped(tt.date, tt.month, tt.year)
			assert.Equal(t, tt.expected, result)
		})
	}
}
