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
	FindOpenByMonth(ctx context.Context, date time.Time) ([]domain.Invoice, error)
	FindByMonth(ctx context.Context, date time.Time) ([]domain.Invoice, error)
	FindByMonthAndCreditCard(ctx context.Context, date time.Time, creditCardID uuid.UUID) (domain.Invoice, error)
	FindOpenByCreditCard(ctx context.Context, creditCardID uuid.UUID) ([]domain.Invoice, error)
	UpdateAmount(ctx context.Context, tx *gorm.DB, id uuid.UUID, amount float64) (domain.Invoice, error)
	UpdateStatus(ctx context.Context, tx *gorm.DB, id uuid.UUID, isPaid bool, paymentDate *time.Time, walletID *uuid.UUID) (domain.Invoice, error)
}

type movRepo interface {
	Add(ctx context.Context, tx *gorm.DB, movement domain.Movement) (domain.Movement, error)
	FindByID(ctx context.Context, id uuid.UUID) (domain.Movement, error)
	FindByInvoiceID(ctx context.Context, invoiceID uuid.UUID) (domain.MovementList, error)
	Delete(ctx context.Context, tx *gorm.DB, id uuid.UUID) error
	DeleteByInvoiceID(ctx context.Context, tx *gorm.DB, invoiceID uuid.UUID) error
}

type Invoice struct {
	repo           InvoiceRepository
	creditCardRepo CreditCardRepository
	walletRepo     WalletRepository
	movementRepo   movRepo
	txManager      transaction.Manager
}

func NewInvoice(
	repo InvoiceRepository,
	creditCardRepo CreditCardRepository,
	walletRepo WalletRepository,
	movementRepo movRepo,
	txManager transaction.Manager,
) Invoice {
	return Invoice{
		repo:           repo,
		creditCardRepo: creditCardRepo,
		walletRepo:     walletRepo,
		movementRepo:   movementRepo,
		txManager:      txManager,
	}
}

