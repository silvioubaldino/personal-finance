package gateway

import (
	"bytes"
	"context"
	"errors"
	"strings"

	"personal-finance/internal/domain"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// PDFCPUDecryptor produces plaintext-ready PDF bytes so the vision gateway can
// parse them. It is stateless and therefore safe for concurrent use.
type PDFCPUDecryptor struct{}

func NewPDFCPUDecryptor() *PDFCPUDecryptor {
	return &PDFCPUDecryptor{}
}

// Prepare classifies the PDF and returns bytes ready for extraction:
//   - not encrypted             -> the input bytes unchanged
//   - encrypted, owner-only     -> decrypted transparently (empty user password)
//   - encrypted, needs password -> domain.ErrStatementPasswordRequired (none given)
//     or domain.ErrStatementWrongPassword (wrong one given)
func (d *PDFCPUDecryptor) Prepare(_ context.Context, fileBytes []byte, password string) ([]byte, error) {
	// Probe with an empty user password to classify the document. pdfcpu cannot
	// tell "no password supplied" apart from "wrong password" — both surface as
	// ErrWrongPassword — so we decide based on whether the caller gave one.
	ctx, err := api.ReadContext(bytes.NewReader(fileBytes), model.NewDefaultConfiguration())
	switch {
	case err == nil && (ctx.XRefTable == nil || ctx.Encrypt == nil):
		// Not encrypted: nothing to do.
		return fileBytes, nil

	case err == nil:
		// Encrypted but opens with an empty user password (owner-only): strip it.
		return decryptPDF(fileBytes, "")

	case isAuthFailure(err):
		// Encrypted and requires a user password to open.
		if password == "" {
			return nil, domain.ErrStatementPasswordRequired
		}
		out, derr := decryptPDF(fileBytes, password)
		if derr != nil {
			if isAuthFailure(derr) {
				return nil, domain.ErrStatementWrongPassword
			}
			return nil, domain.WrapInternalError(derr, "decrypt pdf")
		}
		return out, nil

	default:
		// Corrupt or otherwise unparseable PDF (not an auth problem).
		return nil, domain.WrapInvalidInput(err, "read pdf")
	}
}

func decryptPDF(fileBytes []byte, password string) ([]byte, error) {
	conf := model.NewDefaultConfiguration()
	conf.UserPW = password
	conf.OwnerPW = password
	var buf bytes.Buffer
	if err := api.Decrypt(bytes.NewReader(fileBytes), &buf, conf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func isAuthFailure(err error) bool {
	if errors.Is(err, pdfcpu.ErrWrongPassword) {
		return true
	}
	// Defensive fallback in case the sentinel changes across pdfcpu versions.
	return err != nil && strings.Contains(err.Error(), "please provide the correct password")
}
