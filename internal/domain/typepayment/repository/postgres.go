package repository

import (
	"context"
	"personal-finance/internal/model"
	"time"

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

func (p PgRepository) Add(_ context.Context, typePayment model.TypePayment) (model.TypePayment, error) {
	now := time.Now()
	typePayment.DateCreate = now
	typePayment.DateUpdate = now
	result := p.Gorm.Create(&typePayment)
	if err := result.Error; err != nil {
		return model.TypePayment{}, err
	}
	return typePayment, nil
}

func (p PgRepository) FindAll(_ context.Context) ([]model.TypePayment, error) {
	var typePayments []model.TypePayment
	result := p.Gorm.Find(&typePayments)
	if err := result.Error; err != nil {
		return []model.TypePayment{}, err
	}
	return typePayments, nil
}

func (p PgRepository) FindByID(_ context.Context, id int) (model.TypePayment, error) {
	var typePayment model.TypePayment
	result := p.Gorm.First(&typePayment, id)
	if err := result.Error; err != nil {
		return model.TypePayment{}, err
	}
	return typePayment, nil
}

func (p PgRepository) Update(_ context.Context, id int, typePayment model.TypePayment) (model.TypePayment, error) {
	w, err := p.FindByID(context.Background(), id)
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
