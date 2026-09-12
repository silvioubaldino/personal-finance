package usecase

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/infrastructure/repository/transaction"
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
}

// StatementInvoiceRepository é a interface estreita do InvoiceRepository consumida pelo
// StatementUseCase. A atualização do total da fatura precisa acontecer dentro da mesma
// transação da gravação do movimento, e a InvoiceUseCase.UpdateAmount abre a própria
// transação — por isso o repositório é acessado direto aqui, mesmo padrão da Movement
// usecase (ver movement_usecase.go).
type StatementInvoiceRepository interface {
	UpdateAmount(ctx context.Context, tx *gorm.DB, id uuid.UUID, amount float64) (domain.Invoice, error)
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
	FindInstallmentCandidatesByCreditCard(ctx context.Context, creditCardID uuid.UUID, totalInstallments int) (domain.MovementList, error)
	FindByInstallmentGroupFromNumber(ctx context.Context, groupID uuid.UUID, fromNumber int) (domain.MovementList, error)
	UpdateAmount(ctx context.Context, tx *gorm.DB, id uuid.UUID, amount float64) (domain.Movement, error)
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
	invoiceRepo           StatementInvoiceRepository
	creditCardRepo        StatementCreditCardRepository
	txManager             transaction.Manager
}

func NewStatementUseCase(
	visionGateway StatementVisionGateway,
	classificationGateway StatementClassificationGateway,
	movementRepo StatementMovementRepository,
	categoryRepo StatementCategoryRepository,
	limitsValidator PlanLimitsValidatorInterface,
	pdfDecryptor StatementPDFDecryptor,
	invoiceUseCase StatementInvoiceUseCase,
	invoiceRepo StatementInvoiceRepository,
	creditCardRepo StatementCreditCardRepository,
	txManager transaction.Manager,
) *StatementUseCase {
	return &StatementUseCase{
		visionGateway:         visionGateway,
		classificationGateway: classificationGateway,
		movementRepo:          movementRepo,
		categoryRepo:          categoryRepo,
		limitsValidator:       limitsValidator,
		pdfDecryptor:          pdfDecryptor,
		invoiceUseCase:        invoiceUseCase,
		invoiceRepo:           invoiceRepo,
		creditCardRepo:        creditCardRepo,
		txManager:             txManager,
	}
}

// Extract processes a file (PDF or image) and returns extracted movements without saving.
// For password-protected PDFs, password may carry the user-supplied open password.
// sourceType is the client's declared intent ("statement" | "invoice" | ""); empty means auto-detect.
//
// creditCardID é opcional e só é usado quando o documento é uma fatura: com ele a
// extração ganha os dois enriquecimentos da Fase 6 (marcação de competência futura
// e sugestão de vínculo de parcela), que dependem do fechamento e do histórico
// **daquele** cartão. Sem ele a extração se comporta exatamente como antes — o
// confirm-invoice, que sempre recebe o cartão, reaplica as regras defensivamente
// (AYD-004 §"Por que credit_card_id no /extract").
func (u *StatementUseCase) Extract(
	ctx context.Context,
	fileBytes []byte,
	mimeType, password, sourceType string,
	creditCardID *uuid.UUID,
) (domain.StatementExtractResult, error) {
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
					{Type: domain.WarningLowConfidence},
				},
				Movements: []domain.ExtractedMovement{},
			}, nil
		}
		return domain.StatementExtractResult{}, fmt.Errorf("extract movements: %w", err)
	}

	// Reconcilia intenção do cliente com a detecção da IA.
	if result.IsDocumentTypeMismatch(sourceType) {
		result.Warnings = append(result.Warnings, domain.ExtractWarning{
			Type:     domain.WarningDocumentTypeMismatch,
			Expected: sourceType,
			Detected: string(result.DocumentType),
		})
	}

	// Confiança baixa → warning adicional.
	if result.IsLowConfidence(ClassificationConfidenceThreshold) && !result.HasWarning(domain.WarningLowConfidence) {
		result.Warnings = append(result.Warnings, domain.ExtractWarning{Type: domain.WarningLowConfidence})
	}

	// Itens que não pertencem à fatura (camadas 2 e 3 do AYD-004): marca o
	// pagamento da fatura anterior e confere a soma contra o total declarado.
	markItemsNotBelongingToInvoice(ctx, &result)

	// Enriquecimentos que dependem do cartão de destino (Fase 6): competência
	// futura e vínculo com séries de parcelas já registradas.
	u.enrichInvoiceExtraction(ctx, &result, creditCardID)

	metrics.IncBusiness(
		ctx, "biz_statement_imports_total", 1,
		metrics.String("mime_type", mimeType),
	)

	return result, nil
}

