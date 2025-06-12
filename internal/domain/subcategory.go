package domain

import (
	"time"

	"github.com/google/uuid"
)

type SubCategory struct {
	ID          *uuid.UUID `json:"id,omitempty" gorm:"primaryKey"`
	Description string     `json:"description,omitempty"`
	UserID      string     `json:"user_id"`
	CategoryID  *uuid.UUID `json:"category_id,omitempty"`
	DateCreate  time.Time  `json:"date_create"`
	DateUpdate  time.Time  `json:"date_update"`
}

type SubCategoryList []SubCategory
