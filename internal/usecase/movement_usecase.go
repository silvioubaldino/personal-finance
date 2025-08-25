package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/infrastructure/repository/transaction"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type (
	MovementRepository interface {
		Add(ctx context.Context, tx *gorm.DB, movement domain.Movement) (domain.Movement, error)
		FindByPeriod(ctx context.Context, period domain.Period) (domain.MovementList, error)
		FindByID(ctx context.Context, id uuid.UUID) (domain.Movement, error)
		UpdateIsPaid(ctx context.Context, tx *gorm.DB, id uuid.UUID, movement domain.Movement) (domain.Movement, error)
		UpdateOne(ctx context.Context, tx *gorm.DB, id uuid.UUID, movement domain.Movement) (domain.Movement, error)
	}

	RecurrentRepository interface {
		Add(ctx context.Context, tx *gorm.DB, recurrent domain.RecurrentMovement) (domain.RecurrentMovement, error)
		FindByMonth(ctx context.Context, month time.Time) ([]domain.RecurrentMovement, error)
		FindByID(ctx context.Context, id uuid.UUID) (domain.RecurrentMovement, error)
	}

	InvoiceUseCase interface {
		FindOrCreateInvoiceForMovement(ctx context.Context, invoiceID *uuid.UUID, creditCardID uuid.UUID, movementDate time.Time) (domain.Invoice, error)
	}

	Movement struct {
		movementRepo    MovementRepository
		recurrentRepo   RecurrentRepository
		walletRepo      WalletRepository
		subCategoryRepo SubCategoryRepository
		invoiceRepo     InvoiceRepository
		invoiceUseCase  InvoiceUseCase
		creditCardRepo  CreditCardRepository
		txManager       transaction.Manager
	}
)

func NewMovement(
	movementRepo MovementRepository,
	recurrentRepo RecurrentRepository,
	walletRepo WalletRepository,
	subCategoryRepo SubCategoryRepository,
	invoiceRepo InvoiceRepository,
	invoiceUseCase InvoiceUseCase,
	creditCardRepo CreditCardRepository,
	txManager transaction.Manager,
) Movement {
	return Movement{
		movementRepo:    movementRepo,
		recurrentRepo:   recurrentRepo,
		walletRepo:      walletRepo,
		subCategoryRepo: subCategoryRepo,
		invoiceRepo:     invoiceRepo,
		invoiceUseCase:  invoiceUseCase,
		creditCardRepo:  creditCardRepo,
		txManager:       txManager,
	}
}

func (u *Movement) isSubCategoryValid(ctx context.Context, subCategoryID, categoryID *uuid.UUID) error {
	if subCategoryID == nil {
		return nil
	}

	isSubCategoryValid, err := u.subCategoryRepo.IsSubCategoryBelongsToCategory(ctx, *subCategoryID, *categoryID)
	if err != nil {
		return err
	}

	if !isSubCategoryValid {
		return domain.WrapInvalidInput(
			domain.New("subcategory does not belong to the provided category"),
			"validate subcategory",
		)
	}

	return nil
}

func (u *Movement) updateWalletBalance(ctx context.Context, tx *gorm.DB, walletID *uuid.UUID, amount float64) error {
	wallet, err := u.walletRepo.FindByID(ctx, walletID)
	if err != nil {
		return err
	}

	if !wallet.HasSufficientBalance(amount) {
		return ErrInsufficientBalance
	}

	wallet.Balance += amount

	return u.walletRepo.UpdateAmount(ctx, tx, wallet.ID, wallet.Balance)
}

