package usecase

import (
	"context"
	"fmt"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/infrastructure/repository/transaction"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TransferInput struct {
	OriginWalletID      uuid.UUID `json:"origin_wallet_id"`
	DestinationWalletID uuid.UUID `json:"destination_wallet_id"`
	Amount              float64   `json:"amount"`
	Date                time.Time `json:"date"`
	Description         string    `json:"description"`
	IsPaid              bool      `json:"is_paid"`
}

type TransferOutput struct {
	PairID              uuid.UUID       `json:"pair_id"`
	OriginMovement      domain.Movement `json:"origin_movement"`
	DestinationMovement domain.Movement `json:"destination_movement"`
}

type Transfer struct {
	movementRepo MovementRepository
	walletRepo   WalletRepository
	txManager    transaction.Manager
}

func NewTransfer(
	movementRepo MovementRepository,
	walletRepo WalletRepository,
	txManager transaction.Manager,
) Transfer {
	return Transfer{
		movementRepo: movementRepo,
		walletRepo:   walletRepo,
		txManager:    txManager,
	}
}

func (u *Transfer) Execute(ctx context.Context, input TransferInput) (TransferOutput, error) {
	if err := u.validateInput(input); err != nil {
		return TransferOutput{}, err
	}

	originWallet, err := u.walletRepo.FindByID(ctx, &input.OriginWalletID)
	if err != nil {
		return TransferOutput{}, fmt.Errorf("error finding origin wallet: %w", err)
	}

	destinationWallet, err := u.walletRepo.FindByID(ctx, &input.DestinationWalletID)
	if err != nil {
		return TransferOutput{}, fmt.Errorf("error finding destination wallet: %w", err)
	}

	if input.IsPaid && !originWallet.HasSufficientBalance(-input.Amount) {
		return TransferOutput{}, ErrInsufficientBalance
	}

	pairID := uuid.New()
	outCategoryID := uuid.MustParse(domain.InternalTransferOutCategoryID)
	inCategoryID := uuid.MustParse(domain.InternalTransferInCategoryID)

	originMovement := domain.Movement{
		Description: u.buildDescription(input.Description, originWallet.Description, destinationWallet.Description),
		Amount:      -input.Amount,
		Date:        &input.Date,
		IsPaid:      input.IsPaid,
		PairID:      &pairID,
		WalletID:    &input.OriginWalletID,
		TypePayment: domain.TypePaymentInternalTransfer,
		CategoryID:  &outCategoryID,
	}

	destinationMovement := domain.Movement{
		Description: u.buildDescription(input.Description, originWallet.Description, destinationWallet.Description),
		Amount:      input.Amount,
		Date:        &input.Date,
		IsPaid:      input.IsPaid,
		PairID:      &pairID,
		WalletID:    &input.DestinationWalletID,
		TypePayment: domain.TypePaymentInternalTransfer,
		CategoryID:  &inCategoryID,
	}

	var result TransferOutput

	err = u.txManager.WithTransaction(ctx, func(tx *gorm.DB) error {
		createdOrigin, err := u.movementRepo.Add(ctx, tx, originMovement)
		if err != nil {
			return fmt.Errorf("error creating origin movement: %w", err)
		}

		createdDestination, err := u.movementRepo.Add(ctx, tx, destinationMovement)
		if err != nil {
			return fmt.Errorf("error creating destination movement: %w", err)
		}

		if input.IsPaid {
			if err := u.updateWalletBalance(ctx, tx, &input.OriginWalletID, -input.Amount); err != nil {
				return fmt.Errorf("error updating origin wallet balance: %w", err)
			}

			if err := u.updateWalletBalance(ctx, tx, &input.DestinationWalletID, input.Amount); err != nil {
				return fmt.Errorf("error updating destination wallet balance: %w", err)
			}
		}

		result = TransferOutput{
			PairID:              pairID,
			OriginMovement:      createdOrigin,
			DestinationMovement: createdDestination,
		}

		return nil
	})

	if err != nil {
		return TransferOutput{}, err
	}

	return result, nil
}

func (u *Transfer) validateInput(input TransferInput) error {
	if input.OriginWalletID == input.DestinationWalletID {
		return ErrSameWalletTransfer
	}

	if input.Amount <= 0 {
		return ErrInvalidTransferAmount
	}

	if input.Date.IsZero() {
		return ErrDateRequired
	}

	return nil
}

func (u *Transfer) updateWalletBalance(ctx context.Context, tx *gorm.DB, walletID *uuid.UUID, amount float64) error {
	wallet, err := u.walletRepo.FindByID(ctx, walletID)
	if err != nil {
		return err
	}

	newBalance := wallet.Balance + amount

	return u.walletRepo.UpdateAmount(ctx, tx, walletID, newBalance)
}

func (u *Transfer) buildDescription(description, originWallet, destinationWallet string) string {
	if description != "" {
		return fmt.Sprintf("Transferência de %s para %s - %s", originWallet, destinationWallet, description)
	}
	return fmt.Sprintf("Transferência de %s para %s", originWallet, destinationWallet)
}
