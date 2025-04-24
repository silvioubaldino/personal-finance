package fixture

import (
	"time"

	"personal-finance/internal/domain"

	"github.com/google/uuid"
)

var (
	SCFixedTime       = time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC)
	SCID              = uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	SCCategoryID      = uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	SCOtherCategoryID = uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
)

type SubCategoryMockOption func(s *domain.SubCategory)

func SubCategoryMock(options ...SubCategoryMockOption) domain.SubCategory {
	s := domain.SubCategory{
		ID:          &SCID,
		Description: "Subcategoria de teste",
		UserID:      "user-test-id",
		CategoryID:  &SCCategoryID,
		DateCreate:  SCFixedTime,
		DateUpdate:  SCFixedTime,
	}

	for _, opt := range options {
		opt(&s)
	}

	return s
}

func WithSubCategoryID(id uuid.UUID) SubCategoryMockOption {
	return func(s *domain.SubCategory) {
		s.ID = &id
	}
}

func WithSubCategoryDescription(description string) SubCategoryMockOption {
	return func(s *domain.SubCategory) {
		s.Description = description
	}
}

func WithSubCategoryUserID(userID string) SubCategoryMockOption {
	return func(s *domain.SubCategory) {
		s.UserID = userID
	}
}

func WithSubCategoryCategoryID(categoryID uuid.UUID) SubCategoryMockOption {
	return func(s *domain.SubCategory) {
		s.CategoryID = &categoryID
	}
}

func WithSubCategoryDateCreate(dateCreate time.Time) SubCategoryMockOption {
	return func(s *domain.SubCategory) {
		s.DateCreate = dateCreate
	}
}

func WithSubCategoryDateUpdate(dateUpdate time.Time) SubCategoryMockOption {
	return func(s *domain.SubCategory) {
		s.DateUpdate = dateUpdate
	}
}
