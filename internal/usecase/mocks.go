package usecase

import (
	"context"

	"personal-finance/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

type MockMovementRepository struct {
	mock.Mock
}

func (m *MockMovementRepository) Add(_ context.Context, tx *gorm.DB, movement domain.Movement) (domain.Movement, error) {
	args := m.Called(tx, movement)
	return args.Get(0).(domain.Movement), args.Error(1)
}

type MockRecurrentRepository struct {
	mock.Mock
}

func (m *MockRecurrentRepository) Add(_ context.Context, tx *gorm.DB, recurrent domain.RecurrentMovement) (domain.RecurrentMovement, error) {
	args := m.Called(tx, recurrent)
	return args.Get(0).(domain.RecurrentMovement), args.Error(1)
}

type MockWalletRepository struct {
	mock.Mock
}

func (m *MockWalletRepository) Add(_ context.Context, wallet domain.Wallet) (domain.Wallet, error) {
	args := m.Called(wallet)
	return args.Get(0).(domain.Wallet), args.Error(1)
}

func (m *MockWalletRepository) AddConsistent(_ context.Context, tx *gorm.DB, wallet domain.Wallet) (domain.Wallet, error) {
	args := m.Called(tx, wallet)
	return args.Get(0).(domain.Wallet), args.Error(1)
}

func (m *MockWalletRepository) FindByID(_ context.Context, id *uuid.UUID) (domain.Wallet, error) {
	args := m.Called(id)
	return args.Get(0).(domain.Wallet), args.Error(1)
}

func (m *MockWalletRepository) FindAll(_ context.Context) ([]domain.Wallet, error) {
	args := m.Called()
	return args.Get(0).([]domain.Wallet), args.Error(1)
}

func (m *MockWalletRepository) Update(_ context.Context, wallet domain.Wallet) (domain.Wallet, error) {
	args := m.Called(wallet)
	return args.Get(0).(domain.Wallet), args.Error(1)
}

func (m *MockWalletRepository) UpdateConsistent(_ context.Context, tx *gorm.DB, wallet domain.Wallet) error {
	args := m.Called(tx, wallet)
	return args.Error(0)
}

func (m *MockWalletRepository) Delete(_ context.Context, id *uuid.UUID) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockWalletRepository) RecalculateBalance(_ context.Context, walletID *uuid.UUID) error {
	args := m.Called(walletID)
	return args.Error(0)
}

type MockSubCategory struct {
	mock.Mock
}

func (m *MockSubCategory) Add(_ context.Context, subcategory domain.SubCategory) (domain.SubCategory, error) {
	args := m.Called(subcategory)
	return args.Get(0).(domain.SubCategory), args.Error(1)
}

func (m *MockSubCategory) FindAll(_ context.Context) (domain.SubCategoryList, error) {
	args := m.Called()
	return args.Get(0).(domain.SubCategoryList), args.Error(1)
}

func (m *MockSubCategory) FindByID(_ context.Context, id uuid.UUID) (domain.SubCategory, error) {
	args := m.Called(id)
	return args.Get(0).(domain.SubCategory), args.Error(1)
}

func (m *MockSubCategory) FindByCategoryID(_ context.Context, categoryID uuid.UUID) (domain.SubCategoryList, error) {
	args := m.Called(categoryID)
	return args.Get(0).(domain.SubCategoryList), args.Error(1)
}

func (m *MockSubCategory) IsSubCategoryBelongsToCategory(_ context.Context, subCategoryID uuid.UUID, categoryID uuid.UUID) (bool, error) {
	args := m.Called(subCategoryID, categoryID)
	return args.Bool(0), args.Error(1)
}

func (m *MockSubCategory) Update(_ context.Context, subcategory domain.SubCategory) (domain.SubCategory, error) {
	args := m.Called(subcategory)
	return args.Get(0).(domain.SubCategory), args.Error(1)
}

func (m *MockSubCategory) Delete(_ context.Context, id uuid.UUID) error {
	args := m.Called(id)
	return args.Error(0)
}

type MockTransactionManager struct {
	mock.Mock
}

func (m *MockTransactionManager) WithTransaction(_ context.Context, fn func(tx *gorm.DB) error) error {
	args := m.Called(fn)

	if len(args) > 0 && args.Get(0) == nil {
		txFunc := fn
		_ = txFunc(nil)
	}

	return args.Error(0)
}
