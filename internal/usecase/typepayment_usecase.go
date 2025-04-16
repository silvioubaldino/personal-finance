package usecase

import (
	"context"
	"fmt"

	"personal-finance/internal/domain"
)

type TypePaymentRepository interface {
	Add(ctx context.Context, typepayment domain.TypePayment, userID string) (domain.TypePayment, error)
	FindAll(ctx context.Context, userID string) ([]domain.TypePayment, error)
	FindByID(ctx context.Context, ID int, userID string) (domain.TypePayment, error)
	Update(ctx context.Context, typepayment domain.TypePayment, userID string) (domain.TypePayment, error)
	Delete(ctx context.Context, ID int) error
}

type TypePayment interface {
	Add(ctx context.Context, typepayment domain.TypePayment, userID string) (domain.TypePayment, error)
	FindAll(ctx context.Context, userID string) ([]domain.TypePayment, error)
	FindByID(ctx context.Context, ID int, userID string) (domain.TypePayment, error)
	Update(ctx context.Context, typepayment domain.TypePayment, userID string) (domain.TypePayment, error)
	Delete(ctx context.Context, ID int) error
}

type typePaymentUseCase struct {
	repo TypePaymentRepository
}

func NewTypePayment(repo TypePaymentRepository) TypePayment {
	return typePaymentUseCase{
		repo: repo,
	}
}

func (uc typePaymentUseCase) Add(ctx context.Context, typepayment domain.TypePayment, userID string) (domain.TypePayment, error) {
	result, err := uc.repo.Add(ctx, typepayment, userID)
	if err != nil {
		return domain.TypePayment{}, fmt.Errorf("erro ao adicionar tipo de pagamento: %w", err)
	}
	return result, nil
}

func (uc typePaymentUseCase) FindAll(ctx context.Context, userID string) ([]domain.TypePayment, error) {
	resultList, err := uc.repo.FindAll(ctx, userID)
	if err != nil {
		return []domain.TypePayment{}, fmt.Errorf("erro ao buscar tipos de pagamento: %w", err)
	}
	return resultList, nil
}

func (uc typePaymentUseCase) FindByID(ctx context.Context, id int, userID string) (domain.TypePayment, error) {
	result, err := uc.repo.FindByID(ctx, id, userID)
	if err != nil {
		return domain.TypePayment{}, fmt.Errorf("erro ao buscar tipo de pagamento: %w", err)
	}
	return result, nil
}

func (uc typePaymentUseCase) Update(ctx context.Context, typepayment domain.TypePayment, userID string) (domain.TypePayment, error) {
	result, err := uc.repo.Update(ctx, typepayment, userID)
	if err != nil {
		return domain.TypePayment{}, fmt.Errorf("erro ao atualizar tipo de pagamento: %w", err)
	}
	return result, nil
}

func (uc typePaymentUseCase) Delete(ctx context.Context, id int) error {
	err := uc.repo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("erro ao deletar tipo de pagamento: %w", err)
	}
	return nil
}