// markItemsNotBelongingToInvoice aplica as camadas 2 e 3 do AYD-004
// (§"Itens que não pertencem à fatura") sobre um resultado de extração de fatura:
//
//	camada 2 — marca (nunca remove) o pagamento da fatura anterior;
//	camada 3 — confere a soma dos itens restantes contra invoice_meta.total_amount.
//
// É no-op para extrato bancário: pagamento de fatura só é ruído dentro de fatura.
func markItemsNotBelongingToInvoice(ctx context.Context, result *domain.StatementExtractResult) {
	if result.DocumentType != domain.DocInvoice {
		return
	}

	var excluded int
	for i := range result.Movements {
		if !domain.IsInvoicePaymentDescription(result.Movements[i].Description) {
			continue
		}
		result.Movements[i].Excluded = true
		result.Movements[i].ExclusionReason = domain.ExclusionReasonInvoicePayment
		excluded++
		log.InfoContext(ctx, "extract: item marcado como pagamento de fatura anterior",
			log.String("description", result.Movements[i].Description),
			log.Float64("amount", result.Movements[i].Amount),
		)
	}

	if excluded > 0 && !result.HasWarning(domain.WarningInvoicePaymentExcluded) {
		result.Warnings = append(result.Warnings, domain.ExtractWarning{
			Type: domain.WarningInvoicePaymentExcluded,
		})
	}

	appendTotalMismatchWarning(ctx, result)
}

// appendTotalMismatchWarning compara a soma dos itens que de fato pertencem à
// fatura com o total declarado no documento. Divergência é warning informativo,
// nunca bloqueio (AYD-004, decisão 3).
//
// Compara **módulos**: o modelo é inconsistente no sinal do total (já devolveu
// "6035.06" positivo para uma fatura de itens negativos), e o que se quer checar
// aqui é se o conjunto de linhas extraídas está completo — não o sinal.
func appendTotalMismatchWarning(ctx context.Context, result *domain.StatementExtractResult) {
	if result.InvoiceMeta == nil || result.InvoiceMeta.TotalAmount == nil {
		return
	}

	var sum float64
	for _, m := range result.Movements {
		if m.Excluded {
			continue
		}
		sum += m.Amount
	}

	declared := *result.InvoiceMeta.TotalAmount
	if math.Abs(math.Abs(sum)-math.Abs(declared)) <= domain.InvoiceTotalTolerance {
		return
	}

	log.WarnContext(ctx, "extract: soma dos itens diverge do total da fatura",
		log.Float64("declared_total", declared),
		log.Float64("computed_sum", sum),
	)
	result.Warnings = append(result.Warnings, domain.ExtractWarning{
		Type:     domain.WarningTotalAmountMismatch,
		Expected: strconv.FormatFloat(declared, 'f', 2, 64),
		Detected: strconv.FormatFloat(sum, 'f', 2, 64),
	})
}

// enrichInvoiceExtraction aplica os enriquecimentos da Fase 6 sobre um resultado
// de extração de fatura: marca as parcelas de competência futura (caso A da
// AYD-004) e sugere o vínculo com séries de parcelas já registradas (caso B).
//
// É no-op sem cartão de destino ou fora de uma fatura: as duas regras dependem do
// fechamento e do histórico daquele cartão. Falha de leitura do cartão ou do
// histórico também é no-op — o enriquecimento é uma comodidade da revisão, não
// pode derrubar a extração (princípio 2, "falhar suave"); o confirm-invoice
// reaplica a regra de competência futura de qualquer forma.
func (u *StatementUseCase) enrichInvoiceExtraction(
	ctx context.Context,
	result *domain.StatementExtractResult,
	creditCardID *uuid.UUID,
) {
	if creditCardID == nil || result.DocumentType != domain.DocInvoice || u.creditCardRepo == nil {
		return
	}

	creditCard, err := u.creditCardRepo.FindByID(ctx, *creditCardID)
	if err != nil {
		log.WarnContext(ctx, "extract: enriquecimento de fatura ignorado — cartão não encontrado",
			log.String("credit_card_id", creditCardID.String()),
			log.Err(err),
		)
		return
	}

	if marked := markFutureInstallments(ctx, result.Movements, creditCard); marked > 0 &&
		!result.HasWarning(domain.WarningFutureInstallmentExcluded) {
		result.Warnings = append(result.Warnings, domain.ExtractWarning{
			Type: domain.WarningFutureInstallmentExcluded,
		})
	}

	if matched := u.suggestInstallmentMatches(ctx, result.Movements, *creditCardID); matched > 0 &&
		!result.HasWarning(domain.WarningInstallmentMatchFound) {
		result.Warnings = append(result.Warnings, domain.ExtractWarning{
			Type: domain.WarningInstallmentMatchFound,
		})
	}
}

