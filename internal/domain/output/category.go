package output

import (
	"personal-finance/internal/domain"

	"github.com/google/uuid"
)

// defaultCategoryUserID identifies system-owned categories/subcategories
// seeded for every user (mirrors repository.DefaultCategoryUserID).
const defaultCategoryUserID = "default_category_id"

type CategoryOutput struct {
	ID            *uuid.UUID            `json:"id,omitempty"`
	Description   string                `json:"description,omitempty"`
	Color         string                `json:"color,omitempty"`
	IsIncome      bool                  `json:"is_income"`
	IsDefault     bool                  `json:"is_default"`
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
		IsDefault:     input.UserID == defaultCategoryUserID,
		SubCategories: subCategoriesOutput,
	}
}
