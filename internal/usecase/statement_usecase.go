package usecase

import (
	"context"
	"fmt"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/plataform/authentication"
	"personal-finance/pkg/log"
	"personal-finance/pkg/metrics"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const ClassificationConfidenceThreshold = 0.6

// --- Interfaces ---

type StatementVisionGateway interface {
	ExtractMovements(ctx context.Context, fileBytes []byte, mimeType, sourceType string) (domain.StatementExtractResult, error)
}

// StatementInvoiceUseCase é a interface estreita da InvoiceUseCase consumida pelo StatementUseCase.
type StatementInvoiceUseCase interface {
	FindOrCreateInvoiceForMovement(ctx context.Context, invoiceID *uuid.UUID, creditCardID *uuid.UUID, movementDate time.Time) (domain.Invoice, error)
	UpdateAmount(ctx context.Context, id uuid.UUID, amount float64) (domain.Invoice, error)
}

// StatementCreditCardRepository é a interface estreita do CreditCardRepository consumida pelo StatementUseCase.
type StatementCreditCardRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (domain.CreditCard, error)
	UpdateLimitDelta(ctx context.Context, tx *gorm.DB, id uuid.UUID, delta float64) (domain.CreditCard, error)
}

type StatementPDFDecryptor interface {
	// Prepare returns plaintext-ready PDF bytes, decrypting in memory when needed.
	Prepare(ctx context.Context, fileBytes []byte, password string) ([]byte, error)
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
	pdfDecryptor          StatementPDFDecryptor
	invoiceUseCase        StatementInvoiceUseCase
	creditCardRepo        StatementCreditCardRepository
}

func NewStatementUseCase(
	visionGateway StatementVisionGateway,
	classificationGateway StatementClassificationGateway,
	movementRepo StatementMovementRepository,
	categoryRepo StatementCategoryRepository,
	limitsValidator PlanLimitsValidatorInterface,
	pdfDecryptor StatementPDFDecryptor,
	invoiceUseCase StatementInvoiceUseCase,
	creditCardRepo StatementCreditCardRepository,
) *StatementUseCase {
	return &StatementUseCase{
		visionGateway:         visionGateway,
		classificationGateway: classificationGateway,
		movementRepo:          movementRepo,
		categoryRepo:          categoryRepo,
		limitsValidator:       limitsValidator,
		pdfDecryptor:          pdfDecryptor,
		invoiceUseCase:        invoiceUseCase,
		creditCardRepo:        creditCardRepo,
	}
}

// Extract processes a file (PDF or image) and returns extracted movements without saving.
// For password-protected PDFs, password may carry the user-supplied open password.
// sourceType is the client's declared intent ("statement" | "invoice" | ""); empty means auto-detect.
func (u *StatementUseCase) Extract(ctx context.Context, fileBytes []byte, mimeType, password, sourceType string) (domain.StatementExtractResult, error) {
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

	// Decrypt password-protected PDFs in memory before sending to the model.
	// Images pass through untouched.
	if mimeType == "application/pdf" && u.pdfDecryptor != nil {
		decrypted, err := u.pdfDecryptor.Prepare(ctx, fileBytes, password)
		if err != nil {
			return domain.StatementExtractResult{}, err
		}
		fileBytes = decrypted
	}

	// Call Gemini Vision — gateway selects prompt by sourceType.
	result, err := u.visionGateway.ExtractMovements(ctx, fileBytes, mimeType, sourceType)
	if err != nil {
		// Documentos ambíguos não são hard-fail: viram document_type=unknown + warning.
		if domain.Is(err, domain.ErrStatementNotAStatement) {
			return domain.StatementExtractResult{
				DocumentType: domain.DocUnknown,
				Confidence:   0,
				Warnings: []domain.ExtractWarning{
					{Type: "low_confidence"},
				},
				Movements: []domain.ExtractedMovement{},
			}, nil
		}
		return domain.StatementExtractResult{}, fmt.Errorf("extract movements: %w", err)
	}

	// Reconcilia intenção do cliente com a detecção da IA.
	if sourceType != "" && result.DocumentType != "" &&
		result.DocumentType != domain.DocUnknown &&
		string(result.DocumentType) != sourceType {
		result.Warnings = append(result.Warnings, domain.ExtractWarning{
			Type:     "document_type_mismatch",
			Expected: sourceType,
			Detected: string(result.DocumentType),
		})
	}

	// Confiança baixa → warning adicional.
	if result.DocumentType == domain.DocUnknown ||
		(result.Confidence > 0 && result.Confidence < ClassificationConfidenceThreshold) {
		alreadyHasLowConf := false
		for _, w := range result.Warnings {
			if w.Type == "low_confidence" {
				alreadyHasLowConf = true
				break
			}
		}
		if !alreadyHasLowConf {
			result.Warnings = append(result.Warnings, domain.ExtractWarning{Type: "low_confidence"})
		}
	}

	metrics.IncBusiness(
		ctx, "biz_statement_imports_total", 1,
		metrics.String("mime_type", mimeType),
	)

	return result, nil
}

