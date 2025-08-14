package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/domain/fixture"
	"personal-finance/internal/plataform/authentication"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupInvoiceTestDB() *gorm.DB {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	_ = db.AutoMigrate(&InvoiceDB{}, &CreditCardDB{}, &WalletDB{})

	return db
}

func createInvoiceTestContext() context.Context {
	return context.WithValue(context.Background(), authentication.UserID, "user-test-id")
}

func TestInvoiceRepository_Add(t *testing.T) {
	tests := map[string]struct {
		prepareDB       func() *InvoiceRepository
		input           domain.Invoice
		inputTx         func(repository *InvoiceRepository) *gorm.DB
		expectedErr     error
		expectedInvoice domain.Invoice
	}{
		"should add invoice successfully": {
			prepareDB: func() *InvoiceRepository {
				db := setupInvoiceTestDB()
				return NewInvoiceRepository(db)
			},
			input: fixture.InvoiceMock(),
			inputTx: func(repository *InvoiceRepository) *gorm.DB {
				tx := repository.db.Begin()
				return tx
			},
			expectedInvoice: fixture.InvoiceMock(
				fixture.WithInvoiceUserID("user-test-id"),
			),
			expectedErr: nil,
		},
		"should fail when adding invoice with database error": {
			prepareDB: func() *InvoiceRepository {
				db := setupInvoiceTestDB()
				_ = db.Callback().Create().Before("gorm:create").Register("force_error", func(db *gorm.DB) {
					_ = db.AddError(assert.AnError)
				})
				return NewInvoiceRepository(db)
			},
			input: fixture.InvoiceMock(),
			inputTx: func(repository *InvoiceRepository) *gorm.DB {
				tx := repository.db.Begin()
				return tx
			},
			expectedInvoice: domain.Invoice{},
			expectedErr:     fmt.Errorf("error creating invoice: %w: %s", ErrDatabaseError, assert.AnError.Error()),
		},
		"should add invoice with external transaction": {
			prepareDB: func() *InvoiceRepository {
				db := setupInvoiceTestDB()
				return NewInvoiceRepository(db)
			},
			input: fixture.InvoiceMock(
				fixture.WithInvoiceAmount(2500.0),
			),
			inputTx: func(repository *InvoiceRepository) *gorm.DB {
				return nil
			},
			expectedInvoice: fixture.InvoiceMock(
				fixture.WithInvoiceAmount(2500.0),
			),
			expectedErr: nil,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			repo := tc.prepareDB()
			tx := tc.inputTx(repo)
			ctx := createInvoiceTestContext()
			defer func() {
				if tx != nil {
					tx.Rollback()
				}
			}()

			result, err := repo.Add(ctx, tx, tc.input)

			assert.Equal(t, tc.expectedErr, err)
			if tc.expectedErr == nil {
				assert.NotNil(t, result.ID)
				assert.NotZero(t, result.DateCreate)
				assert.NotZero(t, result.DateUpdate)
				assert.Equal(t, tc.expectedInvoice.UserID, result.UserID)
				assert.Equal(t, tc.expectedInvoice.Amount, result.Amount)
				assert.Equal(t, tc.expectedInvoice.IsPaid, result.IsPaid)
			}
		})
	}
}

func TestInvoiceRepository_FindByID(t *testing.T) {
	tests := map[string]struct {
		prepareDB       func() (*InvoiceRepository, uuid.UUID)
		expectedErr     error
		expectedInvoice domain.Invoice
	}{
		"should find invoice by id successfully": {
			prepareDB: func() (*InvoiceRepository, uuid.UUID) {
				db := setupInvoiceTestDB()
				repo := NewInvoiceRepository(db)

				invoice := fixture.InvoiceMock()
				dbInvoice := FromInvoiceDomain(invoice)
				_ = db.Create(&dbInvoice)

				return repo, *invoice.ID
			},
			expectedInvoice: fixture.InvoiceMock(),
			expectedErr:     nil,
		},
		"should fail when invoice not found": {
			prepareDB: func() (*InvoiceRepository, uuid.UUID) {
				db := setupInvoiceTestDB()
				repo := NewInvoiceRepository(db)
				return repo, uuid.New()
			},
			expectedInvoice: domain.Invoice{},
			expectedErr:     fmt.Errorf("error finding invoice: %w: %s", ErrInvoiceNotFound, "record not found"),
		},
		"should fail when database query fails": {
			prepareDB: func() (*InvoiceRepository, uuid.UUID) {
				db := setupInvoiceTestDB()
				_ = db.Callback().Query().Before("gorm:query").Register("force_error", func(db *gorm.DB) {
					_ = db.AddError(assert.AnError)
				})
				repo := NewInvoiceRepository(db)
				return repo, uuid.New()
			},
			expectedInvoice: domain.Invoice{},
			expectedErr:     fmt.Errorf("error finding invoice: %w: %s", ErrDatabaseError, assert.AnError.Error()),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			repo, id := tc.prepareDB()
			ctx := createInvoiceTestContext()

			result, err := repo.FindByID(ctx, id)

			assert.Equal(t, tc.expectedErr, err)
			if tc.expectedErr == nil {
				assert.Equal(t, tc.expectedInvoice.Amount, result.Amount)
				assert.Equal(t, tc.expectedInvoice.IsPaid, result.IsPaid)
				assert.Equal(t, tc.expectedInvoice.PeriodStart, result.PeriodStart)
				assert.Equal(t, tc.expectedInvoice.PeriodEnd, result.PeriodEnd)
			}
		})
	}
}

