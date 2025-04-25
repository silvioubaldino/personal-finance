package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/plataform/authentication"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WalletRepository struct {
	db *gorm.DB
}

func NewWalletRepository(db *gorm.DB) *WalletRepository {
	return &WalletRepository{
		db: db,
	}
}

func (r *WalletRepository) FindByID(ctx context.Context, id *uuid.UUID) (domain.Wallet, error) {
	var wallet WalletDB
	userID := ctx.Value(authentication.UserID).(string)

	result := r.db.Where("user_id=?", userID).First(&wallet, id)
	if err := result.Error; err != nil {
		return domain.Wallet{}, fmt.Errorf("error finding wallet: %w", err)
	}

	return wallet.ToDomain(), nil
}

func (r *WalletRepository) UpdateAmount(ctx context.Context, tx *gorm.DB, id *uuid.UUID, balance float64) error {
	var isLocalTx bool
	if tx == nil {
		isLocalTx = true
		tx = r.db.Begin()
		defer tx.Rollback()
	}

	userID := ctx.Value(authentication.UserID).(string)
	now := time.Now()

	result := tx.Model(&WalletDB{}).
		Where("id = ? AND user_id = ?", id, userID).
		Updates(map[string]interface{}{
			"balance":     balance,
			"date_update": now,
		})

	if err := result.Error; err != nil {
		return fmt.Errorf("error updating wallet amount: %w", err)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("wallet with ID %s not found", id)
	}

	if isLocalTx {
		tx.Commit()
	}

	return nil
}

func (r *WalletRepository) Add(ctx context.Context, wallet domain.Wallet) (domain.Wallet, error) {
	return domain.Wallet{}, errors.New("method Add not implemented")
}

func (r *WalletRepository) AddConsistent(ctx context.Context, tx *gorm.DB, wallet domain.Wallet) (domain.Wallet, error) {
	return domain.Wallet{}, errors.New("method AddConsistent not implemented")
}

func (r *WalletRepository) FindAll(ctx context.Context) ([]domain.Wallet, error) {
	return nil, errors.New("method FindAll not implemented")
}

func (r *WalletRepository) Update(ctx context.Context, wallet domain.Wallet) (domain.Wallet, error) {
	return domain.Wallet{}, errors.New("method Update not implemented")
}

func (r *WalletRepository) Delete(ctx context.Context, ID *uuid.UUID) error {
	return errors.New("method Delete not implemented")
}

func (r *WalletRepository) RecalculateBalance(ctx context.Context, walletID *uuid.UUID) error {
	return errors.New("method RecalculateBalance not implemented")
}
