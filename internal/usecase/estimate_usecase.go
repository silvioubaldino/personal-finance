package usecase

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"personal-finance/internal/domain"

	"github.com/google/uuid"
)

type (
	EstimateRepository interface {
		FindCategoriesByMonth(ctx context.Context, month int, year int) ([]domain.EstimateCategories, error)
		FindSubcategoriesByMonth(ctx context.Context, month int, year int) ([]domain.EstimateSubCategories, error)
		FindCategoryByID(ctx context.Context, id uuid.UUID) (domain.EstimateCategories, error)
		FindSubCategoryByID(ctx context.Context, id uuid.UUID) (domain.EstimateSubCategories, error)
		AddEstimateCategory(ctx context.Context, category domain.EstimateCategories) (domain.EstimateCategories, error)
		AddEstimateSubCategory(ctx context.Context, subEstimate domain.EstimateSubCategories) (domain.EstimateSubCategories, error)
		UpdateEstimateCategoryAmount(ctx context.Context, id *uuid.UUID, amount float64) (domain.EstimateCategories, error)
		UpdateEstimateSubCategoryAmount(ctx context.Context, id *uuid.UUID, amount float64) (domain.EstimateSubCategories, error)
		DeleteEstimateCategory(ctx context.Context, id *uuid.UUID) error
		DeleteEstimateSubCategory(ctx context.Context, id *uuid.UUID) error
	}

	EstimateCategoryRepository interface {
		FindByID(ctx context.Context, id uuid.UUID) (domain.Category, error)
	}

	EstimateSubCategoryRepository interface {
		FindByID(ctx context.Context, id uuid.UUID) (domain.SubCategory, error)
	}

	EstimateMovementRepository interface {
		FindByPeriod(ctx context.Context, period domain.Period) (domain.MovementList, error)
	}

	EstimateInvoiceUseCase interface {
		FindDetailedInvoicesByPeriod(ctx context.Context, period domain.Period) ([]domain.DetailedInvoice, error)
	}

	Estimate interface {
		FindByMonth(ctx context.Context, month int, year int) ([]domain.EstimateCategories, error)
		FindSummary(ctx context.Context, month int, year int) (domain.EstimateSummary, error)
		AddEstimateCategory(ctx context.Context, category domain.EstimateCategories) (domain.EstimateCategories, error)
		AddEstimateSubCategory(ctx context.Context, subEstimate domain.EstimateSubCategories) (domain.EstimateSubCategories, error)
		UpdateEstimateCategoryAmount(ctx context.Context, id *uuid.UUID, amount float64) (domain.EstimateCategories, error)
		UpdateEstimateSubCategoryAmount(ctx context.Context, id *uuid.UUID, amount float64) (domain.EstimateSubCategories, error)
		DeleteEstimateCategory(ctx context.Context, id *uuid.UUID) error
		DeleteEstimateSubCategory(ctx context.Context, id *uuid.UUID) error
	}

	estimateUseCase struct {
		repo            EstimateRepository
		categoryRepo    EstimateCategoryRepository
		subCategoryRepo EstimateSubCategoryRepository
		movementRepo    EstimateMovementRepository
		invoiceUseCase  EstimateInvoiceUseCase
	}
)

func NewEstimate(
	repo EstimateRepository,
	categoryRepo EstimateCategoryRepository,
	subCategoryRepo EstimateSubCategoryRepository,
	movementRepo EstimateMovementRepository,
	invoiceUseCase EstimateInvoiceUseCase,
) Estimate {
	return estimateUseCase{
		repo:            repo,
		categoryRepo:    categoryRepo,
		subCategoryRepo: subCategoryRepo,
		movementRepo:    movementRepo,
		invoiceUseCase:  invoiceUseCase,
	}
}

