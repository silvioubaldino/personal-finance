package domain

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
)

const (
	MaxStatementPages     = 20
	MaxStatementFileBytes = 10 * 1024 * 1024 // 10MB

	UncategorizedCategoryID       = "c1a2b3c4-d5e6-4f7a-8b9c-0d1e2f3a4b5c"
	UncategorizedIncomeCategoryID = "3fad33b7-48da-467f-be49-2e50b1226b82"
)

// Tipos de aviso não-fatal devolvidos pela extração (AYD-004 §Contrato).
const (
	WarningDocumentTypeMismatch   = "document_type_mismatch"
	WarningLowConfidence          = "low_confidence"
	WarningInvoicePaymentExcluded = "invoice_payment_excluded"
	WarningTotalAmountMismatch    = "total_amount_mismatch"
)

// Motivos pelos quais um item extraído não pertence à fatura (AYD-004
// §"Itens que não pertencem à fatura").
const (
	ExclusionReasonInvoicePayment = "invoice_payment"
)

// InvoiceTotalTolerance é a folga (em reais) ao comparar a soma dos itens com o
// total declarado na fatura — absorve arredondamento de ponto flutuante.
const InvoiceTotalTolerance = 0.01

// DocumentType diferencia o tipo de documento importado pelo usuário.
type DocumentType string

const (
	DocStatement DocumentType = "statement"
	DocInvoice   DocumentType = "invoice"
	DocUnknown   DocumentType = "unknown"
)

// ExtractWarning é um aviso não-fatal retornado na extração (ex.: divergência de tipo).
type ExtractWarning struct {
	Type     string `json:"type"`
	Expected string `json:"expected,omitempty"`
	Detected string `json:"detected,omitempty"`
}

// InvoiceMeta contém os metadados da fatura extraídos pelo modelo de visão.
type InvoiceMeta struct {
	ClosingDate *string  `json:"closing_date,omitempty"`
	DueDate     *string  `json:"due_date,omitempty"`
	TotalAmount *float64 `json:"total_amount,omitempty"`
}

type ExtractedMovement struct {
	Date              string      `json:"date"`
	Description       string      `json:"description"`
	Amount            float64     `json:"amount"`
	TypePayment       TypePayment `json:"type_payment,omitempty"`
	RecurrenceID      *uuid.UUID  `json:"recurrence_id,omitempty"`
	CategoryID        *uuid.UUID  `json:"category_id,omitempty"`
	SubCategoryID     *uuid.UUID  `json:"sub_category_id,omitempty"`
	InstallmentNumber *int        `json:"installment_number,omitempty"`
	TotalInstallments *int        `json:"total_installments,omitempty"`
	// Excluded marca um item que não pertence a esta fatura (ex.: o pagamento da
	// fatura anterior). O item permanece na resposta para a UI exibi-lo
	// desmarcado; o confirm-invoice o ignora.
	Excluded        bool   `json:"excluded,omitempty"`
	ExclusionReason string `json:"exclusion_reason,omitempty"`
}

type StatementExtractResult struct {
	DocumentType DocumentType        `json:"document_type,omitempty"`
	Confidence   float64             `json:"confidence,omitempty"`
	Warnings     []ExtractWarning    `json:"warnings,omitempty"`
	InvoiceMeta  *InvoiceMeta        `json:"invoice_meta,omitempty"`
	Movements    []ExtractedMovement `json:"movements"`
	Errors       []string            `json:"errors,omitempty"`
}

// InvoiceConfirmInput é o payload para confirmar a importação de itens de fatura.
type InvoiceConfirmInput struct {
	CreditCardID uuid.UUID           `json:"credit_card_id"`
	InvoiceID    *uuid.UUID          `json:"invoice_id,omitempty"`
	Movements    []ExtractedMovement `json:"movements"`
}
type StatementConfirmInput struct {
	Movements []ExtractedMovement `json:"movements"`
	WalletID  uuid.UUID           `json:"wallet_id"`
}

type StatementConfirmResult struct {
	Created int      `json:"created"`
	Skipped int      `json:"skipped"`
	Errors  []string `json:"errors,omitempty"`
}

type CategorySuggestion struct {
	Description   string     `json:"description"`
	CategoryID    *uuid.UUID `json:"category_id"`
	SubCategoryID *uuid.UUID `json:"subcategory_id"`
	Confidence    float64    `json:"confidence"`
	Source        string     `json:"source"` // "history" | "ai"
}

