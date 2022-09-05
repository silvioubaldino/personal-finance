package transaction

import (
	"context"
	"personal-finance/internal/business/model"
)

type Repository interface {
	Add(ctx context.Context, car model.Transaction) error
	FindAll(ctx context.Context) ([]model.Transaction, error)
	FindByID(ctx context.Context, ID string) (model.Transaction, error)
	Update(ctx context.Context, ID string, car model.Transaction) (model.Transaction, error)
	Delete(ctx context.Context, ID string) error
}