func (uc estimateUseCase) FindByMonth(ctx context.Context, month int, year int) ([]domain.EstimateCategories, error) {
	estimateCategories, err := uc.repo.FindCategoriesByMonth(ctx, month, year)
	if err != nil {
		return nil, fmt.Errorf("error finding estimate categories by month: %w", err)
	}

	estimateSubCategories, err := uc.repo.FindSubcategoriesByMonth(ctx, month, year)
	if err != nil {
		return nil, fmt.Errorf("error finding estimate sub categories: %w", err)
	}

	subCategoriesByCategory := make(map[uuid.UUID][]domain.EstimateSubCategories)
	for _, subCategory := range estimateSubCategories {
		if subCategory.EstimateCategoryID != nil {
			subCategoriesByCategory[*subCategory.EstimateCategoryID] = append(
				subCategoriesByCategory[*subCategory.EstimateCategoryID], subCategory)
		}
	}

	for i := range estimateCategories {
		if estimateCategories[i].ID != nil {
			estimateCategories[i].SubCategories = subCategoriesByCategory[*estimateCategories[i].ID]
		}
	}

	return estimateCategories, nil
}

func (uc estimateUseCase) FindSummary(ctx context.Context, month int, year int) (domain.EstimateSummary, error) {
	if month < 1 || month > 12 {
		return domain.EstimateSummary{}, domain.WrapInvalidInput(domain.New("month must be between 1 and 12"), "invalid month")
	}

	period := monthPeriod(month, year)

	movements, err := uc.movementRepo.FindByPeriod(ctx, period)
	if err != nil {
		return domain.EstimateSummary{}, fmt.Errorf("error finding movements by period: %w", err)
	}

	invoices, err := uc.invoiceUseCase.FindDetailedInvoicesByPeriod(ctx, period)
	if err != nil {
		return domain.EstimateSummary{}, fmt.Errorf("error finding invoices by period: %w", err)
	}

	estimateCategories, err := uc.repo.FindCategoriesByMonth(ctx, month, year)
	if err != nil {
		return domain.EstimateSummary{}, fmt.Errorf("error finding estimate categories by month: %w", err)
	}

	estimateSubCategories, err := uc.repo.FindSubcategoriesByMonth(ctx, month, year)
	if err != nil {
		return domain.EstimateSummary{}, fmt.Errorf("error finding estimate sub categories: %w", err)
	}

	realized := canonicalRealizedMovements(movements, invoices)

	return buildEstimateSummary(month, year, realized, estimateCategories, estimateSubCategories), nil
}

func (uc estimateUseCase) AddEstimateCategory(ctx context.Context, category domain.EstimateCategories) (domain.EstimateCategories, error) {
	if category.CategoryID == nil {
		return domain.EstimateCategories{}, domain.WrapInvalidInput(domain.New("category_id is required"), "invalid estimate category")
	}

	cat, err := uc.categoryRepo.FindByID(ctx, *category.CategoryID)
	if err != nil {
		return domain.EstimateCategories{}, fmt.Errorf("error finding category: %w", err)
	}
	category.Amount = normalizeEstimateSign(category.Amount, cat.IsIncome)

	result, err := uc.repo.AddEstimateCategory(ctx, category)
	if err != nil {
		return domain.EstimateCategories{}, fmt.Errorf("error adding estimate category: %w", err)
	}
	return result, nil
}

func (uc estimateUseCase) AddEstimateSubCategory(ctx context.Context, subEstimate domain.EstimateSubCategories) (domain.EstimateSubCategories, error) {
	if subEstimate.SubCategoryID == nil {
		return domain.EstimateSubCategories{}, domain.WrapInvalidInput(domain.New("sub_category_id is required"), "invalid estimate sub category")
	}

	subCat, err := uc.subCategoryRepo.FindByID(ctx, *subEstimate.SubCategoryID)
	if err != nil {
		return domain.EstimateSubCategories{}, fmt.Errorf("error finding sub category: %w", err)
	}

	cat, err := uc.categoryOfSubCategory(ctx, subCat)
	if err != nil {
		return domain.EstimateSubCategories{}, err
	}
	subEstimate.Amount = normalizeEstimateSign(subEstimate.Amount, cat.IsIncome)

	result, err := uc.repo.AddEstimateSubCategory(ctx, subEstimate)
	if err != nil {
		return domain.EstimateSubCategories{}, fmt.Errorf("error adding estimate sub category: %w", err)
	}
	return result, nil
}

