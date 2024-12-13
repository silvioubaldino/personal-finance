package repository

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"time"

	"personal-finance/internal/model"

	"gorm.io/gorm"
)

type Repository interface {
	RecalculateBalance(ctx context.Context, walletID *uuid.UUID, userID string) error
	Add(ctx context.Context, wallet model.Wallet, userID string) (model.Wallet, error)
	FindAll(ctx context.Context, userID string) ([]model.Wallet, error)
	FindByID(ctx context.Context, id *uuid.UUID, userID string) (model.Wallet, error)
	Update(ctx context.Context, id *uuid.UUID, wallet model.Wallet, userID string) (model.Wallet, error)
	Delete(ctx context.Context, id *uuid.UUID) error
	UpdateConsistent(_ context.Context, tx *gorm.DB, wallet model.Wallet, userID string) (model.Wallet, error)
}

type PgRepository struct {
	Gorm *gorm.DB
}

func NewPgRepository(gorm *gorm.DB) Repository {
	return PgRepository{Gorm: gorm}
}

func (p PgRepository) RecalculateBalance(ctx context.Context, walletID *uuid.UUID, userID string) error {
	var recalculatedBalance float64
	wallet, err := p.FindByID(ctx, walletID, userID)
	if err != nil {
		return err
	}

	result := p.Gorm.
		Table("movements").
		Where(fmt.Sprintf("%s.user_id=?", "movements"), userID).
		Select("COALESCE(sum(amount), 0) as balance").
		Where("wallet_id=?", walletID).
		Where("date BETWEEN ? AND ?", wallet.InitialDate, time.Now()).
		Where("status_id = ?", model.TransactionStatusPaidID).
		Scan(&recalculatedBalance)
	if err := result.Error; err != nil {
		return fmt.Errorf("repository error: %w", err)
	}

	wallet.Balance = wallet.InitialBalance + recalculatedBalance
	_, err = p.Update(ctx, walletID, wallet, userID)
	if err != nil {
		return err
	}

	return nil
}

func (p PgRepository) Add(_ context.Context, wallet model.Wallet, userID string) (model.Wallet, error) {
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

func (p PgRepository) FindAll(_ context.Context, userID string) ([]model.Wallet, error) {
	var wallets []model.Wallet
	result := p.Gorm.Where("user_id=?", userID).Order("description").Find(&wallets)
	if err := result.Error; err != nil {
		return []model.Wallet{}, err
	}
	return wallets, nil
}

func (p PgRepository) FindByID(_ context.Context, id *uuid.UUID, userID string) (model.Wallet, error) {
	var wallet model.Wallet
	result := p.Gorm.Where("user_id=?", userID).First(&wallet, id)
	if err := result.Error; err != nil {
		return model.Wallet{}, err
	}
	return wallet, nil
}

func (p PgRepository) Update(_ context.Context, id *uuid.UUID, wallet model.Wallet, userID string) (model.Wallet, error) {
	w, err := p.FindByID(context.Background(), id, userID)
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
		if err := p.RecalculateBalance(context.Background(), id, userID); err != nil {
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

func (p PgRepository) UpdateConsistent(_ context.Context, tx *gorm.DB, wallet model.Wallet, userID string) (model.Wallet, error) {
	w, err := p.FindByID(context.Background(), wallet.ID, userID)
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
