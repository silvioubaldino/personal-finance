package repository

import (
	"context"
	"time"

	"personal-finance/internal/model"

	"gorm.io/gorm"
)

type Repository interface {
	Add(ctx context.Context, wallet model.Wallet, userID string) (model.Wallet, error)
	FindAll(ctx context.Context, userID string) ([]model.Wallet, error)
	FindByID(ctx context.Context, id int, userID string) (model.Wallet, error)
	Update(ctx context.Context, id int, wallet model.Wallet, userID string) (model.Wallet, error)
	Delete(ctx context.Context, id int) error
}

type PgRepository struct {
	Gorm *gorm.DB
}

func NewPgRepository(gorm *gorm.DB) Repository {
	return PgRepository{Gorm: gorm}
}

func (p PgRepository) Add(_ context.Context, wallet model.Wallet, userID string) (model.Wallet, error) {
	now := time.Now()
	wallet.DateCreate = now
	wallet.DateUpdate = now
	wallet.UserID = userID
	result := p.Gorm.Create(&wallet)
	if err := result.Error; err != nil {
		return model.Wallet{}, err
	}
	return wallet, nil
}

func (p PgRepository) FindAll(_ context.Context, userID string) ([]model.Wallet, error) {
	var wallets []model.Wallet
	result := p.Gorm.Where("user_id=?", userID).Find(&wallets)
	if err := result.Error; err != nil {
		return []model.Wallet{}, err
	}
	return wallets, nil
}

func (p PgRepository) FindByID(_ context.Context, id int, userID string) (model.Wallet, error) {
	var wallet model.Wallet
	result := p.Gorm.Where("user_id=?", userID).First(&wallet, id)
	if err := result.Error; err != nil {
		return model.Wallet{}, err
	}
	return wallet, nil
}

func (p PgRepository) Update(_ context.Context, id int, wallet model.Wallet, userID string) (model.Wallet, error) {
	w, err := p.FindByID(context.Background(), id, userID)
	if err != nil {
		return model.Wallet{}, err
	}
	w.Description = wallet.Description
	w.Balance = wallet.Balance
	w.DateUpdate = time.Now()
	result := p.Gorm.Save(&w)
	if result.Error != nil {
		return model.Wallet{}, result.Error
	}
	return w, nil
}

func (p PgRepository) Delete(_ context.Context, id int) error {
	if err := p.Gorm.Delete(&model.Wallet{}, id).Error; err != nil {
		return err
	}
	return nil
}