func (uc estimateUseCase) UpdateEstimateCategoryAmount(ctx context.Context, id *uuid.UUID, amount float64) (domain.EstimateCategories, error) {
	existing, err := uc.repo.FindCategoryByID(ctx, *id)
	if err != nil {
		return domain.EstimateCategories{}, fmt.Errorf("error finding estimate category: %w", err)
	}
	if existing.CategoryID == nil {
		return domain.EstimateCategories{}, domain.WrapInvalidInput(domain.New("estimate category has no category_id"), "invalid estimate category")
	}

	cat, err := uc.categoryRepo.FindByID(ctx, *existing.CategoryID)
	if err != nil {
		return domain.EstimateCategories{}, fmt.Errorf("error finding category: %w", err)
	}

	result, err := uc.repo.UpdateEstimateCategoryAmount(ctx, id, normalizeEstimateSign(amount, cat.IsIncome))
	if err != nil {
		return domain.EstimateCategories{}, fmt.Errorf("error updating estimate category amount: %w", err)
	}
	return result, nil
}

func (uc estimateUseCase) UpdateEstimateSubCategoryAmount(ctx context.Context, id *uuid.UUID, amount float64) (domain.EstimateSubCategories, error) {
	existing, err := uc.repo.FindSubCategoryByID(ctx, *id)
	if err != nil {
		return domain.EstimateSubCategories{}, fmt.Errorf("error finding estimate sub category: %w", err)
	}
	if existing.SubCategoryID == nil {
		return domain.EstimateSubCategories{}, domain.WrapInvalidInput(domain.New("estimate sub category has no sub_category_id"), "invalid estimate sub category")
	}

	subCat, err := uc.subCategoryRepo.FindByID(ctx, *existing.SubCategoryID)
	if err != nil {
		return domain.EstimateSubCategories{}, fmt.Errorf("error finding sub category: %w", err)
	}

	cat, err := uc.categoryOfSubCategory(ctx, subCat)
	if err != nil {
		return domain.EstimateSubCategories{}, err
	}

	result, err := uc.repo.UpdateEstimateSubCategoryAmount(ctx, id, normalizeEstimateSign(amount, cat.IsIncome))
	if err != nil {
		return domain.EstimateSubCategories{}, fmt.Errorf("error updating estimate sub category amount: %w", err)
	}
	return result, nil
}

func (uc estimateUseCase) DeleteEstimateCategory(ctx context.Context, id *uuid.UUID) error {
	if err := uc.repo.DeleteEstimateCategory(ctx, id); err != nil {
		return fmt.Errorf("error deleting estimate category: %w", err)
	}
	return nil
}

func (uc estimateUseCase) DeleteEstimateSubCategory(ctx context.Context, id *uuid.UUID) error {
	if err := uc.repo.DeleteEstimateSubCategory(ctx, id); err != nil {
		return fmt.Errorf("error deleting estimate sub category: %w", err)
	}
	return nil
}

func (uc estimateUseCase) categoryOfSubCategory(ctx context.Context, subCat domain.SubCategory) (domain.Category, error) {
	if subCat.CategoryID == nil {
		return domain.Category{}, domain.WrapInvalidInput(domain.New("sub category has no category_id"), "invalid sub category")
	}

	cat, err := uc.categoryRepo.FindByID(ctx, *subCat.CategoryID)
	if err != nil {
		return domain.Category{}, fmt.Errorf("error finding category: %w", err)
	}
	return cat, nil
}

func normalizeEstimateSign(amount float64, isIncome bool) float64 {
	abs := math.Abs(amount)
	if isIncome {
		return abs
	}
	return -abs
}

func monthPeriod(month, year int) domain.Period {
	from := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, -1).Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	return domain.Period{From: from, To: to}
}

var internalTransferCategoryIDs = map[uuid.UUID]bool{
	uuid.MustParse(domain.InternalTransferOutCategoryID): true,
	uuid.MustParse(domain.InternalTransferInCategoryID):  true,
}