func TestInvoiceRepository_FindByPeriod(t *testing.T) {
	tests := map[string]struct {
		prepareDB        func() *InvoiceRepository
		period           domain.Period
		expectedInvoices int
		expectedErr      error
	}{
		"should find invoices by period successfully": {
			prepareDB: func() *InvoiceRepository {
				db := setupInvoiceTestDB()
				repo := NewInvoiceRepository(db)

				// Fatura para outubro 2023
				invoice1 := fixture.InvoiceMock(
					fixture.WithInvoicePeriod(
						time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC),
						time.Date(2023, 10, 31, 0, 0, 0, 0, time.UTC),
					),
				)
				dbInvoice1 := FromInvoiceDomain(invoice1)
				_ = db.Create(&dbInvoice1)

				// Fatura para novembro 2023
				invoice2 := fixture.InvoiceMock(
					fixture.WithInvoicePeriod(
						time.Date(2023, 11, 1, 0, 0, 0, 0, time.UTC),
						time.Date(2023, 11, 30, 0, 0, 0, 0, time.UTC),
					),
				)
				invoice2.ID = &[]uuid.UUID{uuid.New()}[0]
				dbInvoice2 := FromInvoiceDomain(invoice2)
				_ = db.Create(&dbInvoice2)

				// Fatura para dezembro 2023 (não deve aparecer na busca)
				invoice3 := fixture.InvoiceMock(
					fixture.WithInvoicePeriod(
						time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC),
						time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC),
					),
				)
				invoice3.ID = &[]uuid.UUID{uuid.New()}[0]
				dbInvoice3 := FromInvoiceDomain(invoice3)
				_ = db.Create(&dbInvoice3)

				return repo
			},
			period: domain.Period{
				From: time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2023, 11, 30, 0, 0, 0, 0, time.UTC),
			},
			expectedInvoices: 2,
			expectedErr:      nil,
		},
		"should return empty list when no invoices found": {
			prepareDB: func() *InvoiceRepository {
				db := setupInvoiceTestDB()
				return NewInvoiceRepository(db)
			},
			period: domain.Period{
				From: time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2023, 10, 31, 0, 0, 0, 0, time.UTC),
			},
			expectedInvoices: 0,
			expectedErr:      nil,
		},
		"should fail when database query fails": {
			prepareDB: func() *InvoiceRepository {
				db := setupInvoiceTestDB()
				_ = db.Callback().Query().Before("gorm:query").Register("force_error", func(db *gorm.DB) {
					_ = db.AddError(assert.AnError)
				})
				return NewInvoiceRepository(db)
			},
			period: domain.Period{
				From: time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2023, 10, 31, 0, 0, 0, 0, time.UTC),
			},
			expectedInvoices: 0,
			expectedErr:      fmt.Errorf("error finding invoices by period: %w: %s", ErrDatabaseError, assert.AnError.Error()),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			repo := tc.prepareDB()
			ctx := createInvoiceTestContext()

			results, err := repo.FindByPeriod(ctx, tc.period)

			assert.Equal(t, tc.expectedErr, err)
			if tc.expectedErr == nil {
				assert.Len(t, results, tc.expectedInvoices)
			}
		})
	}
}

