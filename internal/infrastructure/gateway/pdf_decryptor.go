package gateway

import (
	"bytes"
	"context"
	"errors"
	"strings"

	"personal-finance/internal/domain"
	"personal-finance/pkg/log"

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
//
// It is intentionally fail-open: the only hard failures are the genuine
// "needs a password" cases. For any other read/decrypt problem (malformed PDF,
// or an encryption variant pdfcpu's stricter parser rejects) it returns the
// original bytes so the vision model — which is far more lenient and handles a
// wider range of bank PDFs — gets a chance to read them. pdfcpu may only help
// (decrypt), never block.
func (d *PDFCPUDecryptor) Prepare(ctx context.Context, fileBytes []byte, password string) ([]byte, error) {
	// Probe with an empty user password to classify the document. pdfcpu cannot
	// tell "no password supplied" apart from "wrong password" — both surface as
	// ErrWrongPassword — so we decide based on whether the caller gave one.
	pdfCtx, err := api.ReadContext(bytes.NewReader(fileBytes), model.NewDefaultConfiguration())
	switch {
	case err == nil && (pdfCtx.XRefTable == nil || pdfCtx.Encrypt == nil):
		// Not encrypted: nothing to do.
		return fileBytes, nil

	case err == nil:
		// Encrypted but opens with an empty user password (owner-only): strip it.
		// If stripping fails, pass the original through — the vision model reads
		// owner-only PDFs natively.
		out, derr := decryptPDF(fileBytes, "")
		if derr != nil {
			log.WarnContext(ctx, "statement pdf: owner-only decrypt failed, passing original through to vision", log.Err(derr))
			return fileBytes, nil
		}
		return out, nil

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
		// pdfcpu could not parse the file (malformed, or an unsupported encryption
		// variant). Fail open: hand the original bytes to the vision model rather
		// than rejecting a PDF it might read fine.
		log.WarnContext(ctx, "statement pdf: pdfcpu could not parse, passing original through to vision", log.Err(err))
		return fileBytes, nil
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
