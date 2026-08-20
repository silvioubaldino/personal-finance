package output

import (
	"personal-finance/internal/domain"

	"github.com/google/uuid"
)

type SubCategoryOutput struct {
	ID          *uuid.UUID `json:"id,omitempty"`
	Description string     `json:"description,omitempty"`
	IsDefault   bool       `json:"is_default"`
}

func ToSubCategoryOutput(input domain.SubCategory) SubCategoryOutput {
	return SubCategoryOutput{
		ID:          input.ID,
		Description: input.Description,
		IsDefault:   input.UserID == defaultCategoryUserID,
	}
}
