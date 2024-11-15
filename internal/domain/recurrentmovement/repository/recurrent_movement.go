package repository

import (
	"context"
	"gorm.io/gorm"
	"time"

	"github.com/google/uuid"

	"personal-finance/internal/model"
)

type RecurrentRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (model.RecurrentMovement, error)
	AddConsistent(ctx context.Context, tx *gorm.DB, recurrent model.RecurrentMovement) (model.RecurrentMovement, error)
	FindByMonth(ctx context.Context, initialDate time.Time) ([]model.RecurrentMovement, error)
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

func (r *recurrentRepository) FindByMonth(ctx context.Context, date time.Time) ([]model.RecurrentMovement, error) {
	userID := ctx.Value("user_id").(string)
	var recurrents []model.RecurrentMovement

	err := r.gorm.
		Select([]string{
			"recurrent_movements.id",
			"recurrent_movements.description",
			"recurrent_movements.initial_date",
			"recurrent_movements.amount",
			`w.id as "Wallet__id"`,
			`w.description as "Wallet__description"`,
			`c.id as "Category__id"`,
			`c.description as "Category__description"`,
			`sc.id as "SubCategory__id"`,
			`sc.description as "SubCategory__description"`,
		}).
		Order("recurrent_movements.initial_date desc").
		Joins("left join wallets w on recurrent_movements.wallet_id = w.id").
		Joins("left join categories c on recurrent_movements.category_id = c.id").
		Joins("left join sub_categories sc on recurrent_movements.sub_category_id = sc.id").
		Where("recurrent_movements.user_id = ?", userID).
		Where("recurrent_movements.initial_date <= ?", date).
		Where("recurrent_movements.end_date >= ?", date).
		Or("recurrent_movements.end_date is null").
		Find(&recurrents).Error
	if err != nil {
		return nil, err
	}
	return recurrents, nil
}

func (r *recurrentRepository) FindByID(ctx context.Context, id uuid.UUID) (model.RecurrentMovement, error) {
	userID := ctx.Value("user_id").(string)
	var recurrent model.RecurrentMovement

	err := r.gorm.
		Select([]string{
			"recurrent_movements.id",
			"recurrent_movements.description",
			"recurrent_movements.initial_date",
			"recurrent_movements.end_date",
			"recurrent_movements.amount",
			"recurrent_movements.type_payment_id",
			`w.id as "Wallet__id"`,
			`w.description as "Wallet__description"`,
			`c.id as "Category__id"`,
			`c.description as "Category__description"`,
			`sc.id as "SubCategory__id"`,
			`sc.description as "SubCategory__description"`,
		}).
		Joins("left join wallets w on recurrent_movements.wallet_id = w.id").
		Joins("left join categories c on recurrent_movements.category_id = c.id").
		Joins("left join sub_categories sc on recurrent_movements.sub_category_id = sc.id").
		Where("recurrent_movements.user_id = ?", userID).
		Where("recurrent_movements.id = ?", id).
		First(&recurrent).Error
	if err != nil {
		return model.RecurrentMovement{}, err
	}
	return recurrent, nil
}
