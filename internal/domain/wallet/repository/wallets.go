package repository

import (
	"context"
	"fmt"
	"time"

	"personal-finance/internal/model"
	"personal-finance/internal/plataform/authentication"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	RecalculateBalance(ctx context.Context, walletID *uuid.UUID) error
	Add(ctx context.Context, wallet model.Wallet) (model.Wallet, error)
	FindAll(ctx context.Context) ([]model.Wallet, error)
	FindByID(ctx context.Context, id *uuid.UUID) (model.Wallet, error)
	Update(ctx context.Context, id *uuid.UUID, wallet model.Wallet) (model.Wallet, error)
	Delete(ctx context.Context, id *uuid.UUID) error
	UpdateConsistent(_ context.Context, tx *gorm.DB, wallet model.Wallet) (model.Wallet, error)
}

type PgRepository struct {
	Gorm *gorm.DB
}

func NewPgRepository(gorm *gorm.DB) Repository {
	return PgRepository{Gorm: gorm}
}

func (p PgRepository) RecalculateBalance(ctx context.Context, walletID *uuid.UUID) error {
	userID := ctx.Value(authentication.UserID).(string)
	var recalculatedBalance float64
	wallet, err := p.FindByID(ctx, walletID)
	if err != nil {
		return err
	}

	result := p.Gorm.
		Table("movements").
		Where(fmt.Sprintf("%s.user_id=?", "movements"), userID).
		Select("COALESCE(sum(amount), 0) as balance").
		Where("wallet_id=?", walletID).
		Where("date BETWEEN ? AND ?", wallet.InitialDate, time.Now()).
		Where("is_paid = ?", true).
		Scan(&recalculatedBalance)
	if err := result.Error; err != nil {
		return fmt.Errorf("repository error: %w", err)
	}

	wallet.Balance = wallet.InitialBalance + recalculatedBalance
	_, err = p.Update(ctx, walletID, wallet)
	if err != nil {
		return err
	}

	return nil
}

func (p PgRepository) Add(ctx context.Context, wallet model.Wallet) (model.Wallet, error) {
	userID := ctx.Value(authentication.UserID).(string)
	now := time.Now()
	id := uuid.New()

	wallet.ID = &id
	wallet.DateCreate = now
	wallet.DateUpdate = now
	if wallet.InitialDate.IsZero() {
		wallet.InitialDate = now
	}
	wallet.UserID = userID
	wallet.Balance = wallet.InitialBalance
	result := p.Gorm.Create(&wallet)
	if err := result.Error; err != nil {
		return model.Wallet{}, err
	}
	return wallet, nil
}

func (p PgRepository) FindAll(ctx context.Context) ([]model.Wallet, error) {
	userID := ctx.Value(authentication.UserID).(string)
	var wallets []model.Wallet
	result := p.Gorm.Where("user_id=?", userID).Order("description").Find(&wallets)
	if err := result.Error; err != nil {
		return []model.Wallet{}, err
	}
	return wallets, nil
}

func (p PgRepository) FindByID(ctx context.Context, id *uuid.UUID) (model.Wallet, error) {
	var wallet model.Wallet
	userID := ctx.Value(authentication.UserID).(string)
	result := p.Gorm.Where("user_id=?", userID).First(&wallet, id)
	if err := result.Error; err != nil {
		return model.Wallet{}, err
	}
	return wallet, nil
}

func (p PgRepository) Update(ctx context.Context, id *uuid.UUID, wallet model.Wallet) (model.Wallet, error) {
	w, err := p.FindByID(ctx, id)
	if err != nil {
		return model.Wallet{}, err
	}
	if wallet.Description != "" {
		w.Description = wallet.Description
	}
	if wallet.Balance != w.Balance && wallet.Balance != 0 {
		w.Balance = wallet.Balance
	}
	var shouldRecalculate bool
	if wallet.InitialBalance != w.InitialBalance && wallet.InitialBalance != 0 {
		w.InitialBalance = wallet.InitialBalance
		shouldRecalculate = true
	}
	if wallet.InitialDate != w.InitialDate && !wallet.InitialDate.IsZero() {
		w.InitialDate = wallet.InitialDate
		shouldRecalculate = true
	}
	w.DateUpdate = time.Now()
	result := p.Gorm.Save(&w)

	if shouldRecalculate {
		if err := p.RecalculateBalance(ctx, id); err != nil {
			return model.Wallet{}, err
		}
	}

	if result.Error != nil {
		return model.Wallet{}, result.Error
	}
	return w, nil
}

func (p PgRepository) Delete(_ context.Context, id *uuid.UUID) error {
	if err := p.Gorm.Delete(&model.Wallet{}, id).Error; err != nil {
		return err
	}
	return nil
}

func (p PgRepository) UpdateConsistent(ctx context.Context, tx *gorm.DB, wallet model.Wallet) (model.Wallet, error) {
	w, err := p.FindByID(ctx, wallet.ID)
	if err != nil {
		return model.Wallet{}, err
	}
	w.Description = wallet.Description
	w.Balance = wallet.Balance
	w.DateUpdate = time.Now()
	result := tx.Save(&w)
	if result.Error != nil {
		return model.Wallet{}, result.Error
	}
	return w, nil
}
