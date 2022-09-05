package typepayment

import (
	"context"
	"personal-finance/internal/business/model"
)

type Repository interface {
	Add(ctx context.Context, car model.TypePayment) error
	FindAll(ctx context.Context) ([]model.TypePayment, error)
	FindByID(ctx context.Context, ID string) (model.TypePayment, error)
	Update(ctx context.Context, ID string, car model.TypePayment) (model.TypePayment, error)
	Delete(ctx context.Context, ID string) error
}
