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
	MovementID          uuid.UUID `json:"movement_id"`
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

func (u *UpdateTransfer) Execute(ctx context.Context, input UpdateTransferInput) (TransferOutput, error) {
	if err := u.validateInput(input); err != nil {
		return TransferOutput{}, err
	}

	movement, err := u.movementRepo.FindByID(ctx, input.MovementID)
	if err != nil {
		return TransferOutput{}, fmt.Errorf("error finding movement: %w", err)
	}

	if movement.TypePayment != domain.TypePaymentInternalTransfer {
		return TransferOutput{}, ErrMovementNotInternalTransfer
	}

	if movement.PairID == nil || *movement.PairID != input.PairID {
		return TransferOutput{}, ErrTransferPairMismatch
	}

	pairMovements, err := u.movementRepo.FindByPairID(ctx, input.PairID)
	if err != nil {
		return TransferOutput{}, fmt.Errorf("error finding pair movements: %w", err)
	}

	if len(pairMovements) != 2 {
		return TransferOutput{}, ErrTransferPairNotFound
	}

	originMovement, destinationMovement := u.identifyMovements(pairMovements)
	if originMovement.ID == nil || destinationMovement.ID == nil {
		return TransferOutput{}, ErrTransferPairNotFound
	}

	originWallet, err := u.walletRepo.FindByID(ctx, &input.OriginWalletID)
	if err != nil {
		return TransferOutput{}, fmt.Errorf("error finding origin wallet: %w", err)
	}

	destinationWallet, err := u.walletRepo.FindByID(ctx, &input.DestinationWalletID)
	if err != nil {
		return TransferOutput{}, fmt.Errorf("error finding destination wallet: %w", err)
	}

	onlyDateChanged := u.isOnlyDateChanged(input, originMovement, destinationMovement, originWallet, destinationWallet)

	var result TransferOutput

	err = u.txManager.WithTransaction(ctx, func(tx *gorm.DB) error {
		if onlyDateChanged {
			return u.updateDateOnly(ctx, tx, input, movement, originMovement, destinationMovement, &result)
		}
		return u.updateBothMovements(ctx, tx, input, originMovement, destinationMovement, originWallet, destinationWallet, &result)
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

func (u *UpdateTransfer) identifyMovements(movements domain.MovementList) (origin, destination domain.Movement) {
	for _, m := range movements {
		if m.Amount < 0 {
			origin = m
		} else {
			destination = m
		}
	}
	return origin, destination
}

func (u *UpdateTransfer) isOnlyDateChanged(
	input UpdateTransferInput,
	origin, destination domain.Movement,
	originWallet, destWallet domain.Wallet,
) bool {
	currentAmount := destination.Amount

	sameAmount := input.Amount == currentAmount
	sameOriginWallet := *origin.WalletID == input.OriginWalletID
	sameDestWallet := *destination.WalletID == input.DestinationWalletID

	currentDesc := u.extractDescription(origin.Description)
	sameDescription := input.Description == currentDesc

	dateChanged := !input.Date.Equal(*origin.Date) || !input.Date.Equal(*destination.Date)

	return sameAmount && sameOriginWallet && sameDestWallet && sameDescription && dateChanged
}

func (u *UpdateTransfer) extractDescription(fullDesc string) string {
	return fullDesc
}

func (u *UpdateTransfer) updateDateOnly(
	ctx context.Context,
	tx *gorm.DB,
	input UpdateTransferInput,
	targetMovement, origin, destination domain.Movement,
	result *TransferOutput,
) error {
	var updatedOrigin, updatedDestination domain.Movement
	var err error

	if *targetMovement.ID == *origin.ID {
		origin.Date = &input.Date
		updatedOrigin, err = u.movementRepo.Update(ctx, tx, *origin.ID, origin)
		if err != nil {
			return fmt.Errorf("error updating origin movement: %w", err)
		}
		updatedDestination = destination
	} else {
		destination.Date = &input.Date
		updatedDestination, err = u.movementRepo.Update(ctx, tx, *destination.ID, destination)
		if err != nil {
			return fmt.Errorf("error updating destination movement: %w", err)
		}
		updatedOrigin = origin
	}

	result.PairID = input.PairID
	result.OriginMovement = updatedOrigin
	result.DestinationMovement = updatedDestination

	return nil
}

func (u *UpdateTransfer) updateBothMovements(
	ctx context.Context,
	tx *gorm.DB,
	input UpdateTransferInput,
	oldOrigin, oldDestination domain.Movement,
	originWallet, destWallet domain.Wallet,
	result *TransferOutput,
) error {
	if oldOrigin.IsPaid {
		if err := u.revertOldBalances(ctx, tx, oldOrigin, oldDestination); err != nil {
			return err
		}
	}

	outCategoryID := uuid.MustParse(domain.InternalTransferOutCategoryID)
	inCategoryID := uuid.MustParse(domain.InternalTransferInCategoryID)
	newDescription := u.buildDescription(input.Description, originWallet.Description, destWallet.Description)

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

	if oldOrigin.IsPaid {
		originW, err := u.walletRepo.FindByID(ctx, &input.OriginWalletID)
		if err != nil {
			return fmt.Errorf("error finding new origin wallet: %w", err)
		}

		if !originW.HasSufficientBalance(-input.Amount) {
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
		if err := u.applyNewBalances(ctx, tx, &input.OriginWalletID, &input.DestinationWalletID, input.Amount); err != nil {
			return err
		}
	}

	result.PairID = input.PairID
	result.OriginMovement = updatedOrigin
	result.DestinationMovement = updatedDestination

	return nil
}

func (u *UpdateTransfer) revertOldBalances(
	ctx context.Context,
	tx *gorm.DB,
	origin, destination domain.Movement,
) error {
	originWallet, err := u.walletRepo.FindByID(ctx, origin.WalletID)
	if err != nil {
		return fmt.Errorf("error finding origin wallet: %w", err)
	}
	newOriginBalance := originWallet.Balance - origin.Amount
	if err := u.walletRepo.UpdateAmount(ctx, tx, origin.WalletID, newOriginBalance); err != nil {
		return fmt.Errorf("error reverting origin wallet balance: %w", err)
	}

	destWallet, err := u.walletRepo.FindByID(ctx, destination.WalletID)
	if err != nil {
		return fmt.Errorf("error finding destination wallet: %w", err)
	}
	newDestBalance := destWallet.Balance - destination.Amount
	if err := u.walletRepo.UpdateAmount(ctx, tx, destination.WalletID, newDestBalance); err != nil {
		return fmt.Errorf("error reverting destination wallet balance: %w", err)
	}

	return nil
}

func (u *UpdateTransfer) applyNewBalances(
	ctx context.Context,
	tx *gorm.DB,
	originWalletID, destWalletID *uuid.UUID,
	amount float64,
) error {
	originWallet, err := u.walletRepo.FindByID(ctx, originWalletID)
	if err != nil {
		return fmt.Errorf("error finding origin wallet: %w", err)
	}
	newOriginBalance := originWallet.Balance - amount
	if err := u.walletRepo.UpdateAmount(ctx, tx, originWalletID, newOriginBalance); err != nil {
		return fmt.Errorf("error updating origin wallet balance: %w", err)
	}

	destWallet, err := u.walletRepo.FindByID(ctx, destWalletID)
	if err != nil {
		return fmt.Errorf("error finding destination wallet: %w", err)
	}
	newDestBalance := destWallet.Balance + amount
	if err := u.walletRepo.UpdateAmount(ctx, tx, destWalletID, newDestBalance); err != nil {
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
