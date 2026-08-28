package domain

import "github.com/google/uuid"

type EstimateTotalsBucket struct {
	Budgeted     float64 `json:"budgeted"`
	Realized     float64 `json:"realized"`
	RealizedPaid float64 `json:"realized_paid"`
	Consolidated float64 `json:"consolidated"`
}

type EstimateSummaryTotals struct {
	Income        EstimateTotalsBucket `json:"income"`
	Expense       EstimateTotalsBucket `json:"expense"`
	PeriodBalance float64              `json:"period_balance"`
}

type EstimateSummarySubCategory struct {
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

type EstimateSummaryCategory struct {
	EstimateID    *uuid.UUID                   `json:"estimate_id"`
	CategoryID    *uuid.UUID                   `json:"category_id"`
	CategoryName  string                       `json:"category_name"`
	IsIncome      bool                         `json:"is_income"`
	IsPlanned     bool                         `json:"is_planned"`
	Budgeted      float64                      `json:"budgeted"`
	Realized      float64                      `json:"realized"`
	RealizedPaid  float64                      `json:"realized_paid"`
	Result        float64                      `json:"result"`
	Progress      *float64                     `json:"progress"`
	SubCategories []EstimateSummarySubCategory `json:"subcategories"`
}

type EstimateSummary struct {
	Month      int                       `json:"month"`
	Year       int                       `json:"year"`
	Totals     EstimateSummaryTotals     `json:"totals"`
	Categories []EstimateSummaryCategory `json:"categories"`
}
