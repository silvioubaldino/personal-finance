package service

import (
	"personal-finance/internal/domain/movement/repository"
	"personal-finance/internal/model"
)

type Balance interface {
	FindByPeriod(period model.Period, userID string) (model.Balance, error)
}

type balance struct {
	movementRepo repository.Repository
}

func NewBalanceService(movementRepo repository.Repository) Balance {
	return balance{movementRepo}
}

func (s balance) FindByPeriod(period model.Period, userID string) (model.Balance, error) {
	expense, err := s.movementRepo.ExpensesByPeriod(period, userID)
	if err != nil {
		return model.Balance{}, err
	}
	income, err := s.movementRepo.IncomesByPeriod(period, userID)
	if err != nil {
		return model.Balance{}, err
	}
	balance := model.Balance{
		Expense: expense,
		Income:  income,
	}
	balance.Consolidate()

	return balance, nil
}
