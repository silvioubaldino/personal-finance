package estimate

import "errors"

var (
	ErrMonthCategoryEstimateExists    = errors.New("month category estimate already exists")
	ErrMonthSubCategoryEstimateExists = errors.New("month sub category estimate already exists")
	ErrSubCategoryNotInCategory       = errors.New("sub category not in category")
)
