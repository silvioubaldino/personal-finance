package repository

import (
	"context"
	"errors"
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
	query := BuildBaseQuery(ctx, r.db, wallet.TableName())

	result := query.First(&wallet, id)
	if err := result.Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Wallet{}, domain.WrapNotFound(err, "wallet")
		}
		return domain.Wallet{}, domain.WrapInternalError(err, "error finding wallet")
	}

	return wallet.ToDomain(), nil
}

func (r *WalletRepository) UpdateAmount(ctx context.Context, tx *gorm.DB, id *uuid.UUID, balance float64) error {
	var isLocalTx bool
	if tx == nil {
		isLocalTx = true
		tx = r.db.WithContext(ctx).Begin()
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
		return domain.WrapInternalError(err, "error updating wallet amount")
	}

	if result.RowsAffected == 0 {
		return domain.WrapNotFound(ErrWalletNotFound, "wallet")
	}

	if isLocalTx {
		if err := tx.Commit().Error; err != nil {
			return domain.WrapInternalError(err, "error committing transaction")
		}
	}

	return nil
}

func (r *WalletRepository) Add(ctx context.Context, wallet domain.Wallet) (domain.Wallet, error) {
	return domain.Wallet{}, domain.WrapInternalError(errors.New("method Add not implemented"), "wallet repository")
}

func (r *WalletRepository) AddConsistent(ctx context.Context, tx *gorm.DB, wallet domain.Wallet) (domain.Wallet, error) {
	return domain.Wallet{}, domain.WrapInternalError(errors.New("method AddConsistent not implemented"), "wallet repository")
}

func (r *WalletRepository) FindAll(ctx context.Context) ([]domain.Wallet, error) {
	return nil, domain.WrapInternalError(errors.New("method FindAll not implemented"), "wallet repository")
}

func (r *WalletRepository) Update(ctx context.Context, wallet domain.Wallet) (domain.Wallet, error) {
	return domain.Wallet{}, domain.WrapInternalError(errors.New("method Update not implemented"), "wallet repository")
}

func (r *WalletRepository) Delete(ctx context.Context, ID *uuid.UUID) error {
	return domain.WrapInternalError(errors.New("method Delete not implemented"), "wallet repository")
}

func (r *WalletRepository) RecalculateBalance(ctx context.Context, walletID *uuid.UUID) error {
	return domain.WrapInternalError(errors.New("method RecalculateBalance not implemented"), "wallet repository")
}
