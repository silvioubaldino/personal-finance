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

func (r *InvoiceRepository) FindByPeriod(ctx context.Context, period domain.Period) ([]domain.Invoice, error) {
	var dbModel InvoiceDB
	tableName := dbModel.TableName()

	query := BuildBaseQuery(ctx, r.db, tableName)
	query = r.appendPreloads(query)

	var dbInvoices []InvoiceDB
	err := query.Where(
		fmt.Sprintf("(%s.period_start <= ? AND %s.period_end >= ?) OR (%s.period_start <= ? AND %s.period_start >= ?)",
			tableName, tableName, tableName, tableName),
		period.To, period.From, period.From, period.From,
	).Find(&dbInvoices).Error

	if err != nil {
		return nil, fmt.Errorf("error finding invoices by period: %w: %s", ErrDatabaseError, err.Error())
	}

	invoices := make([]domain.Invoice, len(dbInvoices))
	for i, dbInvoice := range dbInvoices {
		invoices[i] = dbInvoice.ToDomain()
	}

	return invoices, nil
}

func (r *InvoiceRepository) FindByPeriodAndCreditCard(ctx context.Context, period domain.Period, creditCardID uuid.UUID) ([]domain.Invoice, error) {
	var dbModel InvoiceDB
	tableName := dbModel.TableName()

	query := BuildBaseQuery(ctx, r.db, tableName)
	query = r.appendPreloads(query)

	var dbInvoices []InvoiceDB
	err := query.Where(
		fmt.Sprintf("%s.credit_card_id = ? AND ((%s.period_start <= ? AND %s.period_end >= ?) OR (%s.period_start <= ? AND %s.period_start >= ?))",
			tableName, tableName, tableName, tableName, tableName),
		creditCardID,
		period.To, period.From, period.From, period.From,
	).Find(&dbInvoices).Error

	if err != nil {
		return nil, fmt.Errorf("error finding invoices by period and credit card: %w: %s", ErrDatabaseError, err.Error())
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
	dbModel.Amount = amount
	dbModel.DateUpdate = now

	if err := tx.WithContext(ctx).Save(&dbModel).Error; err != nil {
		return domain.Invoice{}, fmt.Errorf("error updating invoice amount: %w: %s", ErrDatabaseError, err.Error())
	}

	if isLocalTx {
		if err := tx.Commit().Error; err != nil {
			return domain.Invoice{}, fmt.Errorf("error committing transaction: %w: %s", ErrDatabaseError, err.Error())
		}
	}

	return dbModel.ToDomain(), nil
}

func (r *InvoiceRepository) UpdateStatus(ctx context.Context, tx *gorm.DB, id uuid.UUID, isPaid bool, paymentDate *time.Time, walletID *uuid.UUID) (domain.Invoice, error) {
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
