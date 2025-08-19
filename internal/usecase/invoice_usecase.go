package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/infrastructure/repository"
	"personal-finance/internal/infrastructure/repository/transaction"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type InvoiceRepository interface {
	Add(ctx context.Context, tx *gorm.DB, invoice domain.Invoice) (domain.Invoice, error)
	FindByID(ctx context.Context, id uuid.UUID) (domain.Invoice, error)
	FindByMonth(ctx context.Context, date time.Time) ([]domain.Invoice, error)
	FindByMonthAndCreditCard(ctx context.Context, date time.Time, creditCardID uuid.UUID) (domain.Invoice, error)
	UpdateAmount(ctx context.Context, tx *gorm.DB, id uuid.UUID, amount float64) (domain.Invoice, error)
	UpdateStatus(ctx context.Context, tx *gorm.DB, id uuid.UUID, isPaid bool, paymentDate *time.Time, walletID *uuid.UUID) (domain.Invoice, error)
}

type Invoice struct {
	repo           InvoiceRepository
	creditCardRepo CreditCardRepository
	walletRepo     WalletRepository
	txManager      transaction.Manager
}

func NewInvoice(
	repo InvoiceRepository,
	creditCardRepo CreditCardRepository,
	walletRepo WalletRepository,
	txManager transaction.Manager,
) Invoice {
	return Invoice{
		repo:           repo,
		creditCardRepo: creditCardRepo,
		walletRepo:     walletRepo,
		txManager:      txManager,
	}
}

func (uc Invoice) FindOrCreateInvoiceForMovement(ctx context.Context, invoiceID *uuid.UUID, creditCardID uuid.UUID, movementDate time.Time) (domain.Invoice, error) {
	if invoiceID != nil {
		invoice, err := uc.repo.FindByID(ctx, *invoiceID)
		if err != nil && err != ErrInvoiceNotFound {
			return domain.Invoice{}, fmt.Errorf("error finding invoice: %w", err)
		}
		return invoice, nil
	}

	invoices, err := uc.repo.FindByMonthAndCreditCard(ctx, movementDate, creditCardID)
	if err != nil {
		if !errors.Is(err, repository.ErrInvoiceNotFound) {
			return domain.Invoice{}, err
		}
	}

	if invoices.ID != nil {
		return invoices, nil
	}

	return uc.create(ctx, creditCardID, movementDate)
}

func (uc Invoice) create(ctx context.Context, creditCardID uuid.UUID, movementDate time.Time) (domain.Invoice, error) {
	creditCard, err := uc.creditCardRepo.FindByID(ctx, creditCardID)
	if err != nil {
		return domain.Invoice{}, fmt.Errorf("error finding credit card: %w", err)
	}

	invoice := domain.BuildInvoice(creditCard, movementDate)

	var result domain.Invoice
	err = uc.txManager.WithTransaction(ctx, func(tx *gorm.DB) error {
		createdInvoice, err := uc.repo.Add(ctx, tx, invoice)
		if err != nil {
			return fmt.Errorf("error creating new invoice: %w", err)
		}
		result = createdInvoice
		return nil
	})

	if err != nil {
		return domain.Invoice{}, err
	}

	return result, nil
}

func (uc Invoice) FindByMonth(ctx context.Context, date time.Time) ([]domain.Invoice, error) {
	result, err := uc.repo.FindByMonth(ctx, date)
	if err != nil {
		return []domain.Invoice{}, fmt.Errorf("error finding invoices by period: %w", err)
	}
	return result, nil
}

func (uc Invoice) FindByID(ctx context.Context, id uuid.UUID) (domain.Invoice, error) {
	result, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return domain.Invoice{}, fmt.Errorf("error finding invoice: %w", err)
	}
	return result, nil
}

func (uc Invoice) UpdateAmount(ctx context.Context, id uuid.UUID, amount float64) (domain.Invoice, error) {
	invoice, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return domain.Invoice{}, fmt.Errorf("error finding invoice: %w", err)
	}

	if invoice.IsPaid {
		return domain.Invoice{}, ErrInvoiceCannotModify
	}

	var result domain.Invoice
	err = uc.txManager.WithTransaction(ctx, func(tx *gorm.DB) error {
		updatedInvoice, err := uc.repo.UpdateAmount(ctx, tx, id, amount)
		if err != nil {
			return fmt.Errorf("error updating invoice amount: %w", err)
		}
		result = updatedInvoice
		return nil
	})

	if err != nil {
		return domain.Invoice{}, err
	}

	return result, nil
}

func (uc Invoice) Pay(ctx context.Context, id uuid.UUID, walletID uuid.UUID, paymentDate *time.Time) (domain.Invoice, error) {
	invoice, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return domain.Invoice{}, fmt.Errorf("error finding invoice: %w", err)
	}

	if invoice.IsPaid {
		return domain.Invoice{}, ErrInvoiceAlreadyPaid
	}

	wallet, err := uc.walletRepo.FindByID(ctx, &walletID)
	if err != nil {
		return domain.Invoice{}, fmt.Errorf("error finding wallet: %w", err)
	}

	if !wallet.HasSufficientBalance(invoice.Amount) {
		return domain.Invoice{}, ErrInsufficientBalance
	}

	if paymentDate == nil {
		now := time.Now()
		paymentDate = &now
	}

	var result domain.Invoice
	err = uc.txManager.WithTransaction(ctx, func(tx *gorm.DB) error {
		if err := uc.walletRepo.UpdateAmount(ctx, tx, &walletID, wallet.Balance+invoice.Amount); err != nil {
			return fmt.Errorf("error updating wallet balance: %w", err)
		}

		updatedInvoice, err := uc.repo.UpdateStatus(ctx, tx, id, true, paymentDate, &walletID)
		if err != nil {
			return fmt.Errorf("error marking invoice as paid: %w", err)
		}
		result = updatedInvoice
		return nil
	})

	if err != nil {
		return domain.Invoice{}, err
	}

	return result, nil
}
