package domain

import (
	"crypto/sha256"
	"fmt"
	"math"
	"regexp"
	"strconv"
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
	// WarningFutureInstallmentExcluded sinaliza que ≥ 1 item foi marcado como
	// parcela de competência posterior ao fechamento da fatura importada.
	WarningFutureInstallmentExcluded = "future_installment_excluded"
	// WarningInstallmentMatchFound sinaliza que ≥ 1 item corresponde a uma série
	// de parcelas já registrada no app.
	WarningInstallmentMatchFound = "installment_match_found"
)

// Motivos pelos quais um item extraído não pertence à fatura (AYD-004
// §"Itens que não pertencem à fatura").
const (
	ExclusionReasonInvoicePayment    = "invoice_payment"
	ExclusionReasonFutureInstallment = "future_installment"
)

// InvoiceTotalTolerance é a folga (em reais) ao comparar a soma dos itens com o
// total declarado na fatura — absorve arredondamento de ponto flutuante.
const InvoiceTotalTolerance = 0.01

// InstallmentAmountTolerance é a folga (em reais) ao comparar o valor de um item
// extraído com o da parcela já registrada no app (AYD-004 §"Assinatura do match").
// O banco distribui o arredondamento do parcelamento entre as parcelas, então
// parcelas da mesma compra diferem por centavos — tipicamente R$ 0,01, e no
// máximo alguns centavos quando o resto é espalhado por poucas parcelas. R$ 0,05
// cobre essa variação sem afrouxar a assinatura a ponto de colar duas compras
// distintas de valor parecido: o total de parcelas e a raiz da descrição
// continuam tendo de bater.
const InstallmentAmountTolerance = 0.05

// Níveis de confiança do match de parcela (AYD-004 §"Assinatura do match").
// Alta vem pré-vinculada; média vira sugestão; nenhuma cai no caminho de criação.
const (
	InstallmentMatchConfidenceHigh   = 0.95
	InstallmentMatchConfidenceMedium = 0.7
	InstallmentMatchConfidenceNone   = 0.0
)

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

