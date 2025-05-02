package repository

import (
	"context"

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

	dbRecurrentMovement := FromRecurrentMovementDomain(recurrentMovement)

	if err := tx.Create(&dbRecurrentMovement).Error; err != nil {
		if r.isDuplicateError(err) {
			return domain.RecurrentMovement{}, domain.WrapConflict(err, "recurrent movement already exists")
		}
		return domain.RecurrentMovement{}, domain.WrapInternalError(err, "error creating recurrent movement")
	}

	if isLocalTx {
		if err := tx.Commit().Error; err != nil {
			return domain.RecurrentMovement{}, domain.WrapInternalError(err, "error committing transaction")
		}
	}

	return dbRecurrentMovement.ToDomain(), nil
}

// isDuplicateError verifica se o erro é de duplicação de registro
func (r *RecurrentMovementRepository) isDuplicateError(err error) bool {
	// Implementação deve ser adaptada para o tipo de banco de dados utilizado
	return err != nil && (err.Error() == "duplicate key value violates unique constraint" ||
		err.Error() == "UNIQUE constraint failed")
}
