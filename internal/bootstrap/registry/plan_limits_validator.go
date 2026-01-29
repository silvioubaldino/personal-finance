package registry

import (
	"context"
	"time"

	"personal-finance/internal/infrastructure/repository"
	"personal-finance/internal/plataform/authentication"
	"personal-finance/internal/usecase"
)

type PlanLimitsValidator struct {
	walletRepo     *repository.WalletRepository
	creditCardRepo *repository.CreditCardRepository
	movementRepo   *repository.MovementRepository
	recurrentRepo  *repository.RecurrentMovementRepository
}

func NewPlanLimitsValidator(
	walletRepo *repository.WalletRepository,
	creditCardRepo *repository.CreditCardRepository,
	movementRepo *repository.MovementRepository,
	recurrentRepo *repository.RecurrentMovementRepository,
) *PlanLimitsValidator {
	return &PlanLimitsValidator{
		walletRepo:     walletRepo,
		creditCardRepo: creditCardRepo,
		movementRepo:   movementRepo,
		recurrentRepo:  recurrentRepo,
	}
}

func (v *PlanLimitsValidator) ValidateWalletCreation(ctx context.Context) error {
	auth, ok := authentication.AuthFromContext(ctx)
	if !ok {
		return usecase.ErrUnauthorized
	}

	if !auth.IsFree() {
		return nil
	}

	limits := authentication.GetFreePlanLimits()
	count, err := v.walletRepo.CountByUserID(ctx)
	if err != nil {
		return err
	}

	if count >= int64(limits.Wallets) {
		return usecase.ErrWalletLimitReached
	}

	return nil
}

func (v *PlanLimitsValidator) ValidateCreditCardCreation(ctx context.Context) error {
	auth, ok := authentication.AuthFromContext(ctx)
	if !ok {
		return usecase.ErrUnauthorized
	}

	if !auth.IsFree() {
		return nil
	}

	limits := authentication.GetFreePlanLimits()
	count, err := v.creditCardRepo.CountByUserID(ctx)
	if err != nil {
		return err
	}

	if count >= int64(limits.CreditCards) {
		return usecase.ErrCreditCardLimitReached
	}

	return nil
}

func (v *PlanLimitsValidator) ValidateMovementCreation(ctx context.Context) error {
	auth, ok := authentication.AuthFromContext(ctx)
	if !ok {
		return usecase.ErrUnauthorized
	}

	if !auth.IsFree() {
		return nil
	}

	limits := authentication.GetFreePlanLimits()
	now := time.Now()
	count, err := v.movementRepo.CountByUserIDAndMonth(ctx, now.Year(), now.Month())
	if err != nil {
		return err
	}

	if count >= int64(limits.MovementsPerMonth) {
		return usecase.ErrMovementLimitReached
	}

	return nil
}

func (v *PlanLimitsValidator) ValidateRecurrenceCreation(ctx context.Context) error {
	auth, ok := authentication.AuthFromContext(ctx)
	if !ok {
		return usecase.ErrUnauthorized
	}

	if !auth.IsFree() {
		return nil
	}

	limits := authentication.GetFreePlanLimits()
	now := time.Now()
	count, err := v.recurrentRepo.CountActiveByUserIDAndMonth(ctx, now.Year(), now.Month())
	if err != nil {
		return err
	}

	if count >= int64(limits.RecurrencesPerMonth) {
		return usecase.ErrRecurrenceLimitReached
	}

	return nil
}
