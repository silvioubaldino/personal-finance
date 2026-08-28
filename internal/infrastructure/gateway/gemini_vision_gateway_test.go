package gateway

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFlexibleFloat64_UnmarshalJSON(t *testing.T) {
	type (
		input struct {
			data string
		}
		expected struct {
			output  flexibleFloat64
			wantErr bool
		}
	)

	tests := map[string]struct {
		// input
		input input
		// expected
		expected expected
	}{
		"should parse a plain JSON number": {
			input:    input{data: `-3450.27`},
			expected: expected{output: -3450.27, wantErr: false},
		},
		"should parse a number quoted as a string": {
			input:    input{data: `"6035.06"`},
			expected: expected{output: 6035.06, wantErr: false},
		},
		"should parse a string using comma as the decimal separator": {
			input:    input{data: `"6035,06"`},
			expected: expected{output: 6035.06, wantErr: false},
		},
		"should return error when value is neither a number nor a string": {
			input:    input{data: `{"nested":true}`},
			expected: expected{output: 0, wantErr: true},
		},
		"should return error when the string is not a valid number": {
			input:    input{data: `"not-a-number"`},
			expected: expected{output: 0, wantErr: true},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Arrange
			var output flexibleFloat64

			// Act
			err := output.UnmarshalJSON([]byte(tc.input.data))

			// Assert
			assert.Equal(t, tc.expected.output, output)
			assert.Equal(t, tc.expected.wantErr, err != nil)
		})
	}
}
