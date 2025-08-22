package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/plataform/authentication"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrInvoiceNotFound = errors.New("invoice not found")
)

type InvoiceRepository struct {
	db *gorm.DB
}

func NewInvoiceRepository(db *gorm.DB) *InvoiceRepository {
	return &InvoiceRepository{
		db: db,
	}
}

func (r *InvoiceRepository) Add(ctx context.Context, tx *gorm.DB, invoice domain.Invoice) (domain.Invoice, error) {
	var isLocalTx bool
	if tx == nil {
		isLocalTx = true
		tx = r.db.WithContext(ctx).Begin()
		defer tx.Rollback()
	}

	userID := ctx.Value(authentication.UserID).(string)
	now := time.Now()
	id := uuid.New()

	invoice.ID = &id
	invoice.DateCreate = now
	invoice.DateUpdate = now
	invoice.UserID = userID

	dbInvoice := FromInvoiceDomain(invoice)

	if err := tx.WithContext(ctx).Create(&dbInvoice).Error; err != nil {
		return domain.Invoice{}, fmt.Errorf("error creating invoice: %w: %s", ErrDatabaseError, err.Error())
	}

	if isLocalTx {
		if err := tx.Commit().Error; err != nil {
			return domain.Invoice{}, fmt.Errorf("error committing transaction: %w: %s", ErrDatabaseError, err.Error())
		}
	}

	return dbInvoice.ToDomain(), nil
}

func (r *InvoiceRepository) FindByID(ctx context.Context, id uuid.UUID) (domain.Invoice, error) {
	var dbModel InvoiceDB
	tableName := dbModel.TableName()

	query := BuildBaseQuery(ctx, r.db, tableName)
	query = r.appendPreloads(query)

	if err := query.First(&dbModel, fmt.Sprintf("%s.id = ?", tableName), id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Invoice{}, fmt.Errorf("error finding invoice: %w: %s", ErrInvoiceNotFound, err.Error())
		}
		return domain.Invoice{}, fmt.Errorf("error finding invoice: %w: %s", ErrDatabaseError, err.Error())
	}

	return dbModel.ToDomain(), nil
}

func (r *InvoiceRepository) FindOpenByMonth(ctx context.Context, date time.Time) ([]domain.Invoice, error) {
	var dbModel InvoiceDB
	tableName := dbModel.TableName()

	query := BuildBaseQuery(ctx, r.db, tableName)
	query = r.appendPreloads(query)

	firstDay := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, date.Location())
	lastDay := firstDay.AddDate(0, 1, -1).Add(23*time.Hour + 59*time.Minute + 59*time.Second)

	var dbInvoices []InvoiceDB
	err := query.Where(
		fmt.Sprintf("%s.due_date >= ? AND %s.due_date <= ? AND %s.is_paid = false", tableName, tableName, tableName),
		firstDay, lastDay,
	).Find(&dbInvoices).Error

	if err != nil {
		return nil, fmt.Errorf("error finding invoices by month: %w: %s", ErrDatabaseError, err.Error())
	}

	invoices := make([]domain.Invoice, len(dbInvoices))
	for i, dbInvoice := range dbInvoices {
		invoices[i] = dbInvoice.ToDomain()
	}

	return invoices, nil
}

func (r *InvoiceRepository) FindByMonth(ctx context.Context, date time.Time) ([]domain.Invoice, error) {
	var dbModel InvoiceDB
	tableName := dbModel.TableName()

	query := BuildBaseQuery(ctx, r.db, tableName)
	query = r.appendPreloads(query)

	firstDay := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, date.Location())
	lastDay := firstDay.AddDate(0, 1, -1).Add(23*time.Hour + 59*time.Minute + 59*time.Second)

	var dbInvoices []InvoiceDB
	err := query.Where(
		fmt.Sprintf("%s.due_date >= ? AND %s.due_date <= ?", tableName, tableName),
		firstDay, lastDay,
	).Find(&dbInvoices).Error

	if err != nil {
		return nil, fmt.Errorf("error finding invoices by month: %w: %s", ErrDatabaseError, err.Error())
	}

	invoices := make([]domain.Invoice, len(dbInvoices))
	for i, dbInvoice := range dbInvoices {
		invoices[i] = dbInvoice.ToDomain()
	}

	return invoices, nil
}

