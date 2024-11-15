package repository

import (
	"context"
	"gorm.io/gorm"

	"github.com/google/uuid"

	"personal-finance/internal/model"
)

type RecurrentRepository interface {
	AddConsistent(ctx context.Context, tx *gorm.DB, recurrent model.RecurrentMovement) (model.RecurrentMovement, error)
}

type recurrentRepository struct {
	gorm *gorm.DB
}

func NewRecurrentRepository(gorm *gorm.DB) RecurrentRepository {
	return &recurrentRepository{gorm: gorm}
}

func (r *recurrentRepository) AddConsistent(ctx context.Context, tx *gorm.DB, recurrent model.RecurrentMovement) (model.RecurrentMovement, error) {
	id := uuid.New()
	userID := ctx.Value("user_id").(string)
	recurrent.ID = &id
	recurrent.UserID = userID

	err := tx.Create(&recurrent).Error
	if err != nil {
		return model.RecurrentMovement{}, err
	}
	return recurrent, nil
}