type StatementClassifyInput struct {
	Movements []ExtractedMovement `json:"movements"`
}

type StatementClassifyResult struct {
	Suggestions []CategorySuggestion `json:"suggestions"`
}

// --- Idempotency Hash ---

var nonAlphanumericRegex = regexp.MustCompile(`[^a-z0-9 ]`)

func NormalizeDescription(desc string) string {
	s := strings.ToLower(strings.TrimSpace(desc))
	s = strings.Join(strings.Fields(s), " ")
	s = nonAlphanumericRegex.ReplaceAllString(s, "")
	if len([]rune(s)) > 50 {
		s = string([]rune(s)[:50])
	}
	return s
}

// ComputeIdempotencyHash calcula o hash de idempotência para um movimento importado.
// scopeKey é o identificador do escopo (walletID ou creditCardID em formato string).
func ComputeIdempotencyHash(userID, scopeKey string, date time.Time, amount float64, description string) string {
	dateStr := date.Format("2006-01-02")
	normalizedDesc := NormalizeDescription(description)
	data := fmt.Sprintf("%s|%s|%s|%.2f|%s", userID, scopeKey, dateStr, amount, normalizedDesc)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash)
}

// IsDocumentTypeMismatch retorna true quando o cliente declarou um sourceType
// e a IA detectou um tipo diferente e confiante — sinaliza divergência de intenção.
func (r StatementExtractResult) IsDocumentTypeMismatch(sourceType string) bool {
	return sourceType != "" &&
		r.DocumentType != "" &&
		r.DocumentType != DocUnknown &&
		string(r.DocumentType) != sourceType
}

// IsLowConfidence retorna true quando o documento é desconhecido ou a confiança
// da IA está abaixo do limiar informado.
func (r StatementExtractResult) IsLowConfidence(threshold float64) bool {
	return r.DocumentType == DocUnknown ||
		(r.Confidence > 0 && r.Confidence < threshold)
}

// HasWarning retorna true quando já existe um aviso do tipo informado.
func (r StatementExtractResult) HasWarning(warningType string) bool {
	for _, w := range r.Warnings {
		if w.Type == warningType {
			return true
		}
	}
	return false
}

// invoicePaymentTokens são as palavras que abrem a linha de pagamento da fatura
// anterior nos formatos dos bancos brasileiros: "PAGAMENTO ON LINE" (Inter),
// "Pagamento recebido" (Nubank), "PAGAMENTO EFETUADO" (Itaú), "PAGTO. POR DEB.
// CONTA" (Bradesco).
var invoicePaymentTokens = map[string]bool{
	"PAGAMENTO":  true,
	"PAGAMENTOS": true,
	"PAGTO":      true,
	"PGTO":       true,
}

// IsInvoicePaymentDescription informa se a descrição é a linha de pagamento da
// fatura **anterior** — que não é despesa desta fatura e é modelada fora dela,
// como um Movement de TypePaymentInvoicePayment (AYD-004 §"Itens que não
// pertencem à fatura").
//
// Casa apenas quando a **primeira palavra** é um token de pagamento, e não em
// qualquer ocorrência no meio do texto: estabelecimentos reais colidiriam
// ("PAGUE MENOS", "PAGSEGURO", "PAGBANK") e uma linha de pagamento sempre abre
// com o termo. Ainda assim é heurística — por isso o item é **marcado, nunca
// removido**, e a UI deixa o usuário reincluí-lo num falso-positivo.
func IsInvoicePaymentDescription(description string) bool {
	words := strings.FieldsFunc(description, func(r rune) bool {
		return !unicode.IsLetter(r)
	})
	if len(words) == 0 {
		return false
	}
	return invoicePaymentTokens[strings.ToUpper(words[0])]
}

// --- Errors ---

var (
	ErrStatementNotAStatement    = New("the uploaded file does not appear to be a bank statement")
	ErrStatementTooManyPages     = fmt.Errorf("PDF exceeds maximum of %d pages", MaxStatementPages)
	ErrStatementFileTooLarge     = fmt.Errorf("file exceeds maximum size of %dMB", MaxStatementFileBytes/(1024*1024))
	ErrStatementExtractionFailed = New("failed to extract movements from the statement")
	ErrStatementPasswordRequired = New("statement pdf is password protected")
	ErrStatementWrongPassword    = New("incorrect password for statement pdf")
)
