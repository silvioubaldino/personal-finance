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
)

type ExtractedMovement struct {
	Date         string     `json:"date"`
	Description  string     `json:"description"`
	Amount       float64    `json:"amount"`
	RecurrenceID *uuid.UUID `json:"recurrence_id,omitempty"`
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
	ErrStatementNotAStatement = New("the uploaded file does not appear to be a bank statement")
	ErrStatementTooManyPages  = fmt.Errorf("PDF exceeds maximum of %d pages", MaxStatementPages)
	ErrStatementFileTooLarge  = fmt.Errorf("file exceeds maximum size of %dMB", MaxStatementFileBytes/(1024*1024))
	ErrStatementExtractionFailed = New("failed to extract movements from the statement")
)
