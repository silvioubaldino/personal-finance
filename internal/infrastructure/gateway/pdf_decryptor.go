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
//   - not encrypted             -> the input bytes, header-normalized
//   - encrypted, owner-only     -> decrypted transparently (empty user password)
//   - encrypted, needs password -> domain.ErrStatementPasswordRequired (none given)
//     or domain.ErrStatementWrongPassword (wrong one given)
//
// Every path first trims filler bytes preceding the %PDF- signature, so the
// bytes handed to the vision model always start at the header.
//
// It is intentionally fail-open: the only hard failures are the genuine
// "needs a password" cases. For any other read/decrypt problem (malformed PDF,
// or an encryption variant pdfcpu's stricter parser rejects) it returns the
// original bytes so the vision model — which is far more lenient and handles a
// wider range of bank PDFs — gets a chance to read them. pdfcpu may only help
// (decrypt), never block.
func (d *PDFCPUDecryptor) Prepare(ctx context.Context, fileBytes []byte, password string) ([]byte, error) {
	if trimmed := trimBeforePDFHeader(fileBytes); len(trimmed) != len(fileBytes) {
		log.WarnContext(ctx, "statement pdf: filler bytes before the %PDF- header, trimming",
			log.Int("trimmed_bytes", len(fileBytes)-len(trimmed)))
		fileBytes = trimmed
	}

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

// pdfHeaderMarker is the signature a PDF file must start with.
var pdfHeaderMarker = []byte("%PDF-")

// trimBeforePDFHeader drops any bytes preceding the %PDF- signature. Some banks
// export faturas with a block of filler bytes in front of it; pdfcpu reads such
// a file fine (it locates the header and offsets from there), but Vertex is
// stricter and rejects it with "The document has no pages". Per ISO 32000-1 the
// xref offsets of a shifted file are already relative to the header, so cutting
// the prefix yields the same document in a form strict readers accept.
//
// Returns the input unchanged when there is nothing to trim, or when no header
// is present at all — failing open like the rest of Prepare.
func trimBeforePDFHeader(fileBytes []byte) []byte {
	i := bytes.Index(fileBytes, pdfHeaderMarker)
	if i <= 0 {
		return fileBytes
	}
	return fileBytes[i:]
}