// markFutureInstallments marca os itens cuja data ultrapassa o fim do período da
// fatura alvo: eles caem numa fatura seguinte e não são despesa desta (AYD-004,
// decisão 7). Regra determinística, sem decisão do usuário — e, como o pagamento
// da fatura anterior, **marca, nunca remove**: o item continua na lista, a UI o
// mostra desmarcado e o usuário pode reincluí-lo.
func markFutureInstallments(ctx context.Context, movements []domain.ExtractedMovement, creditCard domain.CreditCard) int {
	periodEnd, ok := domain.ResolveTargetInvoicePeriodEnd(creditCard, invoiceReferenceDates(movements))
	if !ok {
		return 0
	}

	var marked int
	for i := range movements {
		if movements[i].Excluded {
			continue
		}

		// Só item PARCELADO. Uma compra comum datada depois do fechamento pertence
		// à fatura seguinte e é parenteada corretamente pela resolução por data
		// (AYD-004 decisão 2, que cobre faturas que cruzam o fechamento) — marcá-la
		// aqui faria o usuário perdê-la. A decisão 7 fala de "parcelas de
		// competência futura", e o caso A da §"Três casos" é explícito no ponto.
		if movements[i].InstallmentNumber == nil || movements[i].TotalInstallments == nil {
			continue
		}

		date, err := time.Parse("2006-01-02", movements[i].Date)
		if err != nil || !date.After(periodEnd) {
			continue
		}

		movements[i].Excluded = true
		movements[i].ExclusionReason = domain.ExclusionReasonFutureInstallment
		marked++
		log.InfoContext(ctx, "extract: item marcado como parcela de competência futura",
			log.String("description", movements[i].Description),
			log.String("date", movements[i].Date),
		)
	}

	return marked
}

// invoiceReferenceDates devolve as datas dos itens que ainda concorrem a pertencer
// à fatura. Itens já marcados (o pagamento da fatura anterior) ficam de fora: eles
// costumam ser datados perto do vencimento, e deixá-los votar deslocaria o período
// alvo.
func invoiceReferenceDates(movements []domain.ExtractedMovement) []time.Time {
	dates := make([]time.Time, 0, len(movements))
	for _, m := range movements {
		if m.Excluded {
			continue
		}
		date, err := time.Parse("2006-01-02", m.Date)
		if err != nil {
			continue
		}
		dates = append(dates, date)
	}
	return dates
}

// suggestInstallmentMatches procura, para cada item parcelado, a série já
// registrada no cartão que corresponde a ele (AYD-004 §"Parcelas já registradas
// no app"). Confiança alta vem **pré-vinculada** (installment_group_id
// preenchido); média vem só como sugestão, para a UI aceitar num toque.
func (u *StatementUseCase) suggestInstallmentMatches(
	ctx context.Context,
	movements []domain.ExtractedMovement,
	creditCardID uuid.UUID,
) int {
	// Uma consulta por total de parcelas distinto, reaproveitada entre os itens:
	// uma fatura com 20 parcelamentos de 12x não faz 20 idas ao banco.
	candidatesByTotal := make(map[int]domain.MovementList)

	var matched int
	for i := range movements {
		item := movements[i]
		if item.Excluded || item.InstallmentNumber == nil || item.TotalInstallments == nil {
			continue
		}

		candidates, cached := candidatesByTotal[*item.TotalInstallments]
		if !cached {
			found, err := u.movementRepo.FindInstallmentCandidatesByCreditCard(ctx, creditCardID, *item.TotalInstallments)
			if err != nil {
				log.WarnContext(ctx, "extract: busca de parcelas já registradas falhou",
					log.String("credit_card_id", creditCardID.String()),
					log.Err(err),
				)
				return matched
			}
			candidates = found
			candidatesByTotal[*item.TotalInstallments] = candidates
		}

		match, confidence := bestInstallmentMatch(item, candidates)
		if match == nil {
			continue
		}

		movements[i].InstallmentMatch = match
		if confidence >= domain.InstallmentMatchConfidenceHigh {
			// Pré-aplicar em vez de perguntar: numa fatura de 70+ itens um modal
			// por parcela mata a revisão, e a UI desvincula num toque.
			movements[i].InstallmentGroupID = &match.InstallmentGroupID
		}
		matched++
	}

	return matched
}

