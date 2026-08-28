package usecase

import (
	"context"
	"fmt"

	"personal-finance/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// identifyTransferPair splits a pair into its origin (negative) and destination (positive)
// legs, rejecting anything that is not a well-formed internal transfer pair.
func identifyTransferPair(movements domain.MovementList) (origin, destination domain.Movement, err error) {
	if len(movements) != 2 {
		return domain.Movement{}, domain.Movement{}, ErrTransferPairNotFound
	}

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

type walletBalanceChange struct {
	walletID *uuid.UUID
	delta    float64
}

// walletBalanceChanges accumulates the balance changes of an operation in memory so that
// each wallet is read once and written once.
//
// It exists because WalletRepository.FindByID reads through the repository's own connection
// and not through the transaction's *gorm.DB: a balance written earlier inside the
// transaction is invisible to a later FindByID. Recomputing a balance between two writes of
// the same wallet would restart from the pre-transaction value and silently drop the first
// write.
type walletBalanceChanges struct {
	repo     WalletRepository
	balances map[uuid.UUID]float64
	wallets  []*uuid.UUID
}

func newWalletBalanceChanges(repo WalletRepository) *walletBalanceChanges {
	return &walletBalanceChanges{
		repo:     repo,
		balances: make(map[uuid.UUID]float64),
	}
}

// addAll applies every delta to its wallet's running balance, loading the stored balance the
// first time a wallet is seen.
func (c *walletBalanceChanges) addAll(ctx context.Context, changes []walletBalanceChange) error {
	for _, change := range changes {
		if _, loaded := c.balances[*change.walletID]; !loaded {
			wallet, err := c.repo.FindByID(ctx, change.walletID)
			if err != nil {
				return fmt.Errorf("error finding wallet: %w", err)
			}

			c.balances[*change.walletID] = wallet.Balance
			c.wallets = append(c.wallets, change.walletID)
		}

		c.balances[*change.walletID] += change.delta
	}

	return nil
}

func (c *walletBalanceChanges) balanceOf(walletID *uuid.UUID) float64 {
	return c.balances[*walletID]
}

// flush writes each wallet's final balance once, in the order the wallets were first seen.
func (c *walletBalanceChanges) flush(ctx context.Context, tx *gorm.DB) error {
	for _, walletID := range c.wallets {
		if err := c.repo.UpdateAmount(ctx, tx, walletID, c.balances[*walletID]); err != nil {
			return fmt.Errorf("error updating wallet balance: %w", err)
		}
	}

	return nil
}
