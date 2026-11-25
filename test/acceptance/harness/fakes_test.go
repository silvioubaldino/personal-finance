//go:build acceptance

package harness

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal-finance/internal/domain"
)

func TestFixtureVisionGateway(t *testing.T) {
	tests := []struct {
		name     string
		register bool
		wantErr  bool
	}{
		{name: "returns the registered reading for the pdf subject", register: true},
		{name: "fails loudly for an unknown fixture", register: false, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			g := NewFixtureVisionGateway()
			reading := domain.StatementExtractResult{
				DocumentType: domain.DocInvoice,
				Movements:    []domain.ExtractedMovement{{Date: "2026-07-19", Description: "MP *CONECTCAR", Amount: -4.37}},
			}
			if tt.register {
				g.Register("inline-1", reading)
			}

			// Act
			got, err := g.ExtractMovements(context.Background(), MinimalPDF("inline-1"), "application/pdf", "invoice")

			// Assert
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, reading, got)
			got.Movements[0].Excluded = true
			again, _ := g.ExtractMovements(context.Background(), MinimalPDF("inline-1"), "application/pdf", "invoice")
			assert.False(t, again.Movements[0].Excluded, "readings must be cloned per call")
		})
	}
}
