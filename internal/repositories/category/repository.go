package category

import (
	"context"
	"personal-finance/internal/business/model"
)

type Repository interface {
	Add(ctx context.Context, car model.Category) error
	FindAll(ctx context.Context) ([]model.Category, error)
	FindByID(ctx context.Context, ID string) (model.Category, error)
	Update(ctx context.Context, ID string, car model.Category) (model.Category, error)
	Delete(ctx context.Context, ID string) error
}