// isCanonicalRealized diz se um Movement avulso entra no recorte canônico de "realizado"
// (AYD-005@context): internal_transfer e invoice_payment ficam de fora — as compras no
// cartão e o invoice_remainder entram pelos itens da própria Invoice, então contar a fatura
// duplicaria o cartão. Os category_id fixos de transferência interna também saem, para
// pegar as linhas antigas gravadas antes do type_payment existir.
//
// Esta é a única definição do recorte no servidor; AYD-003 (Análises) e AYD-005
// (Planejamentos) leem daqui, não de implementações parecidas.
func isCanonicalRealized(movement domain.Movement) bool {
	if movement.TypePayment == domain.TypePaymentInternalTransfer ||
		movement.TypePayment == domain.TypePaymentInvoicePayment {
		return false
	}
	if movement.CategoryID != nil && internalTransferCategoryIDs[*movement.CategoryID] {
		return false
	}
	// Quem pertence a uma Invoice entra pelos itens dela, nunca pela lista avulsa. Hoje
	// MovementRepository.FindByPeriod já não devolve credit_card nem invoice_remainder, mas
	// depender disso é o que fazia a fatura ser contada duas vezes; aqui a garantia é do
	// recorte, não da query.
	if movement.CreditCardInfo != nil && movement.CreditCardInfo.InvoiceID != nil {
		return false
	}
	return true
}

// canonicalRealizedMovements aplica isCanonicalRealized aos Movements avulsos do período e
// acrescenta os itens das Invoices que vencem nele.
func canonicalRealizedMovements(movements domain.MovementList, invoices []domain.DetailedInvoice) domain.MovementList {
	realized := make(domain.MovementList, 0, len(movements))
	for _, movement := range movements {
		if !isCanonicalRealized(movement) {
			continue
		}
		realized = append(realized, movement)
	}

	for _, invoice := range invoices {
		realized = append(realized, invoice.Movements...)
	}

	return realized
}

type categoryAggregate struct {
	name         string
	isIncome     bool
	realized     float64
	realizedPaid float64
}

type subCategoryAggregate struct {
	name         string
	categoryID   uuid.UUID
	realized     float64
	realizedPaid float64
}

func aggregateRealized(movements domain.MovementList) (map[uuid.UUID]*categoryAggregate, map[uuid.UUID]*subCategoryAggregate) {
	categoryAggs := make(map[uuid.UUID]*categoryAggregate)
	subCategoryAggs := make(map[uuid.UUID]*subCategoryAggregate)

	for _, movement := range movements {
		if movement.CategoryID == nil {
			continue
		}

		catAgg, ok := categoryAggs[*movement.CategoryID]
		if !ok {
			catAgg = &categoryAggregate{name: movement.Category.Description, isIncome: movement.Category.IsIncome}
			categoryAggs[*movement.CategoryID] = catAgg
		}
		catAgg.realized += movement.Amount
		if movement.IsPaid {
			catAgg.realizedPaid += movement.Amount
		}

		if movement.SubCategoryID == nil {
			continue
		}

		subAgg, ok := subCategoryAggs[*movement.SubCategoryID]
		if !ok {
			subAgg = &subCategoryAggregate{name: movement.SubCategory.Description, categoryID: *movement.CategoryID}
			subCategoryAggs[*movement.SubCategoryID] = subAgg
		}
		subAgg.realized += movement.Amount
		if movement.IsPaid {
			subAgg.realizedPaid += movement.Amount
		}
	}

	return categoryAggs, subCategoryAggs
}

