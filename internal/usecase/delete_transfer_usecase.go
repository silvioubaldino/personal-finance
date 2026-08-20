package usecase

import (
	"context"
	"fmt"

	"personal-finance/internal/infrastructure/repository/transaction"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DeleteTransfer struct {
	movementRepo MovementRepository
	walletRepo   WalletRepository
	txManager    transaction.Manager
}

func NewDeleteTransfer(
	movementRepo MovementRepository,
	walletRepo WalletRepository,
	txManager transaction.Manager,
) DeleteTransfer {
	return DeleteTransfer{
		movementRepo: movementRepo,
		walletRepo:   walletRepo,
		txManager:    txManager,
	}
}

// Execute deletes both legs of an internal transfer pair atomically, reverting the wallet
// balances it moved if the transfer was paid. A transfer is never deleted one leg at a
// time: half a transfer is an invalid state (one wallet balance would stay moved while the
// other reverts).
func (u *DeleteTransfer) Execute(ctx context.Context, pairID uuid.UUID) error {
	pairMovements, err := u.movementRepo.FindByPairID(ctx, pairID)
	if err != nil {
		return fmt.Errorf("error finding pair movements: %w", err)
	}

	origin, destination, err := identifyTransferPair(pairMovements)
	if err != nil {
		return err
	}

	return u.txManager.WithTransaction(ctx, func(tx *gorm.DB) error {
		if origin.IsPaid {
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
		}

		if err := u.movementRepo.Delete(ctx, tx, *origin.ID); err != nil {
			return fmt.Errorf("error deleting origin movement: %w", err)
		}

		if err := u.movementRepo.Delete(ctx, tx, *destination.ID); err != nil {
			return fmt.Errorf("error deleting destination movement: %w", err)
		}

		return nil
	})
}
