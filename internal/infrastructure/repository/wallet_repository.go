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

func (r *WalletRepository) Add(ctx context.Context, wallet domain.Wallet) (domain.Wallet, error) {
	userID := ctx.Value(authentication.UserID).(string)
	now := time.Now()
	id := uuid.New()

	dbModel := FromWalletDomain(wallet)
	dbModel.ID = &id
	dbModel.UserID = userID
	dbModel.DateCreate = now
	dbModel.DateUpdate = now
	if dbModel.InitialDate.IsZero() {
		dbModel.InitialDate = now
	}
	dbModel.Balance = dbModel.InitialBalance

	if err := r.db.WithContext(ctx).Create(&dbModel).Error; err != nil {
		return domain.Wallet{}, domain.WrapInternalError(err, "error creating wallet")
	}

	return dbModel.ToDomain(), nil
}

func (r *WalletRepository) AddConsistent(ctx context.Context, tx *gorm.DB, wallet domain.Wallet) (domain.Wallet, error) {
	userID := ctx.Value(authentication.UserID).(string)
	now := time.Now()
	id := uuid.New()

	dbModel := FromWalletDomain(wallet)
	dbModel.ID = &id
	dbModel.UserID = userID
	dbModel.DateCreate = now
	dbModel.DateUpdate = now
	if dbModel.InitialDate.IsZero() {
		dbModel.InitialDate = now
	}
	dbModel.Balance = dbModel.InitialBalance

	db := r.db.WithContext(ctx)
	if tx != nil {
		db = tx.WithContext(ctx)
	}

	if err := db.Create(&dbModel).Error; err != nil {
		return domain.Wallet{}, domain.WrapInternalError(err, "error creating wallet")
	}

	return dbModel.ToDomain(), nil
}

func (r *WalletRepository) FindAll(ctx context.Context) ([]domain.Wallet, error) {
	var wallets []WalletDB
	query := BuildBaseQuery(ctx, r.db, WalletDB{}.TableName())

	if err := query.Order("description").Find(&wallets).Error; err != nil {
		return nil, domain.WrapInternalError(err, "error finding wallets")
	}

	result := make([]domain.Wallet, len(wallets))
	for i, w := range wallets {
		result[i] = w.ToDomain()
	}
	return result, nil
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

func (r *WalletRepository) Update(ctx context.Context, wallet domain.Wallet) (domain.Wallet, error) {
	existing, err := r.FindByID(ctx, wallet.ID)
	if err != nil {
		return domain.Wallet{}, err
	}

	if wallet.Description != "" {
		existing.Description = wallet.Description
	}

	var shouldRecalculate bool
	if wallet.InitialBalance != 0 && wallet.InitialBalance != existing.InitialBalance {
		existing.InitialBalance = wallet.InitialBalance
		shouldRecalculate = true
	}
	if !wallet.InitialDate.IsZero() && wallet.InitialDate != existing.InitialDate {
		existing.InitialDate = wallet.InitialDate
		shouldRecalculate = true
	}

	existing.DateUpdate = time.Now()

	dbModel := FromWalletDomain(existing)
	if err := r.db.WithContext(ctx).Save(&dbModel).Error; err != nil {
		return domain.Wallet{}, domain.WrapInternalError(err, "error updating wallet")
	}

	if shouldRecalculate {
		if err := r.RecalculateBalance(ctx, wallet.ID); err != nil {
			return domain.Wallet{}, err
		}
		return r.FindByID(ctx, wallet.ID)
	}

	return dbModel.ToDomain(), nil
}

func (r *WalletRepository) Delete(ctx context.Context, id *uuid.UUID) error {
	userID := ctx.Value(authentication.UserID).(string)

	result := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&WalletDB{})

	if result.Error != nil {
		return domain.WrapInternalError(result.Error, "error deleting wallet")
	}

	if result.RowsAffected == 0 {
		return domain.WrapNotFound(ErrWalletNotFound, "wallet")
	}

	return nil
}

func (r *WalletRepository) RecalculateBalance(ctx context.Context, walletID *uuid.UUID) error {
	userID := ctx.Value(authentication.UserID).(string)

	wallet, err := r.FindByID(ctx, walletID)
	if err != nil {
		return err
	}

	var recalculatedBalance float64
	err = r.db.WithContext(ctx).
		Table("movements").
		Where("movements.user_id = ?", userID).
		Where("wallet_id = ?", walletID).
		Where("date BETWEEN ? AND ?", wallet.InitialDate, time.Now()).
		Where("is_paid = ?", true).
		Select("COALESCE(sum(amount), 0)").
		Row().Scan(&recalculatedBalance)
	if err != nil {
		return domain.WrapInternalError(fmt.Errorf("recalculate balance: %w", err), "wallet repository")
	}

	newBalance := wallet.InitialBalance + recalculatedBalance
	now := time.Now()

	result := r.db.WithContext(ctx).Model(&WalletDB{}).
		Where("id = ? AND user_id = ?", walletID, userID).
		Updates(map[string]interface{}{
			"balance":     newBalance,
			"date_update": now,
		})

	if result.Error != nil {
		return domain.WrapInternalError(result.Error, "error updating wallet balance")
	}

	return nil
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

func (r *WalletRepository) DeleteAllByUserID(ctx context.Context, tx *gorm.DB, userID string) error {
	db := r.db
	if tx != nil {
		db = tx
	}

	err := db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&WalletDB{}).Error
	if err != nil {
		return domain.WrapInternalError(err, "error deleting wallets")
	}

	return nil
}

func (r *WalletRepository) CountByUserID(ctx context.Context) (int64, error) {
	userID := authentication.UserIDFromContext(ctx)
	if userID == "" {
		return 0, domain.WrapInternalError(ErrWalletNotFound, "user_id not found in context")
	}

	var count int64
	err := r.db.WithContext(ctx).
		Model(&WalletDB{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	if err != nil {
		return 0, domain.WrapInternalError(err, "error counting wallets")
	}

	return count, nil
}
