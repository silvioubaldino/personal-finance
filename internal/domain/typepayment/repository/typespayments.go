package repository

import (
	"context"
	"personal-finance/internal/plataform/authentication"
	"time"

	"personal-finance/internal/model"

	"gorm.io/gorm"
)

type Repository interface {
	Add(ctx context.Context, typePayment model.TypePayment) (model.TypePayment, error)
	FindAll(ctx context.Context) ([]model.TypePayment, error)
	FindByID(ctx context.Context, id int) (model.TypePayment, error)
	Update(ctx context.Context, id int, typePayment model.TypePayment) (model.TypePayment, error)
	Delete(ctx context.Context, id int) error
}

type PgRepository struct {
	Gorm *gorm.DB
}

func NewPgRepository(gorm *gorm.DB) Repository {
	return PgRepository{Gorm: gorm}
}

func (p PgRepository) Add(ctx context.Context, typePayment model.TypePayment) (model.TypePayment, error) {
	userID := ctx.Value(authentication.UserID).(string)
	now := time.Now()
	typePayment.DateCreate = now
	typePayment.DateUpdate = now
	typePayment.UserID = userID
	result := p.Gorm.Create(&typePayment)
	if err := result.Error; err != nil {
		return model.TypePayment{}, err
	}
	return typePayment, nil
}

func (p PgRepository) FindAll(ctx context.Context) ([]model.TypePayment, error) {
	userID := ctx.Value(authentication.UserID).(string)
	var typePayments []model.TypePayment
	result := p.Gorm.Where("user_id=?", userID).Find(&typePayments)
	if err := result.Error; err != nil {
		return []model.TypePayment{}, err
	}
	return typePayments, nil
}

func (p PgRepository) FindByID(ctx context.Context, id int) (model.TypePayment, error) {
	userID := ctx.Value(authentication.UserID).(string)
	var typePayment model.TypePayment
	result := p.Gorm.Where("user_id=?", userID).First(&typePayment, id)
	if err := result.Error; err != nil {
		return model.TypePayment{}, err
	}
	return typePayment, nil
}

func (p PgRepository) Update(ctx context.Context, id int, typePayment model.TypePayment) (model.TypePayment, error) {
	w, err := p.FindByID(ctx, id)
	if err != nil {
		return model.TypePayment{}, err
	}
	w.Description = typePayment.Description
	w.DateUpdate = time.Now()
	result := p.Gorm.Save(&w)
	if result.Error != nil {
		return model.TypePayment{}, result.Error
	}
	return w, nil
}

func (p PgRepository) Delete(_ context.Context, id int) error {
	if err := p.Gorm.Delete(&model.TypePayment{}, id).Error; err != nil {
		return err
	}
	return nil
}
