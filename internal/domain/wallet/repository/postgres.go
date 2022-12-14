package repository

import (
	"context"
	"time"

	"personal-finance/internal/model"

	"gorm.io/gorm"
)

type Repository interface {
	Add(ctx context.Context, wallet model.Wallet) (model.Wallet, error)
	FindAll(ctx context.Context) ([]model.Wallet, error)
	FindByID(ctx context.Context, id int) (model.Wallet, error)
	Update(ctx context.Context, id int, wallet model.Wallet) (model.Wallet, error)
	Delete(ctx context.Context, id int) error
}

type PgRepository struct {
	Gorm *gorm.DB
}

func NewPgRepository(gorm *gorm.DB) Repository {
	return PgRepository{Gorm: gorm}
}

func (p PgRepository) Add(_ context.Context, wallet model.Wallet) (model.Wallet, error) {
	now := time.Now()
	wallet.DateCreate = now
	wallet.DateUpdate = now
	result := p.Gorm.Create(&wallet)
	if err := result.Error; err != nil {
		return model.Wallet{}, err
	}
	return wallet, nil
}

func (p PgRepository) FindAll(_ context.Context) ([]model.Wallet, error) {
	var wallets []model.Wallet
	result := p.Gorm.Find(&wallets)
	if err := result.Error; err != nil {
		return []model.Wallet{}, err
	}
	return wallets, nil
}

func (p PgRepository) FindByID(_ context.Context, id int) (model.Wallet, error) {
	var wallet model.Wallet
	result := p.Gorm.First(&wallet, id)
	if err := result.Error; err != nil {
		return model.Wallet{}, err
	}
	return wallet, nil
}

func (p PgRepository) Update(_ context.Context, id int, wallet model.Wallet) (model.Wallet, error) {
	w, err := p.FindByID(context.Background(), id)
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