// bestInstallmentMatch escolhe, entre as parcelas candidatas, a de maior confiança.
// Devolve nil quando nenhuma alcança sequer a confiança média.
func bestInstallmentMatch(item domain.ExtractedMovement, candidates domain.MovementList) (*domain.InstallmentMatch, float64) {
	var (
		best           domain.Movement
		bestConfidence = domain.InstallmentMatchConfidenceNone
	)
	for _, candidate := range candidates {
		confidence := domain.InstallmentMatchConfidence(item, candidate)
		if confidence > bestConfidence {
			best, bestConfidence = candidate, confidence
		}
	}

	if bestConfidence == domain.InstallmentMatchConfidenceNone || best.ID == nil {
		return nil, domain.InstallmentMatchConfidenceNone
	}

	return &domain.InstallmentMatch{
		InstallmentGroupID: *best.CreditCardInfo.InstallmentGroupID,
		MovementID:         *best.ID,
		Description:        best.Description,
		InstallmentNumber:  *best.CreditCardInfo.InstallmentNumber,
		TotalInstallments:  *best.CreditCardInfo.TotalInstallments,
		Amount:             best.Amount,
		Confidence:         bestConfidence,
	}, bestConfidence
}

// creditLimitGuard acompanha o limite disponível do cartão ao longo de uma
// importação. A regra do estouro é a mesma do lançamento manual —
// domain.CreditCard.HasSufficientLimit, também usada por
// Movement.validateCreditLimit — mas aqui o limite é debitado em memória item a
// item: reconsultar o cartão a cada item custaria N idas ao banco, e não
// acompanhar deixaria uma fatura inteira furar o limite, já que cada item passa
// sozinho.
type creditLimitGuard struct {
	creditCard domain.CreditCard
}

func (g *creditLimitGuard) allows(amount float64) bool {
	return g.creditCard.HasSufficientLimit(amount)
}

func (g *creditLimitGuard) consume(amount float64) {
	g.creditCard.CreditLimit += amount
}