func (uc Invoice) FindOrCreateInvoiceForMovement(ctx context.Context, invoiceID *uuid.UUID, creditCardID *uuid.UUID, movementDate time.Time) (domain.Invoice, error) {
	if invoiceID != nil {
		invoice, err := uc.repo.FindByID(ctx, *invoiceID)
		if err != nil {
			if !errors.Is(err, repository.ErrInvoiceNotFound) {
				return domain.Invoice{}, fmt.Errorf("error finding invoice: %w", err)
			}
		}
		if invoice.ID != nil {
			return invoice, nil
		}
	}

	invoices, err := uc.repo.FindByMonthAndCreditCard(ctx, movementDate, *creditCardID)
	if err != nil {
		if !errors.Is(err, repository.ErrInvoiceNotFound) {
			return domain.Invoice{}, err
		}
	}

	if invoices.ID != nil {
		return invoices, nil
	}

	return uc.create(ctx, *creditCardID, movementDate)
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

func (uc Invoice) FindDetailedInvoicesByPeriod(ctx context.Context, period domain.Period) ([]domain.DetailedInvoice, error) {
	invoices, err := uc.repo.FindByMonth(ctx, period.From)
	if err != nil {
		return []domain.DetailedInvoice{}, err
	}

	detailedInvoices := make([]domain.DetailedInvoice, len(invoices))

	for i, invoice := range invoices {
		movements, err := uc.movementRepo.FindByInvoiceID(ctx, *invoice.ID)
		if err != nil {
			return []domain.DetailedInvoice{}, err
		}
		detailedInvoices[i] = domain.DetailedInvoice{
			Invoice:   invoice,
			Movements: movements,
		}
	}

	return detailedInvoices, nil
}

func (uc Invoice) FindByMonth(ctx context.Context, date time.Time) ([]domain.Invoice, error) {
	result, err := uc.repo.FindOpenByMonth(ctx, date)
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
		amount = invoice.Amount + amount
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

func (uc Invoice) Pay(ctx context.Context, id uuid.UUID, walletID uuid.UUID, paymentDate *time.Time, amount *float64) (domain.Invoice, error) {
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

	if paymentDate == nil {
		now := time.Now()
		paymentDate = &now
	}

	paidAmount := invoice.Amount
	if amount != nil {
		if *amount > 0 {
			normalized := -*amount
			paidAmount = normalized
		} else {
			paidAmount = *amount
		}

		if paidAmount < invoice.Amount || paidAmount >= 0 {
			return domain.Invoice{}, ErrInvalidPaymentAmount
		}
	}

	err = wallet.Pay(paidAmount)
	if err != nil {
		return domain.Invoice{}, fmt.Errorf("error paying invoice: %w", err)
	}

	var result domain.Invoice
	err = uc.txManager.WithTransaction(ctx, func(tx *gorm.DB) error {
		if err := uc.walletRepo.UpdateAmount(ctx, tx, &walletID, wallet.Balance); err != nil {
			return fmt.Errorf("error updating wallet balance: %w", err)
		}

		updatedInvoice, err := uc.repo.UpdateStatus(ctx, tx, id, true, paymentDate, &walletID)
		if err != nil {
			return fmt.Errorf("error marking invoice as paid: %w", err)
		}
		result = updatedInvoice

		_, err = uc.movementRepo.Add(ctx, tx, buildMovementWithAmount(result, paidAmount))
		if err != nil {
			return fmt.Errorf("error creating movement: %w", err)
		}

		_, err = uc.creditCardRepo.UpdateLimitDelta(ctx, tx, *invoice.CreditCardID, -paidAmount)
		if err != nil {
			return fmt.Errorf("error updating credit card limit: %w", err)
		}

		if err := uc.handleRemainder(ctx, tx, invoice, paidAmount); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return domain.Invoice{}, err
	}

	return result, nil
}

func (uc Invoice) handleRemainder(ctx context.Context, tx *gorm.DB, invoice domain.Invoice, paidAmount float64) error {
	remainder := invoice.Amount - paidAmount
	if remainder == 0 {
		return nil
	}

	nextDate := invoice.DueDate.AddDate(0, 0, 1)
	nextInvoice, err := uc.FindOrCreateInvoiceForMovement(ctx, nil, invoice.CreditCardID, nextDate)
	if err != nil {
		return fmt.Errorf("error finding/creating next invoice: %w", err)
	}

	newAmount := nextInvoice.Amount + remainder
	_, err = uc.repo.UpdateAmount(ctx, tx, *nextInvoice.ID, newAmount)
	if err != nil {
		return fmt.Errorf("error updating next invoice amount: %w", err)
	}

	remainderMovement := buildRemainderMovement(invoice, nextInvoice, remainder, nextDate)
	_, err = uc.movementRepo.Add(ctx, tx, remainderMovement)
	if err != nil {
		return fmt.Errorf("error creating remainder movement: %w", err)
	}

	return nil
}

func (uc Invoice) RevertPayment(ctx context.Context, id uuid.UUID) (domain.Invoice, error) {
	invoice, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return domain.Invoice{}, fmt.Errorf("error finding invoice: %w", err)
	}

	if !invoice.IsPaid {
		return domain.Invoice{}, ErrInvoiceNotPaid
	}

	paymentMovement, err := uc.movementRepo.FindByID(ctx, *invoice.ID)
	if err != nil {
		return domain.Invoice{}, fmt.Errorf("error finding payment movement: %w", err)
	}
	paidAmount := paymentMovement.Amount

	wallet, err := uc.walletRepo.FindByID(ctx, invoice.WalletID)
	if err != nil {
		return domain.Invoice{}, fmt.Errorf("error finding wallet: %w", err)
	}

	var result domain.Invoice
	err = uc.txManager.WithTransaction(ctx, func(tx *gorm.DB) error {
		if err := wallet.RevertPayment(paidAmount); err != nil {
			return fmt.Errorf("error reverting payment: %w", err)
		}

		if err := uc.walletRepo.UpdateAmount(ctx, tx, invoice.WalletID, wallet.Balance); err != nil {
			return fmt.Errorf("error updating wallet balance: %w", err)
		}

		updatedInvoice, err := uc.repo.UpdateStatus(ctx, tx, id, false, nil, nil)
		if err != nil {
			return fmt.Errorf("error reverting invoice payment status: %w", err)
		}
		result = updatedInvoice

		err = uc.movementRepo.DeleteByInvoiceID(ctx, tx, id)
		if err != nil {
			return fmt.Errorf("error deleting invoice movement: %w", err)
		}

		_, err = uc.creditCardRepo.UpdateLimitDelta(ctx, tx, *invoice.CreditCardID, paidAmount)
		if err != nil {
			return fmt.Errorf("error updating credit card limit: %w", err)
		}

		remainder := invoice.Amount - paidAmount
		if remainder != 0 {
			nextDate := invoice.DueDate.AddDate(0, 0, 1)
			nextInvoice, err := uc.FindOrCreateInvoiceForMovement(ctx, nil, invoice.CreditCardID, nextDate)
			if err != nil {
				return fmt.Errorf("error finding next invoice: %w", err)
			}

			movements, err := uc.movementRepo.FindByInvoiceID(ctx, *nextInvoice.ID)
			if err != nil {
				return fmt.Errorf("error finding movements in next invoice: %w", err)
			}

			for _, mov := range movements {
				if mov.TypePayment == domain.TypePaymentInvoiceRemainder &&
					mov.Date != nil && mov.Date.Equal(nextDate) &&
					mov.Amount == remainder {
					newAmount := nextInvoice.Amount - remainder
					_, err = uc.repo.UpdateAmount(ctx, tx, *nextInvoice.ID, newAmount)
					if err != nil {
						return fmt.Errorf("error updating next invoice amount: %w", err)
					}

					err = uc.movementRepo.Delete(ctx, tx, *mov.ID)
					if err != nil {
						return fmt.Errorf("error deleting remainder movement: %w", err)
					}
					break
				}
			}
		}

		return nil
	})

	if err != nil {
		return domain.Invoice{}, err
	}

	return result, nil
}

func buildMovement(invoice domain.Invoice) domain.Movement {
	defaultCreditCardCategoryID := uuid.MustParse("d47cc960-f08d-480e-bf01-f4ec5ddfcb8b")

	return domain.Movement{
		ID:          invoice.ID,
		Description: buildCreditCardDescription(invoice.CreditCard.Name),
		Amount:      invoice.Amount,
		Date:        &invoice.DueDate,
		UserID:      invoice.UserID,
		IsPaid:      invoice.IsPaid,
		CreditCardInfo: &domain.CreditCardMovement{
			InvoiceID:    invoice.ID,
			CreditCardID: invoice.CreditCardID,
		},
		WalletID:    invoice.WalletID,
		TypePayment: domain.TypePaymentInvoicePayment,
		CategoryID:  &defaultCreditCardCategoryID,
	}
}

func buildMovementWithAmount(invoice domain.Invoice, amount float64) domain.Movement {
	defaultCreditCardCategoryID := uuid.MustParse("d47cc960-f08d-480e-bf01-f4ec5ddfcb8b")

	return domain.Movement{
		ID:          invoice.ID,
		Description: buildCreditCardDescription(invoice.CreditCard.Name),
		Amount:      amount,
		Date:        &invoice.DueDate,
		UserID:      invoice.UserID,
		IsPaid:      invoice.IsPaid,
		CreditCardInfo: &domain.CreditCardMovement{
			InvoiceID:    invoice.ID,
			CreditCardID: invoice.CreditCardID,
		},
		WalletID:    invoice.WalletID,
		TypePayment: domain.TypePaymentInvoicePayment,
		CategoryID:  &defaultCreditCardCategoryID,
	}
}

func buildRemainderMovement(originalInvoice domain.Invoice, nextInvoice domain.Invoice, remainder float64, date time.Time) domain.Movement {
	defaultCreditCardCategoryID := uuid.MustParse("d47cc960-f08d-480e-bf01-f4ec5ddfcb8b")

	return domain.Movement{
		Description: fmt.Sprintf("Remanescente da fatura anterior - %s", originalInvoice.CreditCard.Name),
		Amount:      remainder,
		Date:        &date,
		UserID:      originalInvoice.UserID,
		IsPaid:      false,
		WalletID:    originalInvoice.WalletID,
		CreditCardInfo: &domain.CreditCardMovement{
			InvoiceID:    nextInvoice.ID,
			CreditCardID: nextInvoice.CreditCardID,
		},
		TypePayment: domain.TypePaymentInvoiceRemainder,
		CategoryID:  &defaultCreditCardCategoryID,
	}
}
