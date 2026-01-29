package usecase

import (
	"context"
	"time"

	"personal-finance/internal/plataform/authentication"
)

type LimitsCountRepository interface {
	CountByUserID(ctx context.Context) (int64, error)
}

type MovementCountRepository interface {
	CountByUserIDAndMonth(ctx context.Context, year int, month time.Month) (int64, error)
}

type RecurrentCountRepository interface {
	CountActiveByUserIDAndMonth(ctx context.Context, year int, month time.Month) (int64, error)
}

type LimitsUsage struct {
	Wallets             int64 `json:"wallets"`
	CreditCards         int64 `json:"credit_cards"`
	MovementsPerMonth   int64 `json:"movements_per_month"`
	RecurrencesPerMonth int64 `json:"recurrences_per_month"`
}

type LimitsResponse struct {
	Plan    string                    `json:"plan"`
	Limits  authentication.PlanLimits `json:"limits"`
	Usage   LimitsUsage               `json:"usage"`
	ResetAt time.Time                 `json:"reset_at"`
}

type Limits struct {
	walletRepo     LimitsCountRepository
	creditCardRepo LimitsCountRepository
	movementRepo   MovementCountRepository
	recurrentRepo  RecurrentCountRepository
}

func NewLimits(
	walletRepo LimitsCountRepository,
	creditCardRepo LimitsCountRepository,
	movementRepo MovementCountRepository,
	recurrentRepo RecurrentCountRepository,
) *Limits {
	return &Limits{
		walletRepo:     walletRepo,
		creditCardRepo: creditCardRepo,
		movementRepo:   movementRepo,
		recurrentRepo:  recurrentRepo,
	}
}

func (l *Limits) GetLimits(ctx context.Context) (LimitsResponse, error) {
	auth, ok := authentication.AuthFromContext(ctx)
	if !ok {
		return LimitsResponse{}, ErrUnauthorized
	}

	now := time.Now()
	year, month := now.Year(), now.Month()

	walletsCount, err := l.walletRepo.CountByUserID(ctx)
	if err != nil {
		return LimitsResponse{}, err
	}

	creditCardsCount, err := l.creditCardRepo.CountByUserID(ctx)
	if err != nil {
		return LimitsResponse{}, err
	}

	movementsCount, err := l.movementRepo.CountByUserIDAndMonth(ctx, year, month)
	if err != nil {
		return LimitsResponse{}, err
	}

	recurrencesCount, err := l.recurrentRepo.CountActiveByUserIDAndMonth(ctx, year, month)
	if err != nil {
		return LimitsResponse{}, err
	}

	limits := authentication.PlanLimits{}
	if auth.IsFree() {
		limits = authentication.GetFreePlanLimits()
	}

	firstDayNextMonth := time.Date(year, month+1, 1, 0, 0, 0, 0, time.UTC)

	return LimitsResponse{
		Plan:   string(auth.Plan),
		Limits: limits,
		Usage: LimitsUsage{
			Wallets:             walletsCount,
			CreditCards:         creditCardsCount,
			MovementsPerMonth:   movementsCount,
			RecurrencesPerMonth: recurrencesCount,
		},
		ResetAt: firstDayNextMonth,
	}, nil
}
