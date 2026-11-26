package usecase

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/infrastructure/repository"
	"personal-finance/pkg/clock"
	"personal-finance/pkg/metrics"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AdvancePaymentInput é o pedido de adiantamento de uma Invoice (AYD-007@context).
type AdvancePaymentInput struct {
	Amount      float64
	WalletID    *uuid.UUID
	PaymentDate *time.Time
	// IdempotencyHash é opcional e só vem do import de extrato (AYD-007@context).
	IdempotencyHash *string
}

// AdvancePayment registra o pagamento de parte de uma Invoice ainda não paga: debita a
// Wallet e libera limite do CreditCard na data do adiantamento, e a Invoice segue aberta
// pelo restante. Adiantar o restante inteiro (ou mais) é proibido: quitar é pelo Pay.
func (uc Invoice) AdvancePayment(ctx context.Context, invoiceID uuid.UUID, input AdvancePaymentInput) (domain.Invoice, error) {
	invoice, err := uc.repo.FindByID(ctx, invoiceID)
	if err != nil {
		return domain.Invoice{}, fmt.Errorf("error finding invoice: %w", err)
	}

	if invoice.IsPaid {
		return domain.Invoice{}, fmt.Errorf("advance invoice payment: %w", ErrAdvanceInvoiceAlreadyPaid)
	}

	amount := input.Amount
	if amount > 0 {
		amount = -amount
	}
	if amount == 0 {
		return domain.Invoice{}, fmt.Errorf("advance invoice payment: %w", ErrInvalidAdvancePaymentAmount)
	}

	advancePaid, err := uc.advancePaidAmount(ctx, *invoice.ID)
	if err != nil {
		return domain.Invoice{}, err
	}
	remaining := domain.RoundCents(invoice.Amount - advancePaid)
	if remaining >= 0 || centsOf(amount) >= centsOf(remaining) {
		return domain.Invoice{}, fmt.Errorf("advance invoice payment: %w", ErrAdvancePaymentSettlesInvoice)
	}

	paymentDate := clock.Now(ctx)
	if input.PaymentDate != nil {
		paymentDate = *input.PaymentDate
	}
	if dateOnly(paymentDate).After(dateOnly(invoice.DueDate)) {
		return domain.Invoice{}, fmt.Errorf("advance invoice payment: %w", ErrAdvancePaymentAfterDueDate)
	}

	walletID := input.WalletID
	if walletID == nil {
		walletID = invoice.WalletID
	}
	if walletID == nil {
		return domain.Invoice{}, fmt.Errorf("advance invoice payment: %w", ErrInvoiceWithoutWallet)
	}

	wallet, err := uc.walletRepo.FindByID(ctx, walletID)
	if err != nil {
		return domain.Invoice{}, fmt.Errorf("error finding wallet: %w", err)
	}

	if err := wallet.Pay(amount); err != nil {
		return domain.Invoice{}, fmt.Errorf("error paying invoice in advance: %w", err)
	}

	movement := buildAdvancePaymentMovement(invoice, *walletID, amount, paymentDate)
	movement.IdempotencyHash = input.IdempotencyHash

	err = uc.txManager.WithTransaction(ctx, func(tx *gorm.DB) error {
		if err := uc.walletRepo.UpdateAmount(ctx, tx, walletID, wallet.Balance); err != nil {
			return fmt.Errorf("error updating wallet balance: %w", err)
		}

		if _, err := uc.movementRepo.Add(ctx, tx, movement); err != nil {
			return fmt.Errorf("error creating advance payment movement: %w", err)
		}

		if _, err := uc.creditCardRepo.UpdateLimitDelta(ctx, tx, *invoice.CreditCardID, -amount); err != nil {
			return fmt.Errorf("error updating credit card limit: %w", err)
		}

		return nil
	})
	if err != nil {
		return domain.Invoice{}, err
	}

	metrics.IncBusiness(ctx, "biz_invoice_advance_payments_total", 1)

	return uc.withAdvancePayment(ctx, invoice)
}

