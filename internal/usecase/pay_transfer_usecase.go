package usecase

import (
	"context"
	"fmt"

	"personal-finance/internal/infrastructure/repository/transaction"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PayTransfer struct {
	movementRepo MovementRepository
	walletRepo   WalletRepository
	txManager    transaction.Manager
}

func NewPayTransfer(
	movementRepo MovementRepository,
	walletRepo WalletRepository,
	txManager transaction.Manager,
) PayTransfer {
	return PayTransfer{
		movementRepo: movementRepo,
		walletRepo:   walletRepo,
		txManager:    txManager,
	}
}

// Execute settles both legs of an internal transfer pair atomically, moving both wallet
// balances. A transfer is never settled one leg at a time: half a transfer is an invalid
// state (one balance moves while the other stays put).
func (u *PayTransfer) Execute(ctx context.Context, pairID uuid.UUID) (TransferOutput, error) {
	return u.settle(ctx, pairID, true, ErrMovementAlreadyPaid)
}

// Revert unsettles both legs of an internal transfer pair atomically, undoing the balance
// movement of both wallets.
func (u *PayTransfer) Revert(ctx context.Context, pairID uuid.UUID) (TransferOutput, error) {
	return u.settle(ctx, pairID, false, ErrMovementNotPaid)
}

func (u *PayTransfer) settle(
	ctx context.Context,
	pairID uuid.UUID,
	isPaid bool,
	alreadyInStateErr error,
) (TransferOutput, error) {
	pairMovements, err := u.movementRepo.FindByPairID(ctx, pairID)
	if err != nil {
		return TransferOutput{}, fmt.Errorf("error finding pair movements: %w", err)
	}

	origin, destination, err := identifyTransferPair(pairMovements)
	if err != nil {
		return TransferOutput{}, err
	}

	// A pair whose legs disagree on is_paid has one wallet already moved and the other not,
	// so there is no balance delta that settles it correctly. Reject it instead of guessing.
	if origin.IsPaid != destination.IsPaid {
		return TransferOutput{}, ErrTransferPairInconsistentPayment
	}

	if origin.IsPaid == isPaid {
		return TransferOutput{}, alreadyInStateErr
	}

	// Paying applies the movements to the balances; reverting undoes them. The wallet being
	// debited is the origin when paying and the destination when reverting — that is the one
	// that can end up negative.
	sign := 1.0
	debitedWalletID := origin.WalletID
	if !isPaid {
		sign = -1.0
		debitedWalletID = destination.WalletID
	}

	var result TransferOutput

	err = u.txManager.WithTransaction(ctx, func(tx *gorm.DB) error {
		balances := newWalletBalanceChanges(u.walletRepo)
		if err := balances.addAll(ctx, []walletBalanceChange{
			{walletID: origin.WalletID, delta: sign * origin.Amount},
			{walletID: destination.WalletID, delta: sign * destination.Amount},
		}); err != nil {
			return err
		}

		if balances.balanceOf(debitedWalletID) < 0 {
			return ErrInsufficientBalance
		}

		origin.IsPaid = isPaid
		destination.IsPaid = isPaid

		updatedOrigin, err := u.movementRepo.UpdateIsPaid(ctx, tx, *origin.ID, origin)
		if err != nil {
			return fmt.Errorf("error updating origin movement: %w", err)
		}

		updatedDestination, err := u.movementRepo.UpdateIsPaid(ctx, tx, *destination.ID, destination)
		if err != nil {
			return fmt.Errorf("error updating destination movement: %w", err)
		}

		if err := balances.flush(ctx, tx); err != nil {
			return err
		}

		result = TransferOutput{
			PairID:              pairID,
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
