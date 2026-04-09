package usecase

import (
	"context"
	"fmt"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/plataform/authentication"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const ClassificationConfidenceThreshold = 0.6

// --- Interfaces ---

type StatementVisionGateway interface {
	ExtractMovements(ctx context.Context, fileBytes []byte, mimeType string) (domain.StatementExtractResult, error)
}

type StatementClassificationGateway interface {
	ClassifyMovements(ctx context.Context, movements []domain.ExtractedMovement, categories []domain.Category) ([]domain.CategorySuggestion, error)
}

type StatementMovementRepository interface {
	Add(ctx context.Context, tx *gorm.DB, movement domain.Movement) (domain.Movement, error)
	FindExistingHashes(ctx context.Context, userID string, hashes []string) (map[string]bool, error)
	FindByRecurrentIDAndMonth(ctx context.Context, recurrentID uuid.UUID, month time.Time) (*domain.Movement, error)
	UpdateStatementLink(ctx context.Context, tx *gorm.DB, id uuid.UUID, movement domain.Movement) (domain.Movement, error)
	FindRecentCategorizedByNormalizedDescription(ctx context.Context, normalizedDesc string) (*uuid.UUID, *uuid.UUID, error)
}

type StatementCategoryRepository interface {
	FindAll(ctx context.Context) ([]domain.Category, error)
}

// --- Use Case ---

type StatementUseCase struct {
	visionGateway         StatementVisionGateway
	classificationGateway StatementClassificationGateway
	movementRepo          StatementMovementRepository
	categoryRepo          StatementCategoryRepository
	limitsValidator       PlanLimitsValidatorInterface
}

func NewStatementUseCase(
	visionGateway StatementVisionGateway,
	classificationGateway StatementClassificationGateway,
	movementRepo StatementMovementRepository,
	categoryRepo StatementCategoryRepository,
	limitsValidator PlanLimitsValidatorInterface,
) *StatementUseCase {
	return &StatementUseCase{
		visionGateway:         visionGateway,
		classificationGateway: classificationGateway,
		movementRepo:          movementRepo,
		categoryRepo:          categoryRepo,
		limitsValidator:       limitsValidator,
	}
}

// Extract processes a file (PDF or image) and returns extracted movements without saving.
func (u *StatementUseCase) Extract(ctx context.Context, fileBytes []byte, mimeType string) (domain.StatementExtractResult, error) {
	userID := authentication.UserIDFromContext(ctx)
	if userID == "" {
		return domain.StatementExtractResult{}, domain.ErrUnauthorized
	}

	// Validate file size
	if len(fileBytes) > domain.MaxStatementFileBytes {
		return domain.StatementExtractResult{}, domain.ErrStatementFileTooLarge
	}

	// Validate mime type
	if !isAllowedMimeType(mimeType) {
		return domain.StatementExtractResult{}, domain.WrapInvalidInput(
			domain.New("unsupported file type: must be PDF, JPEG, or PNG"),
			"validate file type",
		)
	}

	// Call Gemini Vision
	result, err := u.visionGateway.ExtractMovements(ctx, fileBytes, mimeType)
	if err != nil {
		return domain.StatementExtractResult{}, fmt.Errorf("extract movements: %w", err)
	}

	return result, nil
}

func (u *StatementUseCase) Classify(ctx context.Context, input domain.StatementClassifyInput) (domain.StatementClassifyResult, error) {
	userID := authentication.UserIDFromContext(ctx)
	if userID == "" {
		return domain.StatementClassifyResult{}, domain.ErrUnauthorized
	}

	if len(input.Movements) == 0 {
		return domain.StatementClassifyResult{}, domain.WrapInvalidInput(
			domain.New("no movements to classify"),
			"validate input",
		)
	}

	categories, err := u.categoryRepo.FindAll(ctx)
	if err != nil {
		return domain.StatementClassifyResult{}, fmt.Errorf("fetch categories: %w", err)
	}

	suggestions := make([]domain.CategorySuggestion, len(input.Movements))
	var needsAI []int

	// Phase 1: history lookup (free, zero LLM calls)
	for i, m := range input.Movements {
		normalizedDesc := domain.NormalizeDescription(m.Description)
		catID, subCatID, err := u.movementRepo.FindRecentCategorizedByNormalizedDescription(ctx, normalizedDesc)
		if err != nil {
			needsAI = append(needsAI, i)
			continue
		}

		if catID != nil {
			suggestions[i] = domain.CategorySuggestion{
				Description:   m.Description,
				CategoryID:    catID,
				SubCategoryID: subCatID,
				Confidence:    1.0,
				Source:        "history",
			}
		} else {
			needsAI = append(needsAI, i)
		}
	}

	// Phase 2: batch LLM call for unmatched movements
	if len(needsAI) > 0 {
		toClassify := make([]domain.ExtractedMovement, len(needsAI))
		for j, idx := range needsAI {
			toClassify[j] = input.Movements[idx]
		}

		aiSuggestions, err := u.classificationGateway.ClassifyMovements(ctx, toClassify, categories)
		if err != nil {
			// Non-fatal: return what we have from history, AI slots remain zero-value
			return domain.StatementClassifyResult{Suggestions: suggestions}, nil
		}

		for j, idx := range needsAI {
			if j < len(aiSuggestions) {
				suggestions[idx] = aiSuggestions[j]
			}
		}
	}

	return domain.StatementClassifyResult{Suggestions: suggestions}, nil
}

func (u *StatementUseCase) Confirm(ctx context.Context, input domain.StatementConfirmInput) (domain.StatementConfirmResult, error) {
	userID := authentication.UserIDFromContext(ctx)
	if userID == "" {
		return domain.StatementConfirmResult{}, domain.ErrUnauthorized
	}

	if len(input.Movements) == 0 {
		return domain.StatementConfirmResult{}, domain.WrapInvalidInput(
			domain.New("no movements to import"),
			"validate input",
		)
	}

	// 1. Compute hashes for all movements
	hashes := make([]string, len(input.Movements))
	for i, m := range input.Movements {
		date, err := time.Parse("2006-01-02", m.Date)
		if err != nil {
			return domain.StatementConfirmResult{}, domain.WrapInvalidInput(
				fmt.Errorf("movement #%d: invalid date '%s'", i+1, m.Date),
				"validate date",
			)
		}
		hashes[i] = domain.ComputeIdempotencyHash(userID, input.WalletID, date, m.Amount, m.Description)
		_ = date // used in hash computation
	}

	// 2. Find existing hashes in the database
	existingHashes, err := u.movementRepo.FindExistingHashes(ctx, userID, hashes)
	if err != nil {
		return domain.StatementConfirmResult{}, fmt.Errorf("find existing hashes: %w", err)
	}

	uncategorizedID := uuid.MustParse(domain.UncategorizedCategoryID)

	// 3. Filter and insert only new movements
	var created, skipped int
	var errorsList []string

	for i, m := range input.Movements {
		categoryID := resolveCategoryID(m.CategoryID, uncategorizedID)

		// --- Recurrence link path ---
		if m.RecurrenceID != nil {
			date, err := time.Parse("2006-01-02", m.Date)
			if err != nil {
				errorsList = append(errorsList, fmt.Sprintf("movement #%d: invalid date '%s'", i+1, m.Date))
				skipped++
				continue
			}

			existing, err := u.movementRepo.FindByRecurrentIDAndMonth(ctx, *m.RecurrenceID, date)
			if err != nil {
				errorsList = append(errorsList, fmt.Sprintf("Could not link '%s': internal system error", m.Description))
				skipped++
				continue
			}

			linked := domain.Movement{
				Description: m.Description,
				Amount:      m.Amount,
				Date:        &date,
				WalletID:    &input.WalletID,
				IsPaid:      true,
			}

			if existing != nil {
				_, err = u.movementRepo.UpdateStatementLink(ctx, nil, *existing.ID, linked)
			} else {
				linked.RecurrentID = m.RecurrenceID
				linked.IsRecurrent = true
				linked.CategoryID = &categoryID
				linked.SubCategoryID = m.SubCategoryID
				_, err = u.movementRepo.Add(ctx, nil, linked)
			}

			if err != nil {
				errorsList = append(errorsList, fmt.Sprintf("Could not link '%s': internal system error", m.Description))
				skipped++
				continue
			}
			created++
			continue
		}

		// --- Normal import path ---
		if existingHashes[hashes[i]] {
			skipped++
			continue
		}

		// Validate plan limits before each creation
		if u.limitsValidator != nil {
			if err := u.limitsValidator.ValidateMovementCreation(ctx); err != nil {
				errorsList = append(errorsList, fmt.Sprintf("plan limit reached at movement #%d: %v", i+1, err))
				break
			}
		}

		date, _ := time.Parse("2006-01-02", m.Date)
		hash := hashes[i]

		movement := domain.Movement{
			Description:     m.Description,
			Amount:          m.Amount,
			Date:            &date,
			WalletID:        &input.WalletID,
			CategoryID:      &categoryID,
			SubCategoryID:   m.SubCategoryID,
			IsPaid:          true,
			IdempotencyHash: &hash,
		}

		_, err := u.movementRepo.Add(ctx, nil, movement)
		if err != nil {
			userReason := "internal validation error"
			if domain.Is(err, domain.ErrInvalidInput) {
				userReason = "invalid data"
			} else if domain.Is(err, domain.ErrConflict) {
				userReason = "duplicate entry"
			} else {
				userReason = "internal system error"
			}

			errorsList = append(errorsList, fmt.Sprintf("Could not save '%s': %s", m.Description, userReason))
			skipped++
			continue
		}

		existingHashes[hash] = true
		created++
	}

	return domain.StatementConfirmResult{
		Created: created,
		Skipped: skipped,
		Errors:  errorsList,
	}, nil
}

func resolveCategoryID(provided *uuid.UUID, defaultID uuid.UUID) uuid.UUID {
	if provided != nil {
		return *provided
	}
	return defaultID
}

func isAllowedMimeType(mimeType string) bool {
	switch mimeType {
	case "application/pdf", "image/jpeg", "image/png":
		return true
	}
	return false
}