// InstallmentMatch é a série de parcelas já registrada no app que corresponde ao
// item extraído. É **sugestão** do servidor; quem decide o vínculo é o cliente,
// devolvendo InstallmentGroupID no confirm-invoice (AYD-004 §"Parcelas já
// registradas no app").
type InstallmentMatch struct {
	InstallmentGroupID uuid.UUID `json:"installment_group_id"`
	MovementID         uuid.UUID `json:"movement_id"`
	Description        string    `json:"description"`
	InstallmentNumber  int       `json:"installment_number"`
	TotalInstallments  int       `json:"total_installments"`
	Amount             float64   `json:"amount"`
	Confidence         float64   `json:"confidence"`
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
	// InstallmentMatch é a sugestão do servidor; InstallmentGroupID é a decisão
	// que o cliente devolve no confirm-invoice, efetivando o vínculo. São campos
	// separados de propósito: a UI pode recusar a sugestão sem perder a evidência
	// que a motivou.
	InstallmentMatch   *InstallmentMatch `json:"installment_match,omitempty"`
	InstallmentGroupID *uuid.UUID        `json:"installment_group_id,omitempty"`
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

// installmentSuffixRegex casa o sufixo de parcela nos formatos que os bancos
// brasileiros usam na descrição do item: "03/12", "3/12", "PARC 3/12",
// "PARC. 03/12", "PARCELA 03/12", "PARCELA 03 DE 12" — com ou sem zero à
// esquerda, no fim da string ou cercado de espaços.
var installmentSuffixRegex = regexp.MustCompile(`(?i)\s*(?:\bparc(?:ela)?\b\.?\s*)?\b(\d{1,3})\s*(?:/|\bde\b)\s*(\d{1,3})\b`)

// StripInstallmentSuffix remove o sufixo de parcela da descrição, devolvendo a
// **raiz** — a parte que se mantém estável entre competências. É o insumo do
// matcher de série (AYD-004 §"Assinatura do match"): NormalizeDescription sozinha
// produziria "mercado livre parcela 0312", com o sufixo embutido, que muda todo
// mês e nunca casaria a parcela 3/12 com a 4/12 da mesma compra.
//
// Só remove quando os dois números formam uma parcela plausível (1 ≤ n ≤ total e
// total ≥ 2). Isso evita o falso-positivo de um estabelecimento cujo nome carrega
// números — "POSTO 24/7" tem n > total e permanece intacto.
func StripInstallmentSuffix(desc string) string {
	matches := installmentSuffixRegex.FindAllStringSubmatchIndex(desc, -1)
	if len(matches) == 0 {
		return desc
	}

	var (
		builder  strings.Builder
		lastEnd  int
		stripped bool
	)
	for _, m := range matches {
		number, errNumber := strconv.Atoi(desc[m[2]:m[3]])
		total, errTotal := strconv.Atoi(desc[m[4]:m[5]])
		if errNumber != nil || errTotal != nil || total < 2 || number < 1 || number > total {
			continue
		}
		builder.WriteString(desc[lastEnd:m[0]])
		lastEnd = m[1]
		stripped = true
	}
	if !stripped {
		return desc
	}
	builder.WriteString(desc[lastEnd:])

	return strings.Join(strings.Fields(builder.String()), " ")
}

// InstallmentMatchConfidence calcula a confiança de que o item extraído da fatura
// é a parcela desta competência de uma série já registrada no app
// (AYD-004 §"Assinatura do match"):
//
//	alta   — mesmo total de parcelas, valor dentro da tolerância e raiz da descrição compatível;
//	média  — total de parcelas e valor batem, mas a raiz divergiu;
//	nenhuma— o resto.
//
// A igualdade do número da parcela não é parte da assinatura: ela identifica
// *qual* parcela da série corresponde a esta competência — sem ela o vínculo
// atualizaria o valor da parcela errada.
func InstallmentMatchConfidence(extracted ExtractedMovement, candidate Movement) float64 {
	if extracted.InstallmentNumber == nil || extracted.TotalInstallments == nil {
		return InstallmentMatchConfidenceNone
	}
	if !candidate.IsInstallmentMovement() || candidate.CreditCardInfo.InstallmentGroupID == nil {
		return InstallmentMatchConfidenceNone
	}
	if *candidate.CreditCardInfo.TotalInstallments != *extracted.TotalInstallments ||
		*candidate.CreditCardInfo.InstallmentNumber != *extracted.InstallmentNumber {
		return InstallmentMatchConfidenceNone
	}
	if math.Abs(candidate.Amount-extracted.Amount) > InstallmentAmountTolerance {
		return InstallmentMatchConfidenceNone
	}

	extractedRoot := NormalizeDescription(StripInstallmentSuffix(extracted.Description))
	candidateRoot := NormalizeDescription(StripInstallmentSuffix(candidate.Description))
	if extractedRoot != "" && extractedRoot == candidateRoot {
		return InstallmentMatchConfidenceHigh
	}

	return InstallmentMatchConfidenceMedium
}

// ResolveTargetInvoicePeriodEnd descobre o fim do período da fatura alvo de um
// conjunto de itens, derivando-o do dia de fechamento do cartão (AYD-004,
// decisão 7). Item com data posterior a esse limite é de competência futura.
//
// A fatura alvo é a **moda** dos períodos dos itens: cada data cai num período
// derivado do fechamento, e o período com mais itens é o da fatura importada.
// Usar a menor ou a maior data seria frágil — vários bancos datam a parcela pela
// data da compra original (jogando o mínimo meses atrás) e a própria fatura lista
// parcelas de competência futura (jogando o máximo meses à frente). Empate
// resolve pelo período mais antigo, para não empurrar a fatura alvo para frente e
// marcar item legítimo como futuro.
func ResolveTargetInvoicePeriodEnd(creditCard CreditCard, dates []time.Time) (time.Time, bool) {
	if creditCard.ClosingDay < 1 || creditCard.ClosingDay > 31 || len(dates) == 0 {
		return time.Time{}, false
	}

	counts := make(map[time.Time]int, len(dates))
	for _, date := range dates {
		counts[BuildInvoice(creditCard, date).PeriodEnd]++
	}

	var (
		target time.Time
		found  bool
	)
	for periodEnd, count := range counts {
		if !found || count > counts[target] || (count == counts[target] && periodEnd.Before(target)) {
			target, found = periodEnd, true
		}
	}

	return target, found
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
