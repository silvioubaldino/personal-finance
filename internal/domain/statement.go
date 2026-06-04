package domain

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	MaxStatementPages     = 20
	MaxStatementFileBytes = 10 * 1024 * 1024 // 10MB

	UncategorizedCategoryID = "c1a2b3c4-d5e6-4f7a-8b9c-0d1e2f3a4b5c"
)

type ExtractedMovement struct {
	Date          string      `json:"date"`
	Description   string      `json:"description"`
	Amount        float64     `json:"amount"`
	TypePayment   TypePayment `json:"type_payment,omitempty"`
	RecurrenceID  *uuid.UUID  `json:"recurrence_id,omitempty"`
	CategoryID    *uuid.UUID  `json:"category_id,omitempty"`
	SubCategoryID *uuid.UUID  `json:"sub_category_id,omitempty"`
}

type StatementExtractResult struct {
	Movements []ExtractedMovement `json:"movements"`
	Errors    []string            `json:"errors,omitempty"`
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

func ComputeIdempotencyHash(userID string, walletID uuid.UUID, date time.Time, amount float64, description string) string {
	dateStr := date.Format("2006-01-02")
	normalizedDesc := NormalizeDescription(description)
	data := fmt.Sprintf("%s|%s|%s|%.2f|%s", userID, walletID.String(), dateStr, amount, normalizedDesc)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash)
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
