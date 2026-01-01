package output

import (
	"personal-finance/internal/domain"

	"github.com/google/uuid"
)

type CategoryOutput struct {
	ID            *uuid.UUID            `json:"id,omitempty"`
	Description   string                `json:"description,omitempty"`
	Color         string                `json:"color,omitempty"`
	IsIncome      bool                  `json:"is_income"`
	SubCategories SubCategoryListOutput `json:"sub_categories,omitempty"`
}

type SubCategoryListOutput []SubCategoryOutput

func ToCategoryOutput(input domain.Category) CategoryOutput {
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
