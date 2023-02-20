package repository

import (
	"context"
	"time"

	"personal-finance/internal/model"

	"gorm.io/gorm"
)

type Repository interface {
	Add(ctx context.Context, typePayment model.TypePayment, userID string) (model.TypePayment, error)
	FindAll(ctx context.Context, userID string) ([]model.TypePayment, error)
	FindByID(ctx context.Context, id int, userID string) (model.TypePayment, error)
	Update(ctx context.Context, id int, typePayment model.TypePayment, userID string) (model.TypePayment, error)
	Delete(ctx context.Context, id int) error
}

type PgRepository struct {
	Gorm *gorm.DB
}

func NewPgRepository(gorm *gorm.DB) Repository {
	return PgRepository{Gorm: gorm}
}

func (p PgRepository) Add(_ context.Context, typePayment model.TypePayment, userID string) (model.TypePayment, error) {
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

func (p PgRepository) FindAll(_ context.Context, userID string) ([]model.TypePayment, error) {
	var typePayments []model.TypePayment
	result := p.Gorm.Where("user_id=?", userID).Find(&typePayments)
	if err := result.Error; err != nil {
		return []model.TypePayment{}, err
	}
	return typePayments, nil
}

func (p PgRepository) FindByID(_ context.Context, id int, userID string) (model.TypePayment, error) {
	var typePayment model.TypePayment
	result := p.Gorm.Where("user_id=?", userID).First(&typePayment, id)
	if err := result.Error; err != nil {
		return model.TypePayment{}, err
	}
	return typePayment, nil
}

func (p PgRepository) Update(_ context.Context, id int, typePayment model.TypePayment, userID string) (model.TypePayment, error) {
	w, err := p.FindByID(context.Background(), id, userID)
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
