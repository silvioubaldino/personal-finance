package domain

import (
	"time"

	"github.com/google/uuid"
)

type EstimateSubCategories struct {
	ID                 *uuid.UUID `json:"id" gorm:"primaryKey"`
	SubCategoryID      *uuid.UUID `json:"sub_category_id"`
	SubCategoryName    string     `json:"sub_category_name"`
	EstimateCategoryID *uuid.UUID `json:"estimate_category_id"`
	Month              time.Month `json:"month"`
	Year               int        `json:"year"`
	Amount             float64    `json:"amount"`
	UserID             string     `json:"user_id"`
}
