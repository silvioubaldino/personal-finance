package repository

import (
	"context"

	"personal-finance/internal/model"

	"gorm.io/gorm"
)

type Repository interface {
	FindAll(ctx context.Context) ([]model.TransactionStatus, error)
}

type PgRepository struct {
	Gorm *gorm.DB
}

func NewPgRepository(gorm *gorm.DB) Repository {
	return PgRepository{Gorm: gorm}
}

func (p PgRepository) FindAll(_ context.Context) ([]model.TransactionStatus, error) {
	var transactionStatus []model.TransactionStatus
	result := p.Gorm.Find(&transactionStatus)
	if err := result.Error; err != nil {
		return []model.TransactionStatus{}, err
	}
	return transactionStatus, nil
}