func (r *InvoiceRepository) FindByMonthAndCreditCard(ctx context.Context, date time.Time, creditCardID uuid.UUID) (domain.Invoice, error) {
	var dbModel InvoiceDB
	tableName := dbModel.TableName()

	query := BuildBaseQuery(ctx, r.db, tableName)
	query = r.appendPreloads(query)

	firstDay := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, date.Location())
	lastDay := firstDay.AddDate(0, 1, -1).Add(23*time.Hour + 59*time.Minute + 59*time.Second)

	var dbInvoices InvoiceDB
	err := query.Where(
		fmt.Sprintf("%s.credit_card_id = ? AND %s.due_date >= ? AND %s.due_date <= ?",
			tableName, tableName, tableName),
		creditCardID, firstDay, lastDay,
	).First(&dbInvoices).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Invoice{}, fmt.Errorf("error finding invoices by month and credit card: %w: %s", ErrInvoiceNotFound, err.Error())
		}
		return domain.Invoice{}, fmt.Errorf("error finding invoices by month and credit card: %w: %s", ErrDatabaseError, err.Error())
	}

	return dbInvoices.ToDomain(), nil
}

func (r *InvoiceRepository) FindOpenByCreditCard(ctx context.Context, creditCardID uuid.UUID) ([]domain.Invoice, error) {
	var dbModel InvoiceDB
	tableName := dbModel.TableName()

	query := BuildBaseQuery(ctx, r.db, tableName)
	query = r.appendPreloads(query)

	query.Where(
		fmt.Sprintf("%s.credit_card_id = ? AND %s.is_paid = false", tableName, tableName),
		creditCardID,
	)

	var dbInvoices []InvoiceDB
	err := query.Find(&dbInvoices).Error
	if err != nil {
		return nil, fmt.Errorf("error finding open invoices by credit card: %w: %s", ErrDatabaseError, err.Error())
	}

	invoices := make([]domain.Invoice, len(dbInvoices))
	for i, dbInvoice := range dbInvoices {
		invoices[i] = dbInvoice.ToDomain()
	}

	return invoices, nil
}

func (r *InvoiceRepository) UpdateAmount(ctx context.Context, tx *gorm.DB, id uuid.UUID, amount float64) (domain.Invoice, error) {
	var isLocalTx bool
	if tx == nil {
		isLocalTx = true
		tx = r.db.WithContext(ctx).Begin()
		defer tx.Rollback()
	}

	var invoiceDB InvoiceDB
	tableName := invoiceDB.TableName()

	query := BuildBaseQuery(ctx, tx, tableName)
	query = r.appendPreloads(query)
	if err := query.First(&invoiceDB, fmt.Sprintf("%s.id = ?", tableName), id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Invoice{}, fmt.Errorf("error finding invoice: %w: %s", ErrInvoiceNotFound, err.Error())
		}
		return domain.Invoice{}, fmt.Errorf("error finding invoice: %w: %s", ErrDatabaseError, err.Error())
	}

	now := time.Now()
	invoiceDB.Amount = amount
	invoiceDB.DateUpdate = now

	if err := tx.WithContext(ctx).Save(&invoiceDB).Error; err != nil {
		return domain.Invoice{}, fmt.Errorf("error updating invoice amount: %w: %s", ErrDatabaseError, err.Error())
	}

	if isLocalTx {
		if err := tx.Commit().Error; err != nil {
			return domain.Invoice{}, fmt.Errorf("error committing transaction: %w: %s", ErrDatabaseError, err.Error())
		}
	}

	return invoiceDB.ToDomain(), nil
}

func (r *InvoiceRepository) UpdateStatus(
	ctx context.Context,
	tx *gorm.DB,
	id uuid.UUID,
	isPaid bool,
	paymentDate *time.Time,
	walletID *uuid.UUID,
) (domain.Invoice, error) {
	var isLocalTx bool
	if tx == nil {
		isLocalTx = true
		tx = r.db.WithContext(ctx).Begin()
		defer tx.Rollback()
	}

	var dbModel InvoiceDB
	tableName := dbModel.TableName()

	query := BuildBaseQuery(ctx, tx, tableName)
	query = r.appendPreloads(query)
	if err := query.First(&dbModel, fmt.Sprintf("%s.id = ?", tableName), id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Invoice{}, fmt.Errorf("error finding invoice: %w: %s", ErrInvoiceNotFound, err.Error())
		}
		return domain.Invoice{}, fmt.Errorf("error finding invoice: %w: %s", ErrDatabaseError, err.Error())
	}

	now := time.Now()
	dbModel.IsPaid = isPaid
	dbModel.PaymentDate = paymentDate
	dbModel.WalletID = walletID
	dbModel.DateUpdate = now

	if err := tx.WithContext(ctx).Save(&dbModel).Error; err != nil {
		return domain.Invoice{}, fmt.Errorf("error updating invoice status: %w: %s", ErrDatabaseError, err.Error())
	}

	if isLocalTx {
		if err := tx.Commit().Error; err != nil {
			return domain.Invoice{}, fmt.Errorf("error committing transaction: %w: %s", ErrDatabaseError, err.Error())
		}
	}

	return dbModel.ToDomain(), nil
}

func (r *InvoiceRepository) appendPreloads(query *gorm.DB) *gorm.DB {
	return query.Preload("CreditCard").Preload("Wallet")
}
