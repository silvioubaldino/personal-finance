//go:build acceptance

package harness

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal-finance/internal/domain"
	"personal-finance/internal/infrastructure/gateway"
)

func TestImportFixturesAreConsistent(t *testing.T) {
	entries, err := os.ReadDir(FixturesDir())
	require.NoError(t, err)

	for _, e := range entries {
		if !e.IsDir() || e.Name() == "generator" {
			continue
		}
		id := e.Name()
		dir := filepath.Join(FixturesDir(), id)

		t.Run(id, func(t *testing.T) {
			// Arrange
			pdf, err := os.ReadFile(filepath.Join(dir, "document.pdf"))
			require.NoError(t, err)
			raw, err := os.ReadFile(filepath.Join(dir, "extraction.json"))
			require.NoError(t, err)
			var reading domain.StatementExtractResult
			require.NoError(t, json.Unmarshal(raw, &reading))

			// Act
			gotID, err := FixtureID(pdf)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, id, gotID)
			assert.NotEmpty(t, reading.Movements)

			if reading.DocumentType == domain.DocInvoice {
				require.NotNil(t, reading.InvoiceMeta)
				require.NotNil(t, reading.InvoiceMeta.TotalAmount)
				var sum float64
				for _, m := range reading.Movements {
					if !domain.IsInvoicePaymentDescription(m.Description) {
						sum += m.Amount
					}
				}
				assert.InDelta(t, math.Abs(*reading.InvoiceMeta.TotalAmount), math.Abs(sum), 0.005,
					"declared total must match the items that belong to the invoice")
			}

			protected := filepath.Join(dir, "document-protegido.pdf")
			if enc, err := os.ReadFile(protected); err == nil {
				d := gateway.NewPDFCPUDecryptor()
				_, err := d.Prepare(context.Background(), enc, "")
				require.ErrorIs(t, err, domain.ErrStatementPasswordRequired)
				plain, err := d.Prepare(context.Background(), enc, "1234")
				require.NoError(t, err)
				gotID, err := FixtureID(plain)
				require.NoError(t, err)
				assert.Equal(t, id, gotID, "the protected copy must carry the same fixture id")
			}
		})
	}
}
