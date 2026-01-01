package model

import (
	"time"

	"github.com/google/uuid"
)

type (
	WalletOutput struct {
		ID             *uuid.UUID `json:"id,omitempty" gorm:"primaryKey"`
		Description    string     `json:"description,omitempty"`
		Balance        float64    `json:"balance"`
		InitialBalance float64    `json:"initial_balance"`
		InitialDate    time.Time  `json:"initial_date"`
	}

	TypePaymentOutput string

	CategoryOutput struct {
		ID            *uuid.UUID            `json:"id,omitempty" gorm:"primaryKey"`
		Description   string                `json:"description,omitempty"`
		Color         string                `json:"color,omitempty"`
		IsIncome      bool                  `json:"is_income"`
		SubCategories SubCategoryListOutput `json:"sub_categories,omitempty"`
	}

	SubCategoryOutput struct {
		ID          *uuid.UUID `json:"id,omitempty" gorm:"primaryKey"`
		Description string     `json:"description,omitempty"`
	}

	SubCategoryListOutput []SubCategoryOutput

	TransactionStatusOutput struct {
		ID          int    `json:"id,omitempty" gorm:"primaryKey"`
		Description string `json:"description,omitempty"`
	}

	MovementOutput struct {
		ID          *uuid.UUID        `json:"id,omitempty" gorm:"primaryKey"`
		Description string            `json:"description,omitempty"`
		Amount      float64           `json:"amount"`
		Date        *time.Time        `json:"date,omitempty"`
		IsPaid      bool              `json:"is_paid"`
		IsRecurrent bool              `json:"is_recurrent"`
		RecurrentID *uuid.UUID        `json:"recurrent_id"`
		Wallet      WalletOutput      `json:"wallet,omitempty"`
		TypePayment TypePaymentOutput `json:"type_payment,omitempty"`
		Category    CategoryOutput    `json:"category,omitempty"`
		SubCategory SubCategoryOutput `json:"sub_category,omitempty"`
		DateUpdate  *time.Time        `json:"date_update,omitempty"`
	}

	MovementListOutput []MovementOutput

	BalanceOutput struct {
		Expense       float64 `json:"expense"`
		Income        float64 `json:"income"`
		PeriodBalance float64 `json:"period_balance"`
	}

	OutputEstimateCategories struct {
		ID                    *uuid.UUID                    `json:"id" gorm:"primaryKey"`
		CategoryID            *uuid.UUID                    `json:"category_id"`
		CategoryName          string                        `json:"category_name"`
		IsCategoryIncome      bool                          `json:"is_category_income"`
		Month                 time.Month                    `json:"month"`
		Year                  int                           `json:"year"`
		Amount                float64                       `json:"amount"`
		EstimateSubCategories []OutputEstimateSubCategories `json:"estimates_sub_categories"`
	}

	OutputEstimateSubCategories struct {
		ID              *uuid.UUID `json:"id" gorm:"primaryKey"`
		SubCategoryID   *uuid.UUID `json:"sub_category_id"`
		SubCategoryName string     `json:"sub_category_name"`
		Month           time.Month `json:"month"`
		Year            int        `json:"year"`
		Amount          float64    `json:"amount"`
	}
)

func ToMovementOutput(input *Movement) *MovementOutput {
	output := &MovementOutput{
		ID:          input.ID,
		Description: input.Description,
		Amount:      input.Amount,
		Date:        input.Date,
		IsPaid:      input.IsPaid,
		IsRecurrent: input.IsRecurrent,
		RecurrentID: input.RecurrentID,
		Wallet:      ToWalletOutput(input.Wallet),
		TypePayment: ToTypePaymentOutput(input.TypePayment),
		Category:    ToCategoryOutput(input.Category),
		SubCategory: ToSubCategoryOutput(input.SubCategory),
		DateUpdate:  &input.DateUpdate,
	}
	return output
}

func ToWalletOutput(input Wallet) WalletOutput {
	return WalletOutput{
		ID:             input.ID,
		Description:    input.Description,
		Balance:        input.Balance,
		InitialBalance: input.InitialBalance,
		InitialDate:    input.InitialDate.Truncate(time.Second),
	}
}

func ToTypePaymentOutput(input TypePayment) TypePaymentOutput {
	return TypePaymentOutput(input)
}

func ToCategoryOutput(input Category) CategoryOutput {
	var subCategoriesOutput SubCategoryListOutput
	for _, subCategory := range input.SubCategories {
		subCategoriesOutput = append(subCategoriesOutput, ToSubCategoryOutput(subCategory))
	}
	return CategoryOutput{
		ID:            input.ID,
		Description:   input.Description,
		Color:         input.Color,
		IsIncome:      input.IsIncome,
		SubCategories: subCategoriesOutput,
	}
}

func ToSubCategoryOutput(input SubCategory) SubCategoryOutput {
	return SubCategoryOutput{
		ID:          input.ID,
		Description: input.Description,
	}
}

func ToBalanceOutput(input Balance) BalanceOutput {
	return BalanceOutput{
		Expense:       input.Expense,
		Income:        input.Income,
		PeriodBalance: input.PeriodBalance,
	}
}

func ToOutputEstimateCategories(input EstimateCategories) OutputEstimateCategories {
	return OutputEstimateCategories{
		ID:               input.ID,
		CategoryID:       input.CategoryID,
		CategoryName:     input.CategoryName,
		IsCategoryIncome: input.IsCategoryIncome,
		Month:            input.Month,
		Year:             input.Year,
		Amount:           input.Amount,
	}
}

func ToOutputEstimateSubCategories(input EstimateSubCategories) OutputEstimateSubCategories {
	return OutputEstimateSubCategories{
		ID:              input.ID,
		SubCategoryID:   input.SubCategoryID,
		SubCategoryName: input.SubCategoryName,
		Month:           input.Month,
		Year:            input.Year,
		Amount:          input.Amount,
	}
}
