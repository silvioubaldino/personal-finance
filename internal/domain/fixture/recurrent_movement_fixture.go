package fixture

import (
	"time"

	"personal-finance/internal/domain"

	"github.com/google/uuid"
)

var RecurrentMovementID = uuid.MustParse("55555555-5555-5555-5555-555555555555")

type RecurrentMovementMockOption func(rm *domain.RecurrentMovement)

func RecurrentMovementMock(options ...RecurrentMovementMockOption) domain.RecurrentMovement {
	initialDate := now
	endDate := now.AddDate(1, 0, 0)

	rm := domain.RecurrentMovement{
		ID:            &RecurrentMovementID,
		Description:   "Movimento recorrente de teste",
		Amount:        -100.0,
		InitialDate:   &initialDate,
		EndDate:       &endDate,
		UserID:        "user-test-id",
		WalletID:      &WalletID,
		TypePayment:   domain.TypePaymentDebit,
		CategoryID:    &CategoryID,
		SubCategoryID: nil,
	}

	for _, opt := range options {
		opt(&rm)
	}

	return rm
}

func WithRecurrentMovementID(id uuid.UUID) RecurrentMovementMockOption {
	return func(rm *domain.RecurrentMovement) {
		rm.ID = &id
	}
}

func WithRecurrentMovementDescription(description string) RecurrentMovementMockOption {
	return func(rm *domain.RecurrentMovement) {
		rm.Description = description
	}
}

func WithRecurrentMovementAmount(amount float64) RecurrentMovementMockOption {
	return func(rm *domain.RecurrentMovement) {
		rm.Amount = amount
	}
}

func WithRecurrentMovementInitialDate(date time.Time) RecurrentMovementMockOption {
	return func(rm *domain.RecurrentMovement) {
		rm.InitialDate = &date
	}
}

func WithRecurrentMovementEndDate(date time.Time) RecurrentMovementMockOption {
	return func(rm *domain.RecurrentMovement) {
		rm.EndDate = &date
	}
}

func WithRecurrentMovementUserID(userID string) RecurrentMovementMockOption {
	return func(rm *domain.RecurrentMovement) {
		rm.UserID = userID
	}
}

func WithRecurrentMovementWalletID(walletID uuid.UUID) RecurrentMovementMockOption {
	return func(rm *domain.RecurrentMovement) {
		rm.WalletID = &walletID
	}
}

func WithoutRecurrentMovementWallet() RecurrentMovementMockOption {
	return func(rm *domain.RecurrentMovement) {
		rm.WalletID = nil
	}
}

func WithRecurrentMovementTypePayment(typePayment string) RecurrentMovementMockOption {
	return func(rm *domain.RecurrentMovement) {
		rm.TypePayment = domain.TypePayment(typePayment)
	}
}

func WithRecurrentMovementCategoryID(categoryID uuid.UUID) RecurrentMovementMockOption {
	return func(rm *domain.RecurrentMovement) {
		rm.CategoryID = &categoryID
	}
}

func WithRecurrentMovementSubCategoryID(subCategoryID uuid.UUID) RecurrentMovementMockOption {
	return func(rm *domain.RecurrentMovement) {
		rm.SubCategoryID = &subCategoryID
	}
}

func AsRecurrentMovementExpense(amount float64) RecurrentMovementMockOption {
	return func(rm *domain.RecurrentMovement) {
		rm.Amount = -amount
	}
}

func AsRecurrentMovementIncome(amount float64) RecurrentMovementMockOption {
	return func(rm *domain.RecurrentMovement) {
		rm.Amount = amount
	}
}