// invoiceLinkOutcome é o desfecho da tentativa de vincular um item extraído a uma
// série de parcelas já registrada. `linked` falso com `errMsg`/`abort` vazios
// significa "não há vínculo aqui" — o item segue para o caminho normal de criação.
type invoiceLinkOutcome struct {
	linked  bool
	skipped int
	errMsg  string
	abort   error
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

	creditCard, err := u.creditCardRepo.FindByID(ctx, input.CreditCardID)
	if err != nil {
		return domain.StatementConfirmResult{}, fmt.Errorf("find credit card: %w", err)
	}

	// Item de fatura nunca traz carteira — quem a fornece é o cartão, via
	// invoice.WalletID herdado da carteira default. Sem ela a fatura fica sem
	// conta de pagamento (AYD-004 §confirm-invoice, "Erros adicionais").
	if creditCard.DefaultWalletID == nil {
		return domain.StatementConfirmResult{}, ErrCreditCardNoDefaultWallet
	}

	dates, hashes, existingHashes, err := u.parseAndHashMovements(ctx, userID, input.Movements, input.CreditCardID.String())
	if err != nil {
		return domain.StatementConfirmResult{}, err
	}

	uncategorizedID := uuid.MustParse(domain.UncategorizedCategoryID)
	uncategorizedIncomeID := uuid.MustParse(domain.UncategorizedIncomeCategoryID)

	guard := &creditLimitGuard{creditCard: creditCard}
	periodEnd, hasPeriodEnd := domain.ResolveTargetInvoicePeriodEnd(creditCard, invoiceReferenceDates(input.Movements))

	var created, skipped int
	var errorsList []string

	for i, m := range input.Movements {
		// Itens que não pertencem à fatura não entram, nem que o cliente os envie:
		// além de honrar a marca vinda do extract, a detecção é reaplicada aqui, para
		// que um cliente que ignore o campo `excluded` não consiga inflar a fatura e
		// consumir limite do cartão (AYD-004 §"Itens que não pertencem à fatura").
		if m.Excluded || domain.IsInvoicePaymentDescription(m.Description) {
			log.Debug(
				"confirm invoice: skipped movement — does not belong to the invoice",
				log.String("description", m.Description),
				log.Float64("amount", m.Amount),
			)
			skipped++
			continue
		}

		// Mesma desconfiança para a competência futura (decisão 7): item datado
		// depois do fechamento da fatura alvo é de uma fatura seguinte.
		isInstallment := m.InstallmentNumber != nil && m.TotalInstallments != nil
		if isInstallment && hasPeriodEnd && dates[i].After(periodEnd) {
			log.Debug(
				"confirm invoice: skipped movement — future installment",
				log.String("description", m.Description),
				log.String("date", m.Date),
			)
			skipped++
			continue
		}

		// Deduplicação por hash. Para movimentos parcelados, o hash de input.Movements[i]
		// corresponde apenas à parcela informada (ex.: 3/12); se ela já existe, toda a série
		// 3..12 já foi importada antes, então contamos o skip pelo total de parcelas restantes.
		if existingHashes[hashes[i]] {
			skipCount := 1
			if m.InstallmentNumber != nil && m.TotalInstallments != nil {
				if remaining := *m.TotalInstallments - *m.InstallmentNumber + 1; remaining > skipCount {
					skipCount = remaining
				}
			}
			log.Debug(
				"confirm invoice: skipped movement — duplicate hash",
				log.String("description", m.Description),
				log.String("date", m.Date),
				log.Float64("amount", m.Amount),
				log.Int("skip_count", skipCount),
			)
			skipped += skipCount
			continue
		}

		// --- Caminho de vínculo: a série já existe, atualiza-se o valor da parcela
		// desta competência em vez de criar duplicatas (AYD-004, decisão 8). ---
		link := u.linkInstallmentSeries(ctx, guard, input.CreditCardID, m)
		if link.abort != nil {
			return domain.StatementConfirmResult{Created: created, Skipped: skipped, Errors: errorsList}, link.abort
		}
		if link.errMsg != "" {
			errorsList = append(errorsList, link.errMsg)
			skipped += link.skipped
			continue
		}
		if link.linked {
			skipped += link.skipped
			continue
		}

		categoryID := resolveCategoryID(m.CategoryID, m.Amount, uncategorizedID, uncategorizedIncomeID)
		date := dates[i]
		hash := hashes[i]

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

		if movement.IsInstallmentMovement() {
			c, s, errs, abort := u.saveInstallmentSeries(ctx, guard, input.CreditCardID, userID, movement, existingHashes)
			if abort != nil {
				return domain.StatementConfirmResult{Created: created + c, Skipped: skipped + s, Errors: append(errorsList, errs...)}, abort
			}
			created += c
			skipped += s
			errorsList = append(errorsList, errs...)
			continue
		}

		ok, errMsg, abort := u.saveSingleInvoiceMovement(ctx, guard, input.CreditCardID, invoice, movement)
		if abort != nil {
			return domain.StatementConfirmResult{Created: created, Skipped: skipped, Errors: errorsList}, abort
		}
		if !ok {
			errorsList = append(errorsList, errMsg)
			skipped++
			continue
		}
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

// linkInstallmentSeries efetiva o vínculo de um item extraído com uma série de
// parcelas já registrada (AYD-004 §"Semântica do vínculo no confirm-invoice"):
// atualiza **só o valor** da parcela daquela competência — a fatura é a fonte de
// verdade sobre quanto foi cobrado, e parcelamentos variam centavos entre
// parcelas — ajustando fatura e limite pelo **delta**, e pula a série inteira.
//
// Não toca em `description` (é o rótulo que o usuário escolheu e reconhece),
// `date` (mudá-la reparentearia a parcela para outra competência) nem `is_paid`
// (quem quita é o pagamento da fatura, modelado fora dela).
func (u *StatementUseCase) linkInstallmentSeries(
	ctx context.Context,
	guard *creditLimitGuard,
	creditCardID uuid.UUID,
	item domain.ExtractedMovement,
) invoiceLinkOutcome {
	if item.InstallmentGroupID == nil || item.InstallmentNumber == nil || item.TotalInstallments == nil {
		return invoiceLinkOutcome{}
	}

	target, found, err := u.findLinkedInstallment(ctx, creditCardID, item)
	if err != nil {
		log.Debug(
			"confirm invoice: skipped movement — installment link lookup error",
			log.String("description", item.Description),
			log.Err(err),
		)
		return invoiceLinkOutcome{
			skipped: 1,
			errMsg:  fmt.Sprintf("Could not link '%s': internal system error", item.Description),
		}
	}
	if !found {
		// Vínculo inválido — grupo inexistente, de outro usuário ou de outro
		// cartão. O servidor não confia no cliente: o item cai no caminho normal
		// de criação em vez de escrever numa série que não é dele.
		log.Debug(
			"confirm invoice: rejected installment link — group does not belong to this card",
			log.String("description", item.Description),
			log.String("installment_group_id", item.InstallmentGroupID.String()),
		)
		return invoiceLinkOutcome{}
	}

	invoice, err := u.invoiceUseCase.FindOrCreateInvoiceForMovement(ctx, target.CreditCardInfo.InvoiceID, &creditCardID, *target.Date)
	if err != nil {
		log.Debug(
			"confirm invoice: skipped movement — invoice resolve error on installment link",
			log.String("description", item.Description),
			log.Err(err),
		)
		return invoiceLinkOutcome{
			skipped: 1,
			errMsg:  fmt.Sprintf("Could not resolve invoice for '%s': internal system error", item.Description),
		}
	}
	if invoice.IsPaid {
		return invoiceLinkOutcome{abort: ErrInvoiceAlreadyPaid}
	}

	// A parcela já foi somada na fatura e no limite quando foi criada; o que muda
	// agora é só a diferença entre o valor registrado e o cobrado.
	delta := item.Amount - target.Amount
	if !guard.allows(delta) {
		return invoiceLinkOutcome{abort: ErrInsufficientCreditLimit}
	}

	linked := target
	linked.Amount = item.Amount

	if err := u.persistInvoiceMovements(ctx, creditCardID, []invoiceMovementItem{{
		movement: linked,
		invoice:  invoice,
		linkTo:   target.ID,
		delta:    delta,
	}}); err != nil {
		log.Debug(
			"confirm invoice: skipped movement — installment link persist error",
			log.String("description", item.Description),
			log.Err(err),
		)
		return invoiceLinkOutcome{
			skipped: 1,
			errMsg:  fmt.Sprintf("Could not link '%s': internal system error", item.Description),
		}
	}
	guard.consume(delta)

	// Vinculada a parcela desta competência, as restantes já existem no mesmo
	// grupo — nada é criado. O skipped contabiliza a série inteira, a mesma
	// contagem que o dedup por hash já faz para séries repetidas.
	return invoiceLinkOutcome{
		linked:  true,
		skipped: *item.TotalInstallments - *item.InstallmentNumber + 1,
	}
}

// findLinkedInstallment localiza a parcela desta competência dentro da série que
// o cliente indicou, revalidando o vínculo do lado do servidor: a consulta é
// escopada pelo usuário e o resultado precisa ser do mesmo cartão, do mesmo total
// de parcelas e do mesmo número de parcela informados.
func (u *StatementUseCase) findLinkedInstallment(
	ctx context.Context,
	creditCardID uuid.UUID,
	item domain.ExtractedMovement,
) (domain.Movement, bool, error) {
	series, err := u.movementRepo.FindByInstallmentGroupFromNumber(ctx, *item.InstallmentGroupID, *item.InstallmentNumber)
	if err != nil {
		return domain.Movement{}, false, fmt.Errorf("find installment group: %w", err)
	}
	if len(series) == 0 {
		return domain.Movement{}, false, nil
	}

	target := series[0]
	info := target.CreditCardInfo
	if target.ID == nil || target.Date == nil || info == nil ||
		info.CreditCardID == nil || *info.CreditCardID != creditCardID ||
		info.InstallmentNumber == nil || *info.InstallmentNumber != *item.InstallmentNumber ||
		info.TotalInstallments == nil || *info.TotalInstallments != *item.TotalInstallments {
		return domain.Movement{}, false, nil
	}

	return target, true, nil
}

// parseAndHashMovements converte as datas dos movimentos de string para time.Time,
// calcula os hashes de idempotência escopados por scopeKey e busca no repositório quais já existem.
func (u *StatementUseCase) parseAndHashMovements(
	ctx context.Context,
	userID string,
	movements []domain.ExtractedMovement,
	scopeKey string,
) ([]time.Time, []string, map[string]bool, error) {
	dates := make([]time.Time, len(movements))
	hashes := make([]string, len(movements))

	for i, m := range movements {
		date, err := time.Parse("2006-01-02", m.Date)
		if err != nil {
			return nil, nil, nil, domain.WrapInvalidInput(
				fmt.Errorf("movement #%d: invalid date '%s'", i+1, m.Date),
				"validate date",
			)
		}
		dates[i] = date
		hashes[i] = domain.ComputeIdempotencyHash(userID, scopeKey, date, m.Amount, m.Description)
	}

	existingHashes, err := u.movementRepo.FindExistingHashes(ctx, userID, hashes)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("find existing hashes: %w", err)
	}

	return dates, hashes, existingHashes, nil
}

// invoiceMovementItem é um movimento pronto para gravar junto da fatura em que ele entra —
// o par de que a transação precisa para somar o valor no total certo.
type invoiceMovementItem struct {
	movement domain.Movement
	invoice  domain.Invoice
	// linkTo, quando preenchido, indica que o item vincula a uma parcela já
	// registrada: em vez de inserir um movimento novo, atualiza só o valor dela.
	linkTo *uuid.UUID
	// delta é quanto o item soma ao total da fatura e ao limite do cartão — o
	// próprio valor numa criação, a diferença contra o valor antigo num vínculo.
	delta float64
}

// invoiceTotal acumula, dentro de uma transação, o quanto os itens somam a uma fatura.
type invoiceTotal struct {
	amount float64 // total da fatura antes desta transação
	delta  float64 // quanto os itens desta transação somam nela
}

// persistInvoiceMovements grava um conjunto de movimentos de cartão e os efeitos colaterais
// deles — total de cada fatura tocada e limite do cartão — numa transação única: ou tudo
// entra, ou nada entra. Antes as três escritas rodavam soltas, e as duas últimas ainda
// descartavam o erro, então uma falha deixava a fatura e o limite dessincronizados dos
// movimentos.
//
// Os deltas são acumulados em memória e aplicados num update só por fatura. Reler o total
// entre updates não funcionaria: InvoiceRepository.FindByID lê fora da transação e não
// enxergaria as escritas ainda não commitadas.
func (u *StatementUseCase) persistInvoiceMovements(
	ctx context.Context,
	creditCardID uuid.UUID,
	items []invoiceMovementItem,
) error {
	return u.txManager.WithTransaction(ctx, func(tx *gorm.DB) error {
		var (
			order      []uuid.UUID
			totals     = make(map[uuid.UUID]*invoiceTotal, len(items))
			limitDelta float64
		)

		for _, item := range items {
			if item.linkTo != nil {
				if _, err := u.movementRepo.UpdateAmount(ctx, tx, *item.linkTo, item.movement.Amount); err != nil {
					return fmt.Errorf("update movement amount: %w", err)
				}
			} else if _, err := u.movementRepo.Add(ctx, tx, item.movement); err != nil {
				return fmt.Errorf("add movement: %w", err)
			}

			invoiceID := *item.invoice.ID
			total, ok := totals[invoiceID]
			if !ok {
				total = &invoiceTotal{amount: item.invoice.Amount}
				totals[invoiceID] = total
				order = append(order, invoiceID)
			}
			total.delta += item.delta
			limitDelta += item.delta
		}

		// Percorre na ordem de inserção, não na do mapa: sequência de updates previsível.
		for _, invoiceID := range order {
			total := totals[invoiceID]
			if _, err := u.invoiceRepo.UpdateAmount(ctx, tx, invoiceID, total.amount+total.delta); err != nil {
				return fmt.Errorf("update invoice amount: %w", err)
			}
		}

		if _, err := u.creditCardRepo.UpdateLimitDelta(ctx, tx, creditCardID, limitDelta); err != nil {
			return fmt.Errorf("update credit card limit: %w", err)
		}

		return nil
	})
}

// saveInstallmentSeries persiste a série completa de parcelas de um movimento parcelado,
// resolvendo a fatura correspondente para cada mês. A série é uma compra só, então grava
// numa transação única: uma série pela metade (parcelas 1..6 gravadas, 7..12 não) seria
// pior que nenhuma — infla a fatura sem representar a compra.
// O mapa existingHashes é atualizado in-place quando a série entra.
func (u *StatementUseCase) saveInstallmentSeries(
	ctx context.Context,
	guard *creditLimitGuard,
	creditCardID uuid.UUID,
	userID string,
	baseMovement domain.Movement,
	existingHashes map[string]bool,
) (created int, skipped int, errors []string, abort error) {
	var (
		items  []invoiceMovementItem
		hashes []string
	)

	// Resolver/criar as faturas fica fora da transação de propósito:
	// FindOrCreateInvoiceForMovement abre a própria transação ao criar uma fatura, e
	// chamá-la lá dentro abriria uma transação paralela, que commitaria por fora de um
	// eventual rollback desta. Fatura criada sem itens é inofensiva e reaproveitável.
	for _, installment := range baseMovement.GenerateInstallmentMovements() {
		inst := installment

		instHash := domain.ComputeIdempotencyHash(userID, creditCardID.String(), *inst.Date, inst.Amount, inst.Description)
		if existingHashes[instHash] {
			log.Debug(
				"confirm invoice: skipped installment — duplicate hash",
				log.String("description", inst.Description),
				log.Float64("amount", inst.Amount),
			)
			skipped++
			continue
		}
		inst.IdempotencyHash = &instHash

		installmentInvoice, err := u.invoiceUseCase.FindOrCreateInvoiceForMovement(ctx, nil, &creditCardID, *inst.Date)
		if err != nil {
			log.Debug(
				"confirm invoice: skipped installment — invoice resolve error",
				log.String("description", inst.Description),
				log.Err(err),
			)
			errors = append(errors, fmt.Sprintf("Could not resolve invoice for installment '%s': internal system error", inst.Description))
			skipped++
			continue
		}

		// A InvoiceUseCase.UpdateAmount barrava fatura paga (ErrInvoiceCannotModify);
		// como o total agora é atualizado pelo repositório, a checagem vem explícita aqui.
		if installmentInvoice.IsPaid {
			log.Debug(
				"confirm invoice: skipped installment — invoice already paid",
				log.String("description", inst.Description),
			)
			errors = append(errors, fmt.Sprintf("Could not save installment '%s': invoice already paid", inst.Description))
			skipped++
			continue
		}

		if inst.CreditCardInfo != nil {
			inst.CreditCardInfo.InvoiceID = installmentInvoice.ID
		}

		items = append(items, invoiceMovementItem{movement: inst, invoice: installmentInvoice, delta: inst.Amount})
		hashes = append(hashes, instHash)
	}

	if len(items) == 0 {
		return created, skipped, errors, nil
	}

	// O limite é validado contra a série inteira, não parcela a parcela: é uma
	// compra só, e aprovar metade dela deixaria a fatura inflada sem representar
	// a compra (mesma regra do lançamento manual, handleCreditCardMovement).
	var seriesTotal float64
	for _, item := range items {
		seriesTotal += item.delta
	}
	if !guard.allows(seriesTotal) {
		return created, skipped, errors, ErrInsufficientCreditLimit
	}

	if err := u.persistInvoiceMovements(ctx, creditCardID, items); err != nil {
		log.Debug(
			"confirm invoice: skipped installment series — persist error",
			log.String("description", baseMovement.Description),
			log.Int("installments", len(items)),
			log.Err(err),
		)
		errors = append(errors, fmt.Sprintf("Could not save installments of '%s': internal system error", baseMovement.Description))
		return created, skipped + len(items), errors, nil
	}
	guard.consume(seriesTotal)

	for _, hash := range hashes {
		existingHashes[hash] = true
	}

	return created + len(items), skipped, errors, nil
}

// saveSingleInvoiceMovement persiste um único movimento de cartão (sem parcelamento),
// atualizando o total da fatura e o limite do cartão na mesma transação.
func (u *StatementUseCase) saveSingleInvoiceMovement(
	ctx context.Context,
	guard *creditLimitGuard,
	creditCardID uuid.UUID,
	invoice domain.Invoice,
	movement domain.Movement,
) (ok bool, errMsg string, abort error) {
	if !guard.allows(movement.Amount) {
		return false, "", ErrInsufficientCreditLimit
	}

	err := u.persistInvoiceMovements(ctx, creditCardID, []invoiceMovementItem{
		{movement: movement, invoice: invoice, delta: movement.Amount},
	})
	if err == nil {
		guard.consume(movement.Amount)
		return true, "", nil
	}

	userReason := "internal system error"
	if domain.Is(err, domain.ErrInvalidInput) {
		userReason = "invalid data"
	} else if domain.Is(err, domain.ErrConflict) {
		userReason = "duplicate entry"
	}
	log.Debug(
		"confirm invoice: skipped movement — persist error",
		log.String("description", movement.Description),
		log.Float64("amount", movement.Amount),
		log.String("reason", userReason),
		log.Err(err),
	)
	return false, fmt.Sprintf("Could not save '%s': %s", movement.Description, userReason), nil
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
	uncategorizedIncomeID := uuid.MustParse(domain.UncategorizedIncomeCategoryID)

	// 3. Filter and insert only new movements
	var created, skipped int
	var errorsList []string

	for i, m := range input.Movements {
		categoryID := resolveCategoryID(m.CategoryID, m.Amount, uncategorizedID, uncategorizedIncomeID)

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

func resolveCategoryID(provided *uuid.UUID, amount float64, expenseFallbackID, incomeFallbackID uuid.UUID) uuid.UUID {
	if provided != nil {
		return *provided
	}
	if amount > 0 {
		return incomeFallbackID
	}
	return expenseFallbackID
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