// ConfirmInvoice cria movimentos de cartão de crédito a partir de itens extraídos de fatura,
// reutilizando a InvoiceUseCase existente para resolver/criar faturas e atualizar limites.
func (u *StatementUseCase) ConfirmInvoice(ctx context.Context, input domain.InvoiceConfirmInput) (domain.StatementConfirmResult, error) {
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

	// Valida que o cartão existe e pertence ao usuário.
	_, err := u.creditCardRepo.FindByID(ctx, input.CreditCardID)
	if err != nil {
		return domain.StatementConfirmResult{}, fmt.Errorf("find credit card: %w", err)
	}

	// Pré-calcula hashes escopados por creditCardID.
	hashes := make([]string, len(input.Movements))
	dates := make([]time.Time, len(input.Movements))
	for i, m := range input.Movements {
		date, err := time.Parse("2006-01-02", m.Date)
		if err != nil {
			return domain.StatementConfirmResult{}, domain.WrapInvalidInput(
				fmt.Errorf("movement #%d: invalid date '%s'", i+1, m.Date),
				"validate date",
			)
		}
		dates[i] = date
		hashes[i] = domain.ComputeIdempotencyHash(userID, input.CreditCardID.String(), date, m.Amount, m.Description)
	}

	existingHashes, err := u.movementRepo.FindExistingHashes(ctx, userID, hashes)
	if err != nil {
		return domain.StatementConfirmResult{}, fmt.Errorf("find existing hashes: %w", err)
	}

	uncategorizedID := uuid.MustParse(domain.UncategorizedCategoryID)

	var created, skipped int
	var errorsList []string

	for i, m := range input.Movements {
		// Deduplicação por hash.
		if existingHashes[hashes[i]] {
			log.Debug(
				"confirm invoice: skipped movement — duplicate hash",
				log.String("description", m.Description),
				log.String("date", m.Date),
				log.Float64("amount", m.Amount),
			)
			skipped++
			continue
		}

		categoryID := resolveCategoryID(m.CategoryID, uncategorizedID)
		date := dates[i]
		hash := hashes[i]

		// Resolve ou cria a fatura alvo pelo mês do movimento.
		invoice, err := u.invoiceUseCase.FindOrCreateInvoiceForMovement(ctx, input.InvoiceID, &input.CreditCardID, date)
		if err != nil {
			log.Debug(
				"confirm invoice: skipped movement — invoice resolve error",
				log.String("description", m.Description),
				log.Err(err),
			)
			errorsList = append(errorsList, fmt.Sprintf("Could not resolve invoice for '%s': internal system error", m.Description))
			skipped++
			continue
		}

		// Valida se a fatura já está paga.
		if invoice.IsPaid {
			return domain.StatementConfirmResult{
				Created: created,
				Skipped: skipped,
				Errors:  errorsList,
			}, ErrInvoiceAlreadyPaid
		}

		creditCardMovement := &domain.CreditCardMovement{
			InvoiceID:    invoice.ID,
			CreditCardID: &input.CreditCardID,
		}

		// Popula dados de parcelamento quando presentes.
		if m.InstallmentNumber != nil && m.TotalInstallments != nil {
			creditCardMovement.InstallmentNumber = m.InstallmentNumber
			creditCardMovement.TotalInstallments = m.TotalInstallments
		}

		movement := domain.Movement{
			Description:     m.Description,
			Amount:          m.Amount,
			Date:            &date,
			CategoryID:      &categoryID,
			SubCategoryID:   m.SubCategoryID,
			IsPaid:          false,
			IdempotencyHash: &hash,
			TypePayment:     domain.TypePaymentCreditCard,
			CreditCardInfo:  creditCardMovement,
		}

		// Itens parcelados geram a série completa de movimentos.
		if movement.IsInstallmentMovement() {
			movements := movement.GenerateInstallmentMovements()
			for _, installment := range movements {
				inst := installment

				// Resolve a fatura pelo mês de cada parcela.
				installmentInvoice, err := u.invoiceUseCase.FindOrCreateInvoiceForMovement(ctx, nil, &input.CreditCardID, *inst.Date)
				if err != nil {
					log.Debug(
						"confirm invoice: skipped installment — invoice resolve error",
						log.String("description", inst.Description),
						log.Err(err),
					)
					errorsList = append(errorsList, fmt.Sprintf("Could not resolve invoice for installment '%s': internal system error", inst.Description))
					skipped++
					continue
				}

				if inst.CreditCardInfo != nil {
					inst.CreditCardInfo.InvoiceID = installmentInvoice.ID
				}

				if _, err := u.movementRepo.Add(ctx, nil, inst); err != nil {
					log.Debug(
						"confirm invoice: skipped installment — add error",
						log.String("description", inst.Description),
						log.Err(err),
					)
					errorsList = append(errorsList, fmt.Sprintf("Could not save installment '%s': internal system error", inst.Description))
					skipped++
					continue
				}

				_, _ = u.invoiceUseCase.UpdateAmount(ctx, *installmentInvoice.ID, inst.Amount)
				_, _ = u.creditCardRepo.UpdateLimitDelta(ctx, nil, input.CreditCardID, inst.Amount)
				created++
			}
			existingHashes[hash] = true
			continue
		}

		// Movimento simples (sem parcelas).
		if _, err := u.movementRepo.Add(ctx, nil, movement); err != nil {
			userReason := "internal system error"
			if domain.Is(err, domain.ErrInvalidInput) {
				userReason = "invalid data"
			} else if domain.Is(err, domain.ErrConflict) {
				userReason = "duplicate entry"
			}
			log.Debug(
				"confirm invoice: skipped movement — add error",
				log.String("description", m.Description),
				log.String("date", m.Date),
				log.Float64("amount", m.Amount),
				log.String("reason", userReason),
				log.Err(err),
			)
			errorsList = append(errorsList, fmt.Sprintf("Could not save '%s': %s", m.Description, userReason))
			skipped++
			continue
		}

		_, _ = u.invoiceUseCase.UpdateAmount(ctx, *invoice.ID, m.Amount)
		_, _ = u.creditCardRepo.UpdateLimitDelta(ctx, nil, input.CreditCardID, m.Amount)

		existingHashes[hash] = true
		created++
	}

	if created > 0 {
		metrics.IncBusiness(ctx, "biz_invoice_imports_total", int64(created))
	}

	return domain.StatementConfirmResult{
		Created: created,
		Skipped: skipped,
		Errors:  errorsList,
	}, nil
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
		hashes[i] = domain.ComputeIdempotencyHash(userID, input.WalletID.String(), date, m.Amount, m.Description)
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
				log.Debug(
					"statement confirm: skipped recurrent movement — invalid date",
					log.String("description", m.Description),
					log.String("date", m.Date),
				)
				errorsList = append(errorsList, fmt.Sprintf("movement #%d: invalid date '%s'", i+1, m.Date))
				skipped++
				continue
			}

			existing, err := u.movementRepo.FindByRecurrentIDAndMonth(ctx, *m.RecurrenceID, date)
			if err != nil {
				log.Debug(
					"statement confirm: skipped recurrent movement — lookup error",
					log.String("description", m.Description),
					log.String("recurrence_id", m.RecurrenceID.String()),
					log.Err(err),
				)
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
				TypePayment: resolveTypePayment(m.TypePayment),
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
				log.Debug(
					"statement confirm: skipped recurrent movement — link/create error",
					log.String("description", m.Description),
					log.String("recurrence_id", m.RecurrenceID.String()),
					log.Err(err),
				)
				errorsList = append(errorsList, fmt.Sprintf("Could not link '%s': internal system error", m.Description))
				skipped++
				continue
			}
			created++
			continue
		}

		// --- Normal import path ---
		if existingHashes[hashes[i]] {
			log.Debug(
				"statement confirm: skipped movement — duplicate hash",
				log.String("description", m.Description),
				log.String("date", m.Date),
				log.Float64("amount", m.Amount),
			)
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
			TypePayment:     resolveTypePayment(m.TypePayment),
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

			log.Debug(
				"statement confirm: skipped movement — add error",
				log.String("description", m.Description),
				log.String("date", m.Date),
				log.Float64("amount", m.Amount),
				log.String("reason", userReason),
				log.Err(err),
			)
			errorsList = append(errorsList, fmt.Sprintf("Could not save '%s': %s", m.Description, userReason))
			skipped++
			continue
		}

		existingHashes[hash] = true
		created++
	}

	if created > 0 {
		metrics.IncBusiness(ctx, "biz_statement_movements_imported_total", int64(created))
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

func resolveTypePayment(extracted domain.TypePayment) domain.TypePayment {
	switch extracted {
	case domain.TypePaymentPix, domain.TypePaymentDebit, domain.TypePaymentTED, domain.TypePaymentDOC:
		return extracted
	default:
		return domain.TypePaymentDebit
	}
}

func isAllowedMimeType(mimeType string) bool {
	switch mimeType {
	case "application/pdf", "image/jpeg", "image/png":
		return true
	}
	return false
}
