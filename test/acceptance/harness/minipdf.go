//go:build acceptance

package harness

import (
	"bytes"
	"fmt"
	"strings"
)

// MinimalPDF gera um PDF válido de uma página, sem conteúdo, cujo /Subject carrega o
// ID de fixture. É o "documento" das leituras inline: o dublê de visão não lê o
// conteúdo, só o ID.
func MinimalPDF(fixtureID string) []byte {
	subject := strings.NewReplacer(`\`, `\\`, `(`, `\(`, `)`, `\)`).Replace(FixturePrefix + fixtureID)
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 200 200] >>",
		fmt.Sprintf("<< /Subject (%s) /Producer (personal-finance acceptance) >>", subject),
	}

	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects))
	for i, obj := range objects {
		offsets[i] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	xref := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for _, off := range offsets {
		fmt.Fprintf(&buf, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R /Info 4 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	return buf.Bytes()
}