// plannedSubCategoryRows builds one row per Estimate sub-category, and reports which
// sub-categories already got a row (so the virtual pass below doesn't duplicate them).
func plannedSubCategoryRows(
	estimateCategories []domain.EstimateCategories,
	estimateSubCategories []domain.EstimateSubCategories,
	subCategoryAggs map[uuid.UUID]*subCategoryAggregate,
) (map[uuid.UUID][]domain.EstimateSummarySubCategory, map[uuid.UUID]bool) {
	categoryIDByEstimateID := make(map[uuid.UUID]uuid.UUID, len(estimateCategories))
	for _, estimateCategory := range estimateCategories {
		if estimateCategory.ID != nil && estimateCategory.CategoryID != nil {
			categoryIDByEstimateID[*estimateCategory.ID] = *estimateCategory.CategoryID
		}
	}

	rowsByCategory := make(map[uuid.UUID][]domain.EstimateSummarySubCategory)
	consumed := make(map[uuid.UUID]bool)

	for _, subEstimate := range estimateSubCategories {
		if subEstimate.EstimateCategoryID == nil {
			continue
		}
		categoryID, ok := categoryIDByEstimateID[*subEstimate.EstimateCategoryID]
		if !ok {
			continue
		}

		var realized, realizedPaid float64
		if subEstimate.SubCategoryID != nil {
			if agg, ok := subCategoryAggs[*subEstimate.SubCategoryID]; ok {
				realized = agg.realized
				realizedPaid = agg.realizedPaid
			}
			consumed[*subEstimate.SubCategoryID] = true
		}

		rowsByCategory[categoryID] = append(rowsByCategory[categoryID], domain.EstimateSummarySubCategory{
			SubEstimateID:   subEstimate.ID,
			SubCategoryID:   subEstimate.SubCategoryID,
			SubCategoryName: subEstimate.SubCategoryName,
			IsPlanned:       true,
			Budgeted:        subEstimate.Amount,
			Realized:        realized,
			RealizedPaid:    realizedPaid,
			Result:          realized - subEstimate.Amount,
			Progress:        progressOf(realized, subEstimate.Amount),
		})
	}

	return rowsByCategory, consumed
}

// virtualSubCategoryRows appends one row per sub-category with movements but no Estimate.
func virtualSubCategoryRows(
	rowsByCategory map[uuid.UUID][]domain.EstimateSummarySubCategory,
	subCategoryAggs map[uuid.UUID]*subCategoryAggregate,
	consumed map[uuid.UUID]bool,
) {
	for subCategoryID, agg := range subCategoryAggs {
		if consumed[subCategoryID] {
			continue
		}
		subCategoryID := subCategoryID

		rowsByCategory[agg.categoryID] = append(rowsByCategory[agg.categoryID], domain.EstimateSummarySubCategory{
			SubCategoryID:   &subCategoryID,
			SubCategoryName: agg.name,
			IsPlanned:       false,
			Budgeted:        0,
			Realized:        agg.realized,
			RealizedPaid:    agg.realizedPaid,
			Result:          agg.realized,
			Progress:        nil,
		})
	}
}

func categoryRows(
	estimateCategories []domain.EstimateCategories,
	categoryAggs map[uuid.UUID]*categoryAggregate,
	subCategoryRowsByCategory map[uuid.UUID][]domain.EstimateSummarySubCategory,
) []domain.EstimateSummaryCategory {
	estimateByCategoryID := make(map[uuid.UUID]domain.EstimateCategories, len(estimateCategories))
	for _, estimateCategory := range estimateCategories {
		if estimateCategory.CategoryID != nil {
			estimateByCategoryID[*estimateCategory.CategoryID] = estimateCategory
		}
	}

	categoryIDs := make(map[uuid.UUID]struct{}, len(estimateByCategoryID)+len(categoryAggs))
	for id := range estimateByCategoryID {
		categoryIDs[id] = struct{}{}
	}
	for id := range categoryAggs {
		categoryIDs[id] = struct{}{}
	}

	categories := make([]domain.EstimateSummaryCategory, 0, len(categoryIDs))
	for categoryID := range categoryIDs {
		categoryID := categoryID
		estimate, isPlanned := estimateByCategoryID[categoryID]
		agg := categoryAggs[categoryID]

		var realized, realizedPaid float64
		if agg != nil {
			realized = agg.realized
			realizedPaid = agg.realizedPaid
		}

		var (
			estimateID *uuid.UUID
			budgeted   float64
			name       string
			isIncome   bool
		)
		switch {
		case isPlanned:
			estimateID = estimate.ID
			budgeted = estimate.Amount
			name = estimate.CategoryName
			isIncome = estimate.IsCategoryIncome
		case agg != nil:
			name = agg.name
			isIncome = agg.isIncome
		}

		subCategories := subCategoryRowsByCategory[categoryID]
		sortSubCategories(subCategories)

		categories = append(categories, domain.EstimateSummaryCategory{
			EstimateID:    estimateID,
			CategoryID:    &categoryID,
			CategoryName:  name,
			IsIncome:      isIncome,
			IsPlanned:     isPlanned,
			Budgeted:      budgeted,
			Realized:      realized,
			RealizedPaid:  realizedPaid,
			Result:        realized - budgeted,
			Progress:      progressOf(realized, budgeted),
			SubCategories: subCategories,
		})
	}

	sortCategories(categories)

	return categories
}

