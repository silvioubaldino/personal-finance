package output

import (
	"personal-finance/internal/domain"

	"github.com/google/uuid"
)

type EstimateTotalsBucketOutput struct {
	Budgeted     float64 `json:"budgeted"`
	Realized     float64 `json:"realized"`
	RealizedPaid float64 `json:"realized_paid"`
	Consolidated float64 `json:"consolidated"`
}

type EstimateSummaryTotalsOutput struct {
	Income        EstimateTotalsBucketOutput `json:"income"`
	Expense       EstimateTotalsBucketOutput `json:"expense"`
	PeriodBalance float64                    `json:"period_balance"`
}

type EstimateSummarySubCategoryOutput struct {
	SubEstimateID   *uuid.UUID `json:"sub_estimate_id"`
	SubCategoryID   *uuid.UUID `json:"sub_category_id"`
	SubCategoryName string     `json:"sub_category_name"`
	IsPlanned       bool       `json:"is_planned"`
	Budgeted        float64    `json:"budgeted"`
	Realized        float64    `json:"realized"`
	RealizedPaid    float64    `json:"realized_paid"`
	Result          float64    `json:"result"`
	Progress        *float64   `json:"progress"`
}

type EstimateSummaryCategoryOutput struct {
	EstimateID    *uuid.UUID                         `json:"estimate_id"`
	CategoryID    *uuid.UUID                         `json:"category_id"`
	CategoryName  string                             `json:"category_name"`
	IsIncome      bool                               `json:"is_income"`
	IsPlanned     bool                               `json:"is_planned"`
	Budgeted      float64                            `json:"budgeted"`
	Realized      float64                            `json:"realized"`
	RealizedPaid  float64                            `json:"realized_paid"`
	Result        float64                            `json:"result"`
	Progress      *float64                           `json:"progress"`
	SubCategories []EstimateSummarySubCategoryOutput `json:"subcategories"`
}

type EstimateSummaryOutput struct {
	Month      int                             `json:"month"`
	Year       int                             `json:"year"`
	Totals     EstimateSummaryTotalsOutput     `json:"totals"`
	Categories []EstimateSummaryCategoryOutput `json:"categories"`
}

func ToEstimateSummaryOutput(input domain.EstimateSummary) EstimateSummaryOutput {
	categories := make([]EstimateSummaryCategoryOutput, len(input.Categories))
	for i, category := range input.Categories {
		categories[i] = toEstimateSummaryCategoryOutput(category)
	}

	return EstimateSummaryOutput{
		Month: input.Month,
		Year:  input.Year,
		Totals: EstimateSummaryTotalsOutput{
			Income:        toEstimateTotalsBucketOutput(input.Totals.Income),
			Expense:       toEstimateTotalsBucketOutput(input.Totals.Expense),
			PeriodBalance: input.Totals.PeriodBalance,
		},
		Categories: categories,
	}
}

func toEstimateTotalsBucketOutput(input domain.EstimateTotalsBucket) EstimateTotalsBucketOutput {
	return EstimateTotalsBucketOutput{
		Budgeted:     input.Budgeted,
		Realized:     input.Realized,
		RealizedPaid: input.RealizedPaid,
		Consolidated: input.Consolidated,
	}
}

func toEstimateSummaryCategoryOutput(input domain.EstimateSummaryCategory) EstimateSummaryCategoryOutput {
	subCategories := make([]EstimateSummarySubCategoryOutput, len(input.SubCategories))
	for i, subCategory := range input.SubCategories {
		subCategories[i] = toEstimateSummarySubCategoryOutput(subCategory)
	}

	return EstimateSummaryCategoryOutput{
		EstimateID:    input.EstimateID,
		CategoryID:    input.CategoryID,
		CategoryName:  input.CategoryName,
		IsIncome:      input.IsIncome,
		IsPlanned:     input.IsPlanned,
		Budgeted:      input.Budgeted,
		Realized:      input.Realized,
		RealizedPaid:  input.RealizedPaid,
		Result:        input.Result,
		Progress:      input.Progress,
		SubCategories: subCategories,
	}
}

func toEstimateSummarySubCategoryOutput(input domain.EstimateSummarySubCategory) EstimateSummarySubCategoryOutput {
	return EstimateSummarySubCategoryOutput{
		SubEstimateID:   input.SubEstimateID,
		SubCategoryID:   input.SubCategoryID,
		SubCategoryName: input.SubCategoryName,
		IsPlanned:       input.IsPlanned,
		Budgeted:        input.Budgeted,
		Realized:        input.Realized,
		RealizedPaid:    input.RealizedPaid,
		Result:          input.Result,
		Progress:        input.Progress,
	}
}
