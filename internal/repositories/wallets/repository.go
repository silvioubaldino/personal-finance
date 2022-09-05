package wallets

import (
	"context"
	"personal-finance/internal/business/model"
)

type Repository interface {
	Add(ctx context.Context, car model.Wallet) error
	FindAll(ctx context.Context) ([]model.Wallet, error)
	FindByID(ctx context.Context, ID string) (model.Wallet, error)
	Update(ctx context.Context, ID string, car model.Wallet) (model.Wallet, error)
	Delete(ctx context.Context, ID string) error
}
