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

type UpdateTransferInput struct {
	PairID              uuid.UUID `json:"pair_id"`
	OriginWalletID      uuid.UUID `json:"origin_wallet_id"`
	DestinationWalletID uuid.UUID `json:"destination_wallet_id"`
	Amount              float64   `json:"amount"`
	Date                time.Time `json:"date"`
	Description         string    `json:"description"`
}

type UpdateTransfer struct {
	movementRepo MovementRepository
	walletRepo   WalletRepository
	txManager    transaction.Manager
}

func NewUpdateTransfer(
	movementRepo MovementRepository,
	walletRepo WalletRepository,
	txManager transaction.Manager,
) UpdateTransfer {
	return UpdateTransfer{
		movementRepo: movementRepo,
		walletRepo:   walletRepo,
		txManager:    txManager,
	}
}

// Execute updates both legs of an internal transfer pair atomically. A transfer is never
// updated one leg at a time: half a transfer is an invalid state (one wallet balance would
// move without the other).
func (u *UpdateTransfer) Execute(ctx context.Context, input UpdateTransferInput) (TransferOutput, error) {
	if err := u.validateInput(input); err != nil {
		return TransferOutput{}, err
	}

	pairMovements, err := u.movementRepo.FindByPairID(ctx, input.PairID)
	if err != nil {
		return TransferOutput{}, fmt.Errorf("error finding pair movements: %w", err)
	}

	if len(pairMovements) != 2 {
		return TransferOutput{}, ErrTransferPairNotFound
	}

	oldOrigin, oldDestination, err := u.identifyMovements(pairMovements)
	if err != nil {
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

	outCategoryID := uuid.MustParse(domain.InternalTransferOutCategoryID)
	inCategoryID := uuid.MustParse(domain.InternalTransferInCategoryID)
	newDescription := u.buildDescription(input.Description, originWallet.Description, destinationWallet.Description)

	newOrigin := oldOrigin
	newOrigin.Description = newDescription
	newOrigin.Amount = -input.Amount
	newOrigin.Date = &input.Date
	newOrigin.WalletID = &input.OriginWalletID
	newOrigin.CategoryID = &outCategoryID

	newDestination := oldDestination
	newDestination.Description = newDescription
	newDestination.Amount = input.Amount
	newDestination.Date = &input.Date
	newDestination.WalletID = &input.DestinationWalletID
	newDestination.CategoryID = &inCategoryID

	var result TransferOutput

	err = u.txManager.WithTransaction(ctx, func(tx *gorm.DB) error {
		if oldOrigin.IsPaid {
			if err := u.revertBalances(ctx, tx, oldOrigin, oldDestination); err != nil {
				return err
			}

			refreshedOriginWallet, err := u.walletRepo.FindByID(ctx, &input.OriginWalletID)
			if err != nil {
				return fmt.Errorf("error finding origin wallet: %w", err)
			}
			if !refreshedOriginWallet.HasSufficientBalance(-input.Amount) {
				return ErrInsufficientBalance
			}
		}

		updatedOrigin, err := u.movementRepo.Update(ctx, tx, *oldOrigin.ID, newOrigin)
		if err != nil {
			return fmt.Errorf("error updating origin movement: %w", err)
		}

		updatedDestination, err := u.movementRepo.Update(ctx, tx, *oldDestination.ID, newDestination)
		if err != nil {
			return fmt.Errorf("error updating destination movement: %w", err)
		}

		if oldOrigin.IsPaid {
			if err := u.applyBalances(ctx, tx, &input.OriginWalletID, &input.DestinationWalletID, input.Amount); err != nil {
				return err
			}
		}

		result = TransferOutput{
			PairID:              input.PairID,
			OriginMovement:      updatedOrigin,
			DestinationMovement: updatedDestination,
		}

		return nil
	})
	if err != nil {
		return TransferOutput{}, err
	}

	return result, nil
}

func (u *UpdateTransfer) validateInput(input UpdateTransferInput) error {
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

func (u *UpdateTransfer) identifyMovements(movements domain.MovementList) (origin, destination domain.Movement, err error) {
	for _, m := range movements {
		if m.TypePayment != domain.TypePaymentInternalTransfer {
			return domain.Movement{}, domain.Movement{}, ErrMovementNotInternalTransfer
		}
		if m.Amount < 0 {
			origin = m
		} else {
			destination = m
		}
	}

	if origin.ID == nil || destination.ID == nil {
		return domain.Movement{}, domain.Movement{}, ErrTransferPairNotFound
	}

	return origin, destination, nil
}

func (u *UpdateTransfer) revertBalances(ctx context.Context, tx *gorm.DB, origin, destination domain.Movement) error {
	originWallet, err := u.walletRepo.FindByID(ctx, origin.WalletID)
	if err != nil {
		return fmt.Errorf("error finding origin wallet: %w", err)
	}
	if err := u.walletRepo.UpdateAmount(ctx, tx, origin.WalletID, originWallet.Balance-origin.Amount); err != nil {
		return fmt.Errorf("error reverting origin wallet balance: %w", err)
	}

	destWallet, err := u.walletRepo.FindByID(ctx, destination.WalletID)
	if err != nil {
		return fmt.Errorf("error finding destination wallet: %w", err)
	}
	if err := u.walletRepo.UpdateAmount(ctx, tx, destination.WalletID, destWallet.Balance-destination.Amount); err != nil {
		return fmt.Errorf("error reverting destination wallet balance: %w", err)
	}

	return nil
}

func (u *UpdateTransfer) applyBalances(ctx context.Context, tx *gorm.DB, originWalletID, destWalletID *uuid.UUID, amount float64) error {
	originWallet, err := u.walletRepo.FindByID(ctx, originWalletID)
	if err != nil {
		return fmt.Errorf("error finding origin wallet: %w", err)
	}
	if err := u.walletRepo.UpdateAmount(ctx, tx, originWalletID, originWallet.Balance-amount); err != nil {
		return fmt.Errorf("error updating origin wallet balance: %w", err)
	}

	destWallet, err := u.walletRepo.FindByID(ctx, destWalletID)
	if err != nil {
		return fmt.Errorf("error finding destination wallet: %w", err)
	}
	if err := u.walletRepo.UpdateAmount(ctx, tx, destWalletID, destWallet.Balance+amount); err != nil {
		return fmt.Errorf("error updating destination wallet balance: %w", err)
	}

	return nil
}

func (u *UpdateTransfer) buildDescription(description, originWallet, destinationWallet string) string {
	if description != "" {
		return fmt.Sprintf("Transferência de %s para %s - %s", originWallet, destinationWallet, description)
	}
	return fmt.Sprintf("Transferência de %s para %s", originWallet, destinationWallet)
}