// RevertAdvancePayment desfaz um adiantamento: devolve o valor à Wallet, volta a consumir
// o limite e apaga o Movement. Só com a Invoice não paga (desfaz-se a quitação primeiro).
func (uc Invoice) RevertAdvancePayment(ctx context.Context, invoiceID, movementID uuid.UUID) (domain.Invoice, error) {
	invoice, err := uc.repo.FindByID(ctx, invoiceID)
	if err != nil {
		return domain.Invoice{}, fmt.Errorf("error finding invoice: %w", err)
	}

	if invoice.IsPaid {
		return domain.Invoice{}, fmt.Errorf("revert advance invoice payment: %w", ErrAdvanceInvoiceAlreadyPaid)
	}

	movement, err := uc.movementRepo.FindByID(ctx, movementID)
	if err != nil {
		if errors.Is(err, repository.ErrMovementNotFound) {
			return domain.Invoice{}, fmt.Errorf("revert advance invoice payment: %w", ErrAdvancePaymentNotFound)
		}
		return domain.Invoice{}, fmt.Errorf("error finding advance payment: %w", err)
	}

	if movement.TypePayment != domain.TypePaymentInvoiceAdvancePayment ||
		movement.CreditCardInfo == nil ||
		movement.CreditCardInfo.InvoiceID == nil ||
		*movement.CreditCardInfo.InvoiceID != *invoice.ID {
		return domain.Invoice{}, fmt.Errorf("revert advance invoice payment: %w", ErrAdvancePaymentNotFound)
	}

	wallet, err := uc.walletRepo.FindByID(ctx, movement.WalletID)
	if err != nil {
		return domain.Invoice{}, fmt.Errorf("error finding wallet: %w", err)
	}

	err = uc.txManager.WithTransaction(ctx, func(tx *gorm.DB) error {
		if err := wallet.RevertPayment(movement.Amount); err != nil {
			return fmt.Errorf("error reverting advance payment: %w", err)
		}

		if err := uc.walletRepo.UpdateAmount(ctx, tx, movement.WalletID, wallet.Balance); err != nil {
			return fmt.Errorf("error updating wallet balance: %w", err)
		}

		if err := uc.movementRepo.Delete(ctx, tx, movementID); err != nil {
			return fmt.Errorf("error deleting advance payment movement: %w", err)
		}

		if _, err := uc.creditCardRepo.UpdateLimitDelta(ctx, tx, *invoice.CreditCardID, movement.Amount); err != nil {
			return fmt.Errorf("error updating credit card limit: %w", err)
		}

		return nil
	})
	if err != nil {
		return domain.Invoice{}, err
	}

	return uc.withAdvancePayment(ctx, invoice)
}

// advancePaidAmount soma os adiantamentos da Invoice (mesmo sinal de Invoice.Amount).
func (uc Invoice) advancePaidAmount(ctx context.Context, invoiceID uuid.UUID) (float64, error) {
	movements, err := uc.movementRepo.FindAdvancePaymentsByInvoiceIDs(ctx, []uuid.UUID{invoiceID})
	if err != nil {
		return 0, fmt.Errorf("error finding advance payments: %w", err)
	}

	var total float64
	for _, movement := range movements {
		total += movement.Amount
	}
	return domain.RoundCents(total), nil
}

// withAdvancePayment preenche os derivados (advance_paid_amount, remaining_amount e a
// lista de adiantamentos) de uma Invoice.
func (uc Invoice) withAdvancePayment(ctx context.Context, invoice domain.Invoice) (domain.Invoice, error) {
	invoices, err := uc.withAdvancePayments(ctx, []domain.Invoice{invoice})
	if err != nil {
		return domain.Invoice{}, err
	}
	return invoices[0], nil
}

// withAdvancePayments faz o mesmo para várias Invoices com uma só consulta.
func (uc Invoice) withAdvancePayments(ctx context.Context, invoices []domain.Invoice) ([]domain.Invoice, error) {
	ids := make([]uuid.UUID, 0, len(invoices))
	for _, invoice := range invoices {
		if invoice.ID != nil {
			ids = append(ids, *invoice.ID)
		}
	}
	if len(ids) == 0 {
		return invoices, nil
	}

	movements, err := uc.movementRepo.FindAdvancePaymentsByInvoiceIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("error finding advance payments: %w", err)
	}

	byInvoice := make(map[uuid.UUID]domain.MovementList, len(ids))
	for _, movement := range movements {
		if movement.CreditCardInfo == nil || movement.CreditCardInfo.InvoiceID == nil {
			continue
		}
		invoiceID := *movement.CreditCardInfo.InvoiceID
		byInvoice[invoiceID] = append(byInvoice[invoiceID], movement)
	}

	result := make([]domain.Invoice, len(invoices))
	for i, invoice := range invoices {
		var payments []domain.InvoiceAdvancePayment
		if invoice.ID != nil {
			payments = domain.NewInvoiceAdvancePayments(byInvoice[*invoice.ID])
		}
		result[i] = invoice.WithAdvancePayments(payments)
	}
	return result, nil
}

func buildAdvancePaymentMovement(invoice domain.Invoice, walletID uuid.UUID, amount float64, date time.Time) domain.Movement {
	defaultCreditCardCategoryID := uuid.MustParse("d47cc960-f08d-480e-bf01-f4ec5ddfcb8b")

	return domain.Movement{
		Description: fmt.Sprintf("Adiantamento da fatura - %s", invoice.CreditCard.Name),
		Amount:      amount,
		Date:        &date,
		UserID:      invoice.UserID,
		IsPaid:      true,
		WalletID:    &walletID,
		CreditCardInfo: &domain.CreditCardMovement{
			InvoiceID:    invoice.ID,
			CreditCardID: invoice.CreditCardID,
		},
		TypePayment: domain.TypePaymentInvoiceAdvancePayment,
		CategoryID:  &defaultCreditCardCategoryID,
	}
}

func centsOf(value float64) int64 {
	return int64(math.Round(math.Abs(value) * 100))
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