func (u *Movement) handleCreditCardMovement(ctx context.Context, tx *gorm.DB, movement *domain.Movement) error {
	if movement.CreditCardInfo == nil {
		return fmt.Errorf("credit_card_id is required for credit card movements")
	}

	invoice, err := u.invoiceUseCase.FindOrCreateInvoiceForMovement(ctx, movement.CreditCardInfo.InvoiceID, *movement.CreditCardInfo.CreditCardID, *movement.Date)
	if err != nil {
		return fmt.Errorf("error finding/creating invoice: %w", err)
	}
	movement.CreditCardInfo.InvoiceID = invoice.ID

	movement.IsPaid = false

	newAmount := invoice.Amount + movement.Amount
	_, err = u.invoiceRepo.UpdateAmount(ctx, tx, *movement.CreditCardInfo.InvoiceID, newAmount) // TODO update credit card limit
	if err != nil {
		return fmt.Errorf("error updating invoice amount: %w", err)
	}

	return nil
}

func (u *Movement) Add(ctx context.Context, movement domain.Movement) (domain.Movement, error) {
	err := u.isSubCategoryValid(ctx, movement.SubCategoryID, movement.CategoryID)
	if err != nil {
		return domain.Movement{}, err
	}

	var result domain.Movement

	err = u.txManager.WithTransaction(ctx, func(tx *gorm.DB) error {
		if movement.IsCreditCardMovement() {
			if err := u.handleCreditCardMovement(ctx, tx, &movement); err != nil {
				return err
			}
		}

		if movement.ShouldCreateRecurrent() {
			recurrent := domain.ToRecurrentMovement(movement)

			createdRecurrent, err := u.recurrentRepo.Add(ctx, tx, recurrent)
			if err != nil {
				return err
			}

			movement.RecurrentID = createdRecurrent.ID
		}

		createdMovement, err := u.movementRepo.Add(ctx, tx, movement)
		if err != nil {
			return err
		}

		if movement.IsPaid && !movement.IsCreditCardMovement() {
			err = u.updateWalletBalance(ctx, tx, movement.WalletID, movement.Amount)
			if err != nil {
				return err
			}
		}

		result = createdMovement
		return nil
	})
	if err != nil {
		return domain.Movement{}, err
	}

	return result, nil
}

func (u *Movement) FindByPeriod(ctx context.Context, period domain.Period) (domain.MovementList, error) {
	movements, err := u.movementRepo.FindByPeriod(ctx, period)
	if err != nil {
		return domain.MovementList{}, err
	}

	recurrents, err := u.recurrentRepo.FindByMonth(ctx, period.To)
	if err != nil {
		return domain.MovementList{}, fmt.Errorf("error to find recurrents: %w", err)
	}

	invoices, err := u.invoiceRepo.FindOpenByMonth(ctx, period.To)
	if err != nil {
		return domain.MovementList{}, fmt.Errorf("error to find invoices: %w", err)
	}

	movementsWithRecurrents := mergeMovementsWithRecurrents(movements, recurrents, period.To)
	movementsWithInvoices, err := u.mergeMovementsWithInvoices(ctx, movementsWithRecurrents, invoices)
	if err != nil {
		return domain.MovementList{}, fmt.Errorf("error to merge movements with invoices: %w", err)
	}

	return movementsWithInvoices, nil
}

func (u *Movement) Pay(ctx context.Context, id uuid.UUID, date time.Time) (domain.Movement, error) {
	var result domain.Movement
	var err error

	txError := u.txManager.WithTransaction(ctx, func(tx *gorm.DB) error {
		if result, err = u.payMovement(ctx, tx, id, date); err != nil {
			return fmt.Errorf("error paying movement with id: %s: %w", id, err)
		}

		if err = u.updateWalletBalance(ctx, tx, result.WalletID, result.Amount); err != nil {
			return fmt.Errorf("error updating wallet: %w", err)
		}
		return nil
	})

	if txError != nil {
		return domain.Movement{}, txError
	}
	return result, nil
}