func buildTotals(estimateCategories []domain.EstimateCategories, categoryAggs map[uuid.UUID]*categoryAggregate) domain.EstimateSummaryTotals {
	incomeBudgetMap := make(map[uuid.UUID]float64)
	expenseBudgetMap := make(map[uuid.UUID]float64)

	var incomeBudgeted, expenseBudgeted float64
	for _, estimateCategory := range estimateCategories {
		if estimateCategory.CategoryID == nil {
			continue
		}
		if estimateCategory.IsCategoryIncome {
			incomeBudgetMap[*estimateCategory.CategoryID] += estimateCategory.Amount
			incomeBudgeted += estimateCategory.Amount
		} else {
			expenseBudgetMap[*estimateCategory.CategoryID] += estimateCategory.Amount
			expenseBudgeted += estimateCategory.Amount
		}
	}

	incomeRealizedMap := make(map[uuid.UUID]float64)
	expenseRealizedMap := make(map[uuid.UUID]float64)
	var incomeRealized, incomeRealizedPaid, expenseRealized, expenseRealizedPaid float64

	for categoryID, agg := range categoryAggs {
		if agg.isIncome {
			incomeRealizedMap[categoryID] = agg.realized
			incomeRealized += agg.realized
			incomeRealizedPaid += agg.realizedPaid
		} else {
			expenseRealizedMap[categoryID] = agg.realized
			expenseRealized += agg.realized
			expenseRealizedPaid += agg.realizedPaid
		}
	}

	totals := domain.EstimateSummaryTotals{
		Income: domain.EstimateTotalsBucket{
			Budgeted:     incomeBudgeted,
			Realized:     incomeRealized,
			RealizedPaid: incomeRealizedPaid,
			Consolidated: getBalanceSum(incomeBudgetMap, incomeRealizedMap, true),
		},
		Expense: domain.EstimateTotalsBucket{
			Budgeted:     expenseBudgeted,
			Realized:     expenseRealized,
			RealizedPaid: expenseRealizedPaid,
			Consolidated: getBalanceSum(expenseBudgetMap, expenseRealizedMap, false),
		},
	}
	totals.PeriodBalance = totals.Income.Consolidated + totals.Expense.Consolidated

	return totals
}

func buildEstimateSummary(
	month, year int,
	movements domain.MovementList,
	estimateCategories []domain.EstimateCategories,
	estimateSubCategories []domain.EstimateSubCategories,
) domain.EstimateSummary {
	categoryAggs, subCategoryAggs := aggregateRealized(movements)

	subCategoryRowsByCategory, consumed := plannedSubCategoryRows(estimateCategories, estimateSubCategories, subCategoryAggs)
	virtualSubCategoryRows(subCategoryRowsByCategory, subCategoryAggs, consumed)

	return domain.EstimateSummary{
		Month:      month,
		Year:       year,
		Totals:     buildTotals(estimateCategories, categoryAggs),
		Categories: categoryRows(estimateCategories, categoryAggs, subCategoryRowsByCategory),
	}
}

func progressOf(realized, budgeted float64) *float64 {
	if budgeted == 0 {
		return nil
	}
	progress := math.Abs(realized) / math.Abs(budgeted)
	return &progress
}

func sortCategories(categories []domain.EstimateSummaryCategory) {
	sort.SliceStable(categories, func(i, j int) bool {
		a, b := categories[i], categories[j]
		if a.IsIncome != b.IsIncome {
			return a.IsIncome
		}
		if a.IsPlanned != b.IsPlanned {
			return a.IsPlanned
		}
		return a.CategoryName < b.CategoryName
	})
}

func sortSubCategories(subCategories []domain.EstimateSummarySubCategory) {
	sort.SliceStable(subCategories, func(i, j int) bool {
		a, b := subCategories[i], subCategories[j]
		if a.IsPlanned != b.IsPlanned {
			return a.IsPlanned
		}
		return a.SubCategoryName < b.SubCategoryName
	})
}
