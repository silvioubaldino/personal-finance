package usecase

import (
	"context"
	"fmt"

	"personal-finance/internal/domain"
	"personal-finance/internal/infrastructure/repository/transaction"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CreditCardRepository interface {
	Add(ctx context.Context, tx *gorm.DB, creditCard domain.CreditCard) (domain.CreditCard, error)
	FindAll(ctx context.Context) ([]domain.CreditCard, error)
	FindByID(ctx context.Context, id uuid.UUID) (domain.CreditCard, error)
	FindNameByID(ctx context.Context, id uuid.UUID) (string, error)
	Update(ctx context.Context, tx *gorm.DB, id uuid.UUID, creditCard domain.CreditCard) (domain.CreditCard, error)
	Delete(ctx context.Context, tx *gorm.DB, id uuid.UUID) error
}

type CreditCard struct {
	repo      CreditCardRepository
	txManager transaction.Manager
}

func NewCreditCard(repo CreditCardRepository, txManager transaction.Manager) CreditCard {
	return CreditCard{
		repo:      repo,
		txManager: txManager,
	}
}

func (uc CreditCard) validateCreditCard(creditCard domain.CreditCard) error {
	if creditCard.ClosingDay < 1 || creditCard.ClosingDay > 31 {
		return ErrInvalidClosingDay
	}

	if creditCard.DueDay < 1 || creditCard.DueDay > 31 {
		return ErrInvalidDueDay
	}

	if creditCard.CreditLimit < 0 {
		return ErrInvalidCreditLimit
	}

	return nil
}

func (uc CreditCard) Add(ctx context.Context, creditCard domain.CreditCard) (domain.CreditCard, error) {
	if err := uc.validateCreditCard(creditCard); err != nil {
		return domain.CreditCard{}, err
	}

	var result domain.CreditCard
	err := uc.txManager.WithTransaction(ctx, func(tx *gorm.DB) error {
		createdCreditCard, err := uc.repo.Add(ctx, tx, creditCard)
		if err != nil {
			return fmt.Errorf("error adding credit card: %w", err)
		}
		result = createdCreditCard
		return nil
	})

	if err != nil {
		return domain.CreditCard{}, err
	}

	return result, nil
}

func (uc CreditCard) FindAll(ctx context.Context) ([]domain.CreditCard, error) {
	result, err := uc.repo.FindAll(ctx)
	if err != nil {
		return []domain.CreditCard{}, fmt.Errorf("error finding credit cards: %w", err)
	}
	return result, nil
}

func (uc CreditCard) FindByID(ctx context.Context, id uuid.UUID) (domain.CreditCard, error) {
	result, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return domain.CreditCard{}, fmt.Errorf("error finding credit card: %w", err)
	}
	return result, nil
}

func (uc CreditCard) Update(ctx context.Context, id uuid.UUID, creditCard domain.CreditCard) (domain.CreditCard, error) {
	if err := uc.validateCreditCard(creditCard); err != nil {
		return domain.CreditCard{}, err
	}

	var result domain.CreditCard
	err := uc.txManager.WithTransaction(ctx, func(tx *gorm.DB) error {
		updatedCreditCard, err := uc.repo.Update(ctx, tx, id, creditCard)
		if err != nil {
			return fmt.Errorf("error updating credit card: %w", err)
		}
		result = updatedCreditCard
		return nil
	})

	if err != nil {
		return domain.CreditCard{}, err
	}

	return result, nil
}

func (uc CreditCard) Delete(ctx context.Context, id uuid.UUID) error {
	err := uc.txManager.WithTransaction(ctx, func(tx *gorm.DB) error {
		if err := uc.repo.Delete(ctx, tx, id); err != nil {
			return fmt.Errorf("error deleting credit card: %w", err)
		}
		return nil
	})

	if err != nil {
		return err
	}

	return nil
}
