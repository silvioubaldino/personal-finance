package service

import (
	"context"

	"github.com/google/uuid"

	estimateRepository "personal-finance/internal/domain/estimate/repository"
	"personal-finance/internal/domain/movement/repository"
	"personal-finance/internal/model"
)

type Balance interface {
	FindByPeriod(ctx context.Context, period model.Period, userID string) (model.Balance, error)
}

type balance struct {
	movementRepo repository.Repository
	estimateRepo estimateRepository.Repository
}

func NewBalanceService(movementRepo repository.Repository, estimateRepo estimateRepository.Repository) Balance {
	return balance{
		movementRepo: movementRepo,
		estimateRepo: estimateRepo,
	}
}

func (s balance) FindByPeriod(ctx context.Context, period model.Period, userID string) (model.Balance, error) {
	movements, err := s.movementRepo.FindByPeriod(ctx, period, userID)
	if err != nil {
		return model.Balance{}, err
	}

	estimates, err := s.estimateRepo.FindCategoriesByMonth(
		ctx,
		int(period.From.Month()),
		period.From.Year(),
		userID,
	)
	if err != nil {
		return model.Balance{}, err
	}

	sumByCategoryMap := movements.GetExpenseMovements().GetPaidMovements().GetSumByCategory()
	estimatesByCategoryMap := estimates.GetExpenseEstimates().GetEstimateByCategory()
	totalExpense := getExpenseSum(estimatesByCategoryMap, sumByCategoryMap)

	incomeSumByCategoryMap := movements.GetIncomeMovements().GetPaidMovements().GetSumByCategory()
	incomeEstimatesByCategoryMap := estimates.GetIncomeEstimates().GetEstimateByCategory()
	totalIncome := getIncomeSum(incomeEstimatesByCategoryMap, incomeSumByCategoryMap)

	return model.Balance{
		Expense:       totalExpense,
		Income:        totalIncome,
		PeriodBalance: totalIncome + totalExpense,
	}, nil
}

func getExpenseSum(estimatesByCategoryMap, sumByCategoryMap map[*uuid.UUID]float64) float64 {
	expensesResultMap := make(map[uuid.UUID]float64)
	for id, estimate := range estimatesByCategoryMap {
		expensesResultMap[*id] = estimate
	}

	for id, sumByCategory := range sumByCategoryMap {
		if estimate, ok := expensesResultMap[*id]; ok {
			if sumByCategory < estimate {
				expensesResultMap[*id] = sumByCategory
			}
			continue
		}
		expensesResultMap[*id] = sumByCategory
	}

	var totalExpense float64
	for _, amount := range expensesResultMap {
		totalExpense += amount
	}
	return totalExpense
}

func getIncomeSum(estimatesByCategoryMap, sumByCategoryMap map[*uuid.UUID]float64) float64 {
	expensesResultMap := make(map[uuid.UUID]float64)
	for id, estimate := range estimatesByCategoryMap {
		expensesResultMap[*id] = estimate
	}

	for id, sumByCategory := range sumByCategoryMap {
		if estimate, ok := expensesResultMap[*id]; ok {
			if sumByCategory > estimate {
				expensesResultMap[*id] = sumByCategory
			}
			continue
		}
		expensesResultMap[*id] = sumByCategory
	}

	var totalExpense float64
	for _, amount := range expensesResultMap {
		totalExpense += amount
	}
	return totalExpense
}
