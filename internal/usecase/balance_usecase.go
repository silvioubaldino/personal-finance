package usecase

import (
	"context"
	"fmt"

	"personal-finance/internal/domain"
)

type Balance interface {
	CalculateBalance(ctx context.Context, period domain.Period) (domain.Balance, error)
}

type balanceUseCase struct {
	movementUC any // Movement
}

func NewBalance(movementUC any) Balance {
	return balanceUseCase{
		movementUC: movementUC,
	}
}

func (uc balanceUseCase) CalculateBalance(ctx context.Context, period domain.Period) (domain.Balance, error) {
	if err := period.Validate(); err != nil {
		return domain.Balance{}, fmt.Errorf("período inválido: %w", err)
	}

	// return uc.movementUC.CalculateBalance(ctx, period)
	return domain.Balance{}, nil
}
