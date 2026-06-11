package gateway

import (
	"bytes"
	"context"
	"os"
	"testing"

	"personal-finance/internal/domain"
	"personal-finance/pkg/log"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	// Prepare logs via the context-aware helpers, which require a global logger.
	log.Initialize()
	os.Exit(m.Run())
}

// minimalPDF is a tiny valid one-page PDF used as the base for encrypted fixtures.
const minimalPDF = `%PDF-1.4
1 0 obj
<< /Type /Catalog /Pages 2 0 R >>
endobj
2 0 obj
<< /Type /Pages /Kids [3 0 R] /Count 1 >>
endobj
3 0 obj
<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << >> >>
endobj
xref
0 4
0000000000 65535 f
0000000009 00000 n
0000000058 00000 n
0000000115 00000 n
trailer
<< /Size 4 /Root 1 0 R >>
startxref
203
%%EOF
`

func encryptPDF(t *testing.T, plain []byte, userPW, ownerPW string) []byte {
	t.Helper()
	conf := model.NewDefaultConfiguration()
	conf.UserPW = userPW
	conf.OwnerPW = ownerPW
	var buf bytes.Buffer
	require.NoError(t, api.Encrypt(bytes.NewReader(plain), &buf, conf))
	return buf.Bytes()
}

func isReadablePDF(t *testing.T, b []byte) bool {
	t.Helper()
	_, err := api.ReadContext(bytes.NewReader(b), model.NewDefaultConfiguration())
	return err == nil
}

func TestPDFCPUDecryptor_Prepare(t *testing.T) {
	plain := []byte(minimalPDF)
	d := NewPDFCPUDecryptor()
	ctx := context.Background()

	t.Run("not encrypted returns input unchanged", func(t *testing.T) {
		out, err := d.Prepare(ctx, plain, "")
		assert.NoError(t, err)
		assert.Equal(t, plain, out)
	})

	t.Run("owner-only is decrypted transparently", func(t *testing.T) {
		enc := encryptPDF(t, plain, "", "owneronly")
		out, err := d.Prepare(ctx, enc, "")
		assert.NoError(t, err)
		assert.True(t, isReadablePDF(t, out), "decrypted output should open with empty password")
	})

	t.Run("user password correct decrypts", func(t *testing.T) {
		enc := encryptPDF(t, plain, "s3cret", "owner")
		out, err := d.Prepare(ctx, enc, "s3cret")
		assert.NoError(t, err)
		assert.True(t, isReadablePDF(t, out), "decrypted output should open with empty password")
	})

	t.Run("user password missing returns password required", func(t *testing.T) {
		enc := encryptPDF(t, plain, "s3cret", "owner")
		_, err := d.Prepare(ctx, enc, "")
		assert.ErrorIs(t, err, domain.ErrStatementPasswordRequired)
	})

	t.Run("user password wrong returns wrong password", func(t *testing.T) {
		enc := encryptPDF(t, plain, "s3cret", "owner")
		_, err := d.Prepare(ctx, enc, "nope")
		assert.ErrorIs(t, err, domain.ErrStatementWrongPassword)
	})

	t.Run("unparseable bytes fail open (pass through to vision)", func(t *testing.T) {
		garbage := []byte("not a pdf at all")
		out, err := d.Prepare(ctx, garbage, "")
		assert.NoError(t, err)
		assert.Equal(t, garbage, out)
	})
}
