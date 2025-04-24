package repository

import (
	"context"
	"fmt"

	"personal-finance/internal/domain"
	"personal-finance/internal/plataform/authentication"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RecurrentMovementRepository struct {
	db *gorm.DB
}

func NewRecurrentMovementRepository(db *gorm.DB) *RecurrentMovementRepository {
	return &RecurrentMovementRepository{
		db: db,
	}
}

func (r *RecurrentMovementRepository) Add(ctx context.Context, tx *gorm.DB, recurrentMovement domain.RecurrentMovement) (domain.RecurrentMovement, error) {
	var isLocalTx bool
	if tx == nil {
		isLocalTx = true
		tx = r.db.Begin()
		defer tx.Rollback()
	}

	userID := ctx.Value(authentication.UserID).(string)
	id := uuid.New()

	recurrentMovement.ID = &id
	recurrentMovement.UserID = userID

	dbRecurrentMovement := ToRecurrentMovementModel(recurrentMovement)

	if err := tx.Create(&dbRecurrentMovement).Error; err != nil {
		return domain.RecurrentMovement{}, fmt.Errorf("error creating recurrent movement: %w", err)
	}

	if isLocalTx {
		tx.Commit()
	}

	return dbRecurrentMovement.ToDomain(), nil
}
