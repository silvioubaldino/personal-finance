package domain

import (
	"time"

	"github.com/google/uuid"
)

type Category struct {
	ID            *uuid.UUID      `json:"id,omitempty" gorm:"primaryKey"`
	Description   string          `json:"description,omitempty"`
	UserID        string          `json:"user_id"`
	Color         string          `json:"color,omitempty"`
	IsIncome      bool            `json:"is_income"`
	SubCategories SubCategoryList `json:"sub_categories"`
	DateCreate    time.Time       `json:"date_create"`
	DateUpdate    time.Time       `json:"date_update"`
}