func TestInvoiceRepository_FindByPeriodAndCreditCard(t *testing.T) {
	tests := map[string]struct {
		prepareDB        func() (*InvoiceRepository, uuid.UUID)
		period           domain.Period
		expectedInvoices int
		expectedErr      error
	}{
		"should find invoices by period and credit card successfully": {
			prepareDB: func() (*InvoiceRepository, uuid.UUID) {
				db := setupInvoiceTestDB()
				repo := NewInvoiceRepository(db)

				creditCardID1 := uuid.New()
				creditCardID2 := uuid.New()

				// Fatura para cartão 1
				invoice1 := fixture.InvoiceMock(
					fixture.WithInvoiceCreditCardID(creditCardID1),
				)
				dbInvoice1 := FromInvoiceDomain(invoice1)
				_ = db.Create(&dbInvoice1)

				// Fatura para cartão 2 (não deve aparecer na busca)
				invoice2 := fixture.InvoiceMock(
					fixture.WithInvoiceCreditCardID(creditCardID2),
				)
				invoice2.ID = &[]uuid.UUID{uuid.New()}[0]
				dbInvoice2 := FromInvoiceDomain(invoice2)
				_ = db.Create(&dbInvoice2)

				return repo, creditCardID1
			},
			period: domain.Period{
				From: time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2023, 10, 31, 0, 0, 0, 0, time.UTC),
			},
			expectedInvoices: 1,
			expectedErr:      nil,
		},
		"should fail when database query fails": {
			prepareDB: func() (*InvoiceRepository, uuid.UUID) {
				db := setupInvoiceTestDB()
				_ = db.Callback().Query().Before("gorm:query").Register("force_error", func(db *gorm.DB) {
					_ = db.AddError(assert.AnError)
				})
				repo := NewInvoiceRepository(db)
				return repo, uuid.New()
			},
			period: domain.Period{
				From: time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2023, 10, 31, 0, 0, 0, 0, time.UTC),
			},
			expectedInvoices: 0,
			expectedErr:      fmt.Errorf("error finding invoices by period and credit card: %w: %s", ErrDatabaseError, assert.AnError.Error()),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			repo, creditCardID := tc.prepareDB()
			ctx := createInvoiceTestContext()

			results, err := repo.FindByPeriodAndCreditCard(ctx, tc.period, creditCardID)

			assert.Equal(t, tc.expectedErr, err)
			if tc.expectedErr == nil {
				assert.Len(t, results, tc.expectedInvoices)
				if len(results) > 0 {
					assert.Equal(t, creditCardID, *results[0].CreditCardID)
				}
			}
		})
	}
}

func TestInvoiceRepository_UpdateAmount(t *testing.T) {
	tests := map[string]struct {
		prepareDB       func() (*InvoiceRepository, uuid.UUID)
		newAmount       float64
		expectedErr     error
		expectedInvoice domain.Invoice
	}{
		"should update amount successfully": {
			prepareDB: func() (*InvoiceRepository, uuid.UUID) {
				db := setupInvoiceTestDB()
				repo := NewInvoiceRepository(db)

				invoice := fixture.InvoiceMock()
				dbInvoice := FromInvoiceDomain(invoice)
				_ = db.Create(&dbInvoice)

				return repo, *invoice.ID
			},
			newAmount: 2000.0,
			expectedInvoice: fixture.InvoiceMock(
				fixture.WithInvoiceAmount(2000.0),
			),
			expectedErr: nil,
		},
		"should fail when invoice not found": {
			prepareDB: func() (*InvoiceRepository, uuid.UUID) {
				db := setupInvoiceTestDB()
				repo := NewInvoiceRepository(db)
				return repo, uuid.New()
			},
			newAmount:       2000.0,
			expectedInvoice: domain.Invoice{},
			expectedErr:     fmt.Errorf("error finding invoice: %w: %s", ErrInvoiceNotFound, "record not found"),
		},
		"should fail when database query fails": {
			prepareDB: func() (*InvoiceRepository, uuid.UUID) {
				db := setupInvoiceTestDB()
				_ = db.Callback().Query().Before("gorm:query").Register("force_error", func(db *gorm.DB) {
					_ = db.AddError(assert.AnError)
				})
				repo := NewInvoiceRepository(db)
				return repo, uuid.New()
			},
			newAmount:       2000.0,
			expectedInvoice: domain.Invoice{},
			expectedErr:     fmt.Errorf("error finding invoice: %w: %s", ErrDatabaseError, assert.AnError.Error()),
		},
		"should fail when database update fails": {
			prepareDB: func() (*InvoiceRepository, uuid.UUID) {
				db := setupInvoiceTestDB()
				repo := NewInvoiceRepository(db)

				invoice := fixture.InvoiceMock()
				dbInvoice := FromInvoiceDomain(invoice)
				_ = db.Create(&dbInvoice)

				_ = db.Callback().Update().Before("gorm:update").Register("force_error", func(db *gorm.DB) {
					_ = db.AddError(assert.AnError)
				})

				return repo, *invoice.ID
			},
			newAmount:       2000.0,
			expectedInvoice: domain.Invoice{},
			expectedErr:     fmt.Errorf("error updating invoice amount: %w: %s", ErrDatabaseError, assert.AnError.Error()),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			repo, id := tc.prepareDB()
			ctx := createInvoiceTestContext()

			result, err := repo.UpdateAmount(ctx, nil, id, tc.newAmount)

			assert.Equal(t, tc.expectedErr, err)
			if tc.expectedErr == nil {
				assert.Equal(t, tc.newAmount, result.Amount)
			}
		})
	}
}

