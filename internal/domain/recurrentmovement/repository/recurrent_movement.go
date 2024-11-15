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
	Update(ctx context.Context, id *uuid.UUID, newRecurrent model.RecurrentMovement) (model.RecurrentMovement, error)
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
		Where("recurrent_movements.end_date >= ? OR recurrent_movements.end_date IS NULL", date).
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
			"recurrent_movements.user_id",
			"recurrent_movements.type_payment_id",
			"recurrent_movements.wallet_id",
			"recurrent_movements.category_id",
			"recurrent_movements.sub_category_id",
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

func (r recurrentRepository) Update(
	ctx context.Context,
	id *uuid.UUID,
	newRecurrent model.RecurrentMovement,
) (model.RecurrentMovement, error) {
	recurrentFound, err := r.FindByID(ctx, *id)
	if err != nil {
		return model.RecurrentMovement{}, err
	}
	recurrentFound = update(newRecurrent, recurrentFound)

	err = r.gorm.
		Select([]string{
			"id",
			"description",
			"amount",
			"initial_date",
			"end_date",
			"user_id",
			"type_payment_id",
			"wallet_id",
			"category_id",
			"sub_category_id",
		}).
		Save(&recurrentFound).Error
	if err != nil {
		return model.RecurrentMovement{}, err
	}
	return recurrentFound, nil
}

func update(newRecurrent model.RecurrentMovement, recurrentFound model.RecurrentMovement) model.RecurrentMovement {
	if newRecurrent.Description != "" && newRecurrent.Description != recurrentFound.Description {
		recurrentFound.Description = newRecurrent.Description
	}
	if newRecurrent.Amount != 0 && newRecurrent.Amount != recurrentFound.Amount {
		recurrentFound.Amount = newRecurrent.Amount
	}
	if newRecurrent.InitialDate != nil && *newRecurrent.InitialDate != *recurrentFound.InitialDate {
		recurrentFound.InitialDate = newRecurrent.InitialDate
	}
	if newRecurrent.EndDate != nil && *newRecurrent.EndDate != *recurrentFound.EndDate {
		recurrentFound.EndDate = newRecurrent.EndDate
	}
	if newRecurrent.WalletID != nil && *newRecurrent.WalletID != *recurrentFound.WalletID {
		recurrentFound.WalletID = newRecurrent.WalletID
	}
	if newRecurrent.TypePaymentID != 0 && newRecurrent.TypePaymentID != recurrentFound.TypePaymentID {
		recurrentFound.TypePaymentID = newRecurrent.TypePaymentID
	}
	if newRecurrent.CategoryID != nil && *newRecurrent.CategoryID != *recurrentFound.CategoryID {
		recurrentFound.CategoryID = newRecurrent.CategoryID
	}
	if newRecurrent.SubCategoryID != nil && *newRecurrent.SubCategoryID != *recurrentFound.SubCategoryID {
		recurrentFound.SubCategoryID = newRecurrent.SubCategoryID
	}

	return recurrentFound
}