func (u *Movement) payMovement(ctx context.Context, tx *gorm.DB, id uuid.UUID, date time.Time) (domain.Movement, error) {
	movement, err := u.movementRepo.FindByID(ctx, id)
	if err != nil {
		if !errors.Is(err, domain.ErrNotFound) {
			return domain.Movement{}, err
		}

		recurrent, err := u.recurrentRepo.FindByID(ctx, id)
		if err != nil {
			return domain.Movement{}, err
		}

		if date.IsZero() {
			return domain.Movement{}, ErrDateRequired
		}

		mov := domain.FromRecurrentMovement(recurrent, date)
		mov.IsPaid = true

		createdMovement, err := u.movementRepo.Add(ctx, tx, mov)
		if err != nil {
			return domain.Movement{}, err
		}

		return createdMovement, nil
	}

	if movement.IsPaid {
		return domain.Movement{}, ErrMovementAlreadyPaid
	}

	if movement.IsCreditCardMovement() {
		return domain.Movement{}, ErrCreditCardPay
	}

	movement.IsPaid = true

	updatedMovement, err := u.movementRepo.UpdateIsPaid(ctx, tx, id, movement)
	if err != nil {
		return domain.Movement{}, err
	}

	return updatedMovement, nil
}

func (u *Movement) RevertPay(ctx context.Context, id uuid.UUID) (domain.Movement, error) {
	var result domain.Movement

	txError := u.txManager.WithTransaction(ctx, func(tx *gorm.DB) error {
		movement, err := u.movementRepo.FindByID(ctx, id)
		if err != nil {
			return fmt.Errorf("error finding movement with id: %s: %w", id, err)
		}

		if !movement.IsPaid {
			return ErrMovementNotPaid
		}

		movement.IsPaid = false

		result, err = u.movementRepo.UpdateIsPaid(ctx, tx, id, movement)
		if err != nil {
			return fmt.Errorf("error updating movement: %w", err)
		}

		if err = u.updateWalletBalance(ctx, tx, result.WalletID, result.ReverseAmount()); err != nil {
			return fmt.Errorf("error updating wallet: %w", err)
		}

		return nil
	})

	if txError != nil {
		return domain.Movement{}, txError
	}
	return result, nil
}

func (u *Movement) mergeMovementsWithInvoices(
	ctx context.Context,
	movements domain.MovementList,
	invoices []domain.Invoice,
) (domain.MovementList, error) {
	for _, invoice := range invoices {
		if !invoice.IsPaid {
			creditCardName, err := u.creditCardRepo.FindNameByID(ctx, *invoice.CreditCardID)
			if err != nil {
				return domain.MovementList{}, err
			}

			virtualMovement := domain.Movement{
				ID:          invoice.ID,
				Description: buildCreditCardDescription(creditCardName),
				Amount:      invoice.Amount,
				Date:        &invoice.DueDate,
				UserID:      invoice.UserID,
				CreditCardInfo: &domain.CreditCardMovement{
					InvoiceID:    invoice.ID,
					CreditCardID: invoice.CreditCardID,
				},
				WalletID:    invoice.WalletID,
				TypePayment: domain.TypePaymentInvoicePayment,
			}
			movements = append(movements, virtualMovement)
		}
	}

	return movements, nil
}

func mergeMovementsWithRecurrents(
	movements domain.MovementList,
	recurrents []domain.RecurrentMovement,
	date time.Time,
) domain.MovementList {
	recurrentMap := make(map[uuid.UUID]struct{}, len(recurrents))
	for i, mov := range movements {
		if mov.RecurrentID != nil {
			movements[i].IsRecurrent = true
			recurrentMap[*mov.RecurrentID] = struct{}{}
		}
	}

	for _, recurrent := range recurrents {
		if _, ok := recurrentMap[*recurrent.ID]; !ok {
			mov := domain.FromRecurrentMovement(recurrent, date)
			mov.ID = mov.RecurrentID
			movements = append(movements, mov)
		}
	}

	return movements
}

func buildCreditCardDescription(creditCardName string) string {
	return fmt.Sprintf("Pagamento da fatura %s", creditCardName)
}
