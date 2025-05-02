package repository

import (
	"context"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/plataform/authentication"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MovementRepository struct {
	db *gorm.DB
}

func NewMovementRepository(db *gorm.DB) *MovementRepository {
	return &MovementRepository{
		db: db,
	}
}

func (r *MovementRepository) Add(ctx context.Context, tx *gorm.DB, movement domain.Movement) (domain.Movement, error) {
	var isLocalTx bool
	if tx == nil {
		isLocalTx = true
		tx = r.db.Begin()
		defer tx.Rollback()
	}

	userID := ctx.Value(authentication.UserID).(string)
	now := time.Now()
	id := uuid.New()

	movement.ID = &id
	movement.DateCreate = now
	movement.DateUpdate = now
	movement.UserID = userID

	dbMovement := FromMovementDomain(movement)

	if err := tx.Create(&dbMovement).Error; err != nil {
		if r.isDuplicateError(err) {
			return domain.Movement{}, domain.WrapConflict(err, "movement already exists")
		}
		return domain.Movement{}, domain.WrapInternalError(err, "error creating movement")
	}

	if isLocalTx {
		if err := tx.Commit().Error; err != nil {
			return domain.Movement{}, domain.WrapInternalError(err, "error committing transaction")
		}
	}

	return dbMovement.ToDomain(), nil
}

func (r *MovementRepository) isDuplicateError(err error) bool { // Improve
	return err != nil && (err.Error() == "duplicate key value violates unique constraint" ||
		err.Error() == "UNIQUE constraint failed")
}
