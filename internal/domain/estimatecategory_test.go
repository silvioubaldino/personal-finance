package domain_test

import (
	"testing"

	"personal-finance/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestEstimateCategoriesList_GetEstimateByCategory(t *testing.T) {
	type (
		input struct {
			estimates domain.EstimateCategoriesList
		}
		expected struct {
			sums map[uuid.UUID]float64
		}
	)

	categoryA := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	categoryB := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	tests := map[string]struct {
		// input
		input input
		// expected
		expected expected
	}{
		"should accumulate two estimates of the same category": {
			input: input{
				estimates: domain.EstimateCategoriesList{
					{CategoryID: &categoryA, Amount: -600},
					{CategoryID: &categoryA, Amount: -400},
				},
			},
			expected: expected{
				sums: map[uuid.UUID]float64{categoryA: -1000},
			},
		},
		"should keep separate sums for different categories": {
			input: input{
				estimates: domain.EstimateCategoriesList{
					{CategoryID: &categoryA, Amount: -1000},
					{CategoryID: &categoryB, Amount: 5000},
				},
			},
			expected: expected{
				sums: map[uuid.UUID]float64{categoryA: -1000, categoryB: 5000},
			},
		},
		"should ignore estimates without a category": {
			input: input{
				estimates: domain.EstimateCategoriesList{
					{CategoryID: nil, Amount: -1000},
				},
			},
			expected: expected{
				sums: map[uuid.UUID]float64{},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Act
			sums := tc.input.estimates.GetEstimateByCategory()

			// Assert
			assert.Equal(t, tc.expected.sums, sums)
		})
	}
}
