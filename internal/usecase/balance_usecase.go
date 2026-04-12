package usecase

import (
	"context"
	"fmt"

	"personal-finance/internal/domain"

	"github.com/google/uuid"
)

type BalanceMovementRepository interface {
	FindByPeriod(ctx context.Context, period domain.Period) (domain.MovementList, error)
}

type BalanceEstimateRepository interface {
	FindCategoriesByMonth(ctx context.Context, month int, year int) ([]domain.EstimateCategories, error)
}

type Balance interface {
	CalculateBalance(ctx context.Context, period domain.Period) (domain.Balance, error)
}

type balanceUseCase struct {
	movementRepo BalanceMovementRepository
	estimateRepo BalanceEstimateRepository
}

func NewBalance(movementRepo BalanceMovementRepository, estimateRepo BalanceEstimateRepository) Balance {
	return balanceUseCase{
		movementRepo: movementRepo,
		estimateRepo: estimateRepo,
	}
}

func (uc balanceUseCase) CalculateBalance(ctx context.Context, period domain.Period) (domain.Balance, error) {
	if err := period.Validate(); err != nil {
		return domain.Balance{}, fmt.Errorf("período inválido: %w", err)
	}

	movements, err := uc.movementRepo.FindByPeriod(ctx, period)
	if err != nil {
		return domain.Balance{}, fmt.Errorf("error finding movements: %w", err)
	}

	estimates, err := uc.estimateRepo.FindCategoriesByMonth(ctx, int(period.From.Month()), period.From.Year())
	if err != nil {
		return domain.Balance{}, fmt.Errorf("error finding estimates: %w", err)
	}

	estimateList := domain.EstimateCategoriesList(estimates)
	sumByCategoryMap := movements.GetExpenseMovements().GetPaidMovements().GetSumByCategory()
	estimatesByCategoryMap := estimateList.GetExpenseEstimates().GetEstimateByCategory()
	totalExpense := getBalanceSum(estimatesByCategoryMap, sumByCategoryMap, false)

	incomeSumByCategoryMap := movements.GetIncomeMovements().GetPaidMovements().GetSumByCategory()
	incomeEstimatesByCategoryMap := estimateList.GetIncomeEstimates().GetEstimateByCategory()
	totalIncome := getBalanceSum(incomeEstimatesByCategoryMap, incomeSumByCategoryMap, true)

	balance := domain.Balance{
		Expense: totalExpense,
		Income:  totalIncome,
	}
	balance.Consolidate()

	return balance, nil
}

// getBalanceSum applies estimate as ceiling for expenses (take min) and floor for income (take max).
func getBalanceSum(estimatesByCategoryMap, sumByCategoryMap map[*uuid.UUID]float64, isIncome bool) float64 {
	resultMap := make(map[uuid.UUID]float64)
	for id, estimate := range estimatesByCategoryMap {
		resultMap[*id] = estimate
	}

	for id, actual := range sumByCategoryMap {
		if estimate, ok := resultMap[*id]; ok {
			if isIncome {
				if actual > estimate {
					resultMap[*id] = actual
				}
			} else {
				if actual < estimate {
					resultMap[*id] = actual
				}
			}
			continue
		}
		resultMap[*id] = actual
	}

	var total float64
	for _, amount := range resultMap {
		total += amount
	}
	return total
}
