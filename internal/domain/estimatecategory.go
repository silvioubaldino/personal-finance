package domain

import (
	"time"

	"github.com/google/uuid"
)

type EstimateCategories struct {
	ID               *uuid.UUID              `json:"id" gorm:"primaryKey"`
	CategoryID       *uuid.UUID              `json:"category_id"`
	CategoryName     string                  `json:"category_name"`
	IsCategoryIncome bool                    `json:"is_category_income"`
	Month            time.Month              `json:"month"`
	Year             int                     `json:"year"`
	Amount           float64                 `json:"amount"`
	UserID           string                  `json:"user_id"`
	SubCategories    []EstimateSubCategories `json:"-" gorm:"-"`
}

type EstimateCategoriesList []EstimateCategories

func (el EstimateCategoriesList) GetEstimateByCategory() map[*uuid.UUID]float64 {
	m := make(map[*uuid.UUID]float64)
	for _, estimate := range el {
		if _, ok := m[estimate.CategoryID]; !ok {
			m[estimate.CategoryID] = estimate.Amount
		} else {
			m[estimate.CategoryID] += estimate.Amount
		}
	}
	return m
}

func (el EstimateCategoriesList) GetExpenseEstimates() EstimateCategoriesList {
	var expenseList EstimateCategoriesList
	for _, estimate := range el {
		if estimate.Amount < 0 {
			expenseList = append(expenseList, estimate)
		}
	}
	return expenseList
}

func (el EstimateCategoriesList) GetIncomeEstimates() EstimateCategoriesList {
	var expenseList EstimateCategoriesList
	for _, estimate := range el {
		if estimate.Amount > 0 {
			expenseList = append(expenseList, estimate)
		}
	}
	return expenseList
}