func TestInvoiceRepository_UpdateStatus(t *testing.T) {
	tests := map[string]struct {
		prepareDB       func() (*InvoiceRepository, uuid.UUID)
		isPaid          bool
		paymentDate     *time.Time
		walletID        *uuid.UUID
		expectedErr     error
		expectedInvoice domain.Invoice
	}{
		"should update status to paid successfully": {
			prepareDB: func() (*InvoiceRepository, uuid.UUID) {
				db := setupInvoiceTestDB()
				repo := NewInvoiceRepository(db)

				invoice := fixture.InvoiceMock()
				dbInvoice := FromInvoiceDomain(invoice)
				_ = db.Create(&dbInvoice)

				return repo, *invoice.ID
			},
			isPaid:      true,
			paymentDate: &[]time.Time{time.Now()}[0],
			walletID:    &fixture.WalletID,
			expectedInvoice: fixture.InvoiceMock(
				fixture.WithInvoiceIsPaid(true),
			),
			expectedErr: nil,
		},
		"should update status to unpaid successfully": {
			prepareDB: func() (*InvoiceRepository, uuid.UUID) {
				db := setupInvoiceTestDB()
				repo := NewInvoiceRepository(db)

				invoice := fixture.InvoiceMock(
					fixture.WithInvoiceIsPaid(true),
				)
				dbInvoice := FromInvoiceDomain(invoice)
				_ = db.Create(&dbInvoice)

				return repo, *invoice.ID
			},
			isPaid:      false,
			paymentDate: nil,
			walletID:    nil,
			expectedInvoice: fixture.InvoiceMock(
				fixture.WithInvoiceIsPaid(false),
			),
			expectedErr: nil,
		},
		"should fail when invoice not found": {
			prepareDB: func() (*InvoiceRepository, uuid.UUID) {
				db := setupInvoiceTestDB()
				repo := NewInvoiceRepository(db)
				return repo, uuid.New()
			},
			isPaid:          true,
			paymentDate:     &[]time.Time{time.Now()}[0],
			walletID:        &fixture.WalletID,
			expectedInvoice: domain.Invoice{},
			expectedErr:     fmt.Errorf("error finding invoice: %w: %s", ErrInvoiceNotFound, "record not found"),
		},
		"should fail when database query fails": {
			prepareDB: func() (*InvoiceRepository, uuid.UUID) {
				db := setupInvoiceTestDB()
				_ = db.Callback().Query().Before("gorm:query").Register("force_error", func(db *gorm.DB) {
					_ = db.AddError(assert.AnError)
				})
				repo := NewInvoiceRepository(db)
				return repo, uuid.New()
			},
			isPaid:          true,
			paymentDate:     &[]time.Time{time.Now()}[0],
			walletID:        &fixture.WalletID,
			expectedInvoice: domain.Invoice{},
			expectedErr:     fmt.Errorf("error finding invoice: %w: %s", ErrDatabaseError, assert.AnError.Error()),
		},
		"should fail when database update fails": {
			prepareDB: func() (*InvoiceRepository, uuid.UUID) {
				db := setupInvoiceTestDB()
				repo := NewInvoiceRepository(db)

				invoice := fixture.InvoiceMock()
				dbInvoice := FromInvoiceDomain(invoice)
				_ = db.Create(&dbInvoice)

				_ = db.Callback().Update().Before("gorm:update").Register("force_error", func(db *gorm.DB) {
					_ = db.AddError(assert.AnError)
				})

				return repo, *invoice.ID
			},
			isPaid:          true,
			paymentDate:     &[]time.Time{time.Now()}[0],
			walletID:        &fixture.WalletID,
			expectedInvoice: domain.Invoice{},
			expectedErr:     fmt.Errorf("error updating invoice status: %w: %s", ErrDatabaseError, assert.AnError.Error()),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			repo, id := tc.prepareDB()
			ctx := createInvoiceTestContext()

			result, err := repo.UpdateStatus(ctx, nil, id, tc.isPaid, tc.paymentDate, tc.walletID)

			assert.Equal(t, tc.expectedErr, err)
			if tc.expectedErr == nil {
				assert.Equal(t, tc.isPaid, result.IsPaid)
				if tc.paymentDate != nil {
					assert.NotNil(t, result.PaymentDate)
				}
				if tc.walletID != nil {
					assert.Equal(t, *tc.walletID, *result.WalletID)
				}
			}
		})
	}
}
