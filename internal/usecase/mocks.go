package usecase

import (
	"context"
	"time"

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

func (m *MockMovementRepository) FindByPeriod(_ context.Context, period domain.Period) (domain.MovementList, error) {
	args := m.Called(period)
	return args.Get(0).(domain.MovementList), args.Error(1)
}

func (m *MockMovementRepository) UpdateIsPaid(_ context.Context, tx *gorm.DB, id uuid.UUID, movement domain.Movement) (domain.Movement, error) {
	args := m.Called(tx, id, movement)
	return args.Get(0).(domain.Movement), args.Error(1)
}

func (m *MockMovementRepository) FindByID(_ context.Context, id uuid.UUID) (domain.Movement, error) {
	args := m.Called(id)
	return args.Get(0).(domain.Movement), args.Error(1)
}

func (m *MockMovementRepository) FindByInvoiceID(_ context.Context, invoiceID uuid.UUID) (domain.MovementList, error) {
	args := m.Called(invoiceID)
	return args.Get(0).(domain.MovementList), args.Error(1)
}

func (m *MockMovementRepository) UpdateOne(_ context.Context, tx *gorm.DB, id uuid.UUID, movement domain.Movement) (domain.Movement, error) {
	args := m.Called(tx, id, movement)
	return args.Get(0).(domain.Movement), args.Error(1)
}

func (m *MockMovementRepository) DeleteByInvoiceID(_ context.Context, tx *gorm.DB, invoiceID uuid.UUID) error {
	args := m.Called(tx, invoiceID)
	return args.Error(0)
}

type MockRecurrentRepository struct {
	mock.Mock
}

func (m *MockRecurrentRepository) Add(_ context.Context, tx *gorm.DB, recurrent domain.RecurrentMovement) (domain.RecurrentMovement, error) {
	args := m.Called(tx, recurrent)
	return args.Get(0).(domain.RecurrentMovement), args.Error(1)
}

func (m *MockRecurrentRepository) FindByMonth(_ context.Context, month time.Time) ([]domain.RecurrentMovement, error) {
	args := m.Called(month)
	return args.Get(0).([]domain.RecurrentMovement), args.Error(1)
}

func (m *MockRecurrentRepository) FindByID(_ context.Context, id uuid.UUID) (domain.RecurrentMovement, error) {
	args := m.Called(id)
	return args.Get(0).(domain.RecurrentMovement), args.Error(1)
}

func (m *MockRecurrentRepository) Update(_ context.Context, tx *gorm.DB, id *uuid.UUID, newRecurrent domain.RecurrentMovement) (domain.RecurrentMovement, error) {
	args := m.Called(tx, id, newRecurrent)
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

func (m *MockWalletRepository) UpdateAmount(_ context.Context, tx *gorm.DB, walletID *uuid.UUID, amout float64) error {
	args := m.Called(tx, walletID, amout)
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
		if err := txFunc(nil); err != nil {
			return err
		}
	}

	return args.Error(0)
}

type MockCreditCardRepository struct {
	mock.Mock
}

func (m *MockCreditCardRepository) Add(_ context.Context, tx *gorm.DB, creditCard domain.CreditCard) (domain.CreditCard, error) {
	args := m.Called(tx, creditCard)
	return args.Get(0).(domain.CreditCard), args.Error(1)
}

func (m *MockCreditCardRepository) FindAll(_ context.Context) ([]domain.CreditCard, error) {
	args := m.Called()
	return args.Get(0).([]domain.CreditCard), args.Error(1)
}

func (m *MockCreditCardRepository) FindByID(_ context.Context, id uuid.UUID) (domain.CreditCard, error) {
	args := m.Called(id)
	return args.Get(0).(domain.CreditCard), args.Error(1)
}

func (m *MockCreditCardRepository) FindNameByID(_ context.Context, id uuid.UUID) (string, error) {
	args := m.Called(id)
	return args.String(0), args.Error(1)
}

func (m *MockCreditCardRepository) Update(_ context.Context, tx *gorm.DB, id uuid.UUID, creditCard domain.CreditCard) (domain.CreditCard, error) {
	args := m.Called(tx, id, creditCard)
	return args.Get(0).(domain.CreditCard), args.Error(1)
}

func (m *MockCreditCardRepository) Delete(_ context.Context, tx *gorm.DB, id uuid.UUID) error {
	args := m.Called(tx, id)
	return args.Error(0)
}

type MockInvoiceRepository struct {
	mock.Mock
}

func (m *MockInvoiceRepository) Add(_ context.Context, tx *gorm.DB, invoice domain.Invoice) (domain.Invoice, error) {
	args := m.Called(tx, invoice)
	return args.Get(0).(domain.Invoice), args.Error(1)
}

func (m *MockInvoiceRepository) FindByID(_ context.Context, id uuid.UUID) (domain.Invoice, error) {
	args := m.Called(id)
	return args.Get(0).(domain.Invoice), args.Error(1)
}

func (m *MockInvoiceRepository) FindByPeriod(_ context.Context, period domain.Period) ([]domain.Invoice, error) {
	args := m.Called(period)
	return args.Get(0).([]domain.Invoice), args.Error(1)
}

func (m *MockInvoiceRepository) FindOpenByMonth(_ context.Context, date time.Time) ([]domain.Invoice, error) {
	args := m.Called(date)
	return args.Get(0).([]domain.Invoice), args.Error(1)
}

func (m *MockInvoiceRepository) FindByMonth(_ context.Context, date time.Time) ([]domain.Invoice, error) {
	args := m.Called(date)
	return args.Get(0).([]domain.Invoice), args.Error(1)
}

func (m *MockInvoiceRepository) FindByMonthAndCreditCard(_ context.Context, date time.Time, creditCardID uuid.UUID) (domain.Invoice, error) {
	args := m.Called(date, creditCardID)
	return args.Get(0).(domain.Invoice), args.Error(1)
}

func (m *MockInvoiceRepository) FindOpenByCreditCard(_ context.Context, creditCardID uuid.UUID) ([]domain.Invoice, error) {
	args := m.Called(creditCardID)
	return args.Get(0).([]domain.Invoice), args.Error(1)
}

func (m *MockInvoiceRepository) UpdateAmount(_ context.Context, tx *gorm.DB, id uuid.UUID, amount float64) (domain.Invoice, error) {
	args := m.Called(tx, id, amount)
	return args.Get(0).(domain.Invoice), args.Error(1)
}

func (m *MockInvoiceRepository) UpdateStatus(ctx context.Context, tx *gorm.DB, id uuid.UUID, isPaid bool, paymentDate *time.Time, walletID *uuid.UUID) (domain.Invoice, error) {
	args := m.Called(ctx, tx, id, isPaid, paymentDate, walletID)
	return args.Get(0).(domain.Invoice), args.Error(1)
}

type MockInvoice struct {
	mock.Mock
}

func (m *MockInvoice) FindOrCreateInvoiceForMovement(ctx context.Context, invoiceID *uuid.UUID, creditCardID *uuid.UUID, movementDate time.Time) (domain.Invoice, error) {
	args := m.Called(ctx, invoiceID, creditCardID, movementDate)
	return args.Get(0).(domain.Invoice), args.Error(1)
}
