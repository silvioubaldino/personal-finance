package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/infrastructure/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DeleteOne remove uma ocorrência específica de uma série recorrente ou uma movimentação avulsa.
// Para recorrências: Split da série (Original encerra T-1, Nova inicia T+1 com dados ORIGINAIS).
// A movement do mês T é deletada. Se estava paga, reverte o saldo.
func (u *Movement) DeleteOne(ctx context.Context, id uuid.UUID, targetDate time.Time) error {
	return u.txManager.WithTransaction(ctx, func(tx *gorm.DB) error {
		return u.executeDeleteOne(ctx, tx, id, targetDate)
	})
}

func (u *Movement) executeDeleteOne(ctx context.Context, tx *gorm.DB, id uuid.UUID, targetDate time.Time) error {
	existingMovement, isVirtual, recurrentID, err := u.resolveMovementForDelete(ctx, id, targetDate)
	if err != nil {
		return err
	}

	effectiveTargetDate := targetDate
	if !isVirtual && existingMovement.Date != nil {
		effectiveTargetDate = *existingMovement.Date
	}

	if recurrentID != nil {
		if err := u.splitRecurrentSeriesForDeleteOne(ctx, tx, *recurrentID, effectiveTargetDate); err != nil {
			return err
		}
	}

	// Se é uma movement virtual (não persistida), não há nada mais a fazer
	// O split já criou a nova recorrência, e a ocorrência T simplesmente não existirá
	if isVirtual {
		return nil
	}

	// Se a movement estava paga, reverte o saldo
	if existingMovement.IsPaid {
		if err := u.revertWalletBalance(ctx, tx, existingMovement); err != nil {
			return err
		}
	}

	// Deleta a movement persistida
	return u.movementRepo.Delete(ctx, tx, id)
}

// resolveMovementForDelete identifica se a movement existe e se é recorrente.
// Retorna: (movement existente ou virtual, isVirtual, recurrentID, error)
func (u *Movement) resolveMovementForDelete(ctx context.Context, id uuid.UUID, targetDate time.Time) (domain.Movement, bool, *uuid.UUID, error) {
	// Tenta encontrar a movement persistida
	existingMovement, err := u.movementRepo.FindByID(ctx, id)
	if err == nil {
		// Movement existe, verifica se é vinculada a uma recorrência
		return existingMovement, false, existingMovement.RecurrentID, nil
	}

	if !errors.Is(err, repository.ErrMovementNotFound) {
		return domain.Movement{}, false, nil, err
	}

	// Movement não encontrada, pode ser uma movement virtual de recorrência
	recurrent, err := u.recurrentRepo.FindByID(ctx, id)
	if err != nil {
		return domain.Movement{}, false, nil, fmt.Errorf("movement or recurrent not found: %w", err)
	}

	// É uma movement virtual - criamos uma representação para retornar
	virtualMovement := domain.FromRecurrentMovement(recurrent, targetDate)

	return virtualMovement, true, recurrent.ID, nil
}

// splitRecurrentSeriesForDeleteOne encerra a recorrência original em T-1 e cria
// uma nova recorrência com os dados ORIGINAIS iniciando em T+1.
// A ocorrência T simplesmente não existirá na nova série.
func (u *Movement) splitRecurrentSeriesForDeleteOne(ctx context.Context, tx *gorm.DB, recurrentID uuid.UUID, targetDate time.Time) error {
	recurrent, err := u.recurrentRepo.FindByID(ctx, recurrentID)
	if err != nil {
		return fmt.Errorf("error finding recurrent: %w", err)
	}

	// Preserva o EndDate original ANTES de modificar a recorrência
	originalEndDate := recurrent.EndDate

	targetMonth := targetDate.Month()
	targetYear := targetDate.Year()

	// Encerra a recorrência original no mês anterior (T-1)
	endDate := domain.SetMonthYear(*recurrent.InitialDate, targetMonth-1, targetYear)

	// Se a endDate calculada for antes ou igual à initialDate, significa que estamos
	// deletando no primeiro mês da recorrência - devemos DELETAR a recorrência inteira
	if !endDate.After(*recurrent.InitialDate) {
		// Se não há nova recorrência a criar (era o último mês ou delete no primeiro mês sem meses futuros),
		// simplesmente deleta a recorrência
		if originalEndDate != nil {
			newInitialDate := domain.SetMonthYear(*recurrent.InitialDate, targetMonth+1, targetYear)
			if !newInitialDate.Before(*originalEndDate) {
				return u.recurrentRepo.Delete(ctx, tx, recurrentID)
			}
		}

		// Se há meses futuros, deleta a recorrência atual e cria nova começando em T+1
		if err := u.recurrentRepo.Delete(ctx, tx, recurrentID); err != nil {
			return fmt.Errorf("error deleting recurrent: %w", err)
		}

		newRecurrent := u.buildNewRecurrentFromOriginal(recurrent, targetMonth, targetYear)
		if _, err := u.recurrentRepo.Add(ctx, tx, newRecurrent); err != nil {
			return fmt.Errorf("error creating new recurrent series: %w", err)
		}

		return nil
	}

	recurrent.EndDate = &endDate
	if _, err := u.recurrentRepo.Update(ctx, tx, &recurrentID, recurrent); err != nil {
		return fmt.Errorf("error updating recurrent end date: %w", err)
	}

	// Calcula a data inicial da nova recorrência (T+1)
	newInitialDate := domain.SetMonthYear(*recurrent.InitialDate, targetMonth+1, targetYear)

	// Se a recorrência original já terminava no mês do delete (ou antes de T+1),
	// NÃO cria nova recorrência - a série simplesmente encerra aqui
	if originalEndDate != nil && !newInitialDate.Before(*originalEndDate) {
		return nil
	}

	// Restaura o EndDate original para passar para a nova recorrência
	recurrent.EndDate = originalEndDate

	// Cria nova recorrência com dados ORIGINAIS iniciando em T+1
	newRecurrent := u.buildNewRecurrentFromOriginal(recurrent, targetMonth, targetYear)

	if _, err := u.recurrentRepo.Add(ctx, tx, newRecurrent); err != nil {
		return fmt.Errorf("error creating new recurrent series: %w", err)
	}

	return nil
}

// buildNewRecurrentFromOriginal cria uma nova RecurrentMovement com os dados ORIGINAIS.
// Usado em DeleteOne para manter a série futura inalterada.
// IMPORTANTE: Preserva o EndDate original para manter o limite da série.
func (u *Movement) buildNewRecurrentFromOriginal(
	originalRecurrent domain.RecurrentMovement,
	targetMonth time.Month,
	targetYear int,
) domain.RecurrentMovement {
	newInitialDate := domain.SetMonthYear(*originalRecurrent.InitialDate, targetMonth+1, targetYear)

	return domain.RecurrentMovement{
		ID:            nil,
		Description:   originalRecurrent.Description,
		Amount:        originalRecurrent.Amount,
		InitialDate:   &newInitialDate,
		EndDate:       originalRecurrent.EndDate, // Preserva o EndDate original
		WalletID:      originalRecurrent.WalletID,
		CategoryID:    originalRecurrent.CategoryID,
		SubCategoryID: originalRecurrent.SubCategoryID,
		TypePayment:   originalRecurrent.TypePayment,
		UserID:        originalRecurrent.UserID,
	}
}

// revertWalletBalance reverte o impacto de uma movement paga no saldo da carteira.
func (u *Movement) revertWalletBalance(ctx context.Context, tx *gorm.DB, movement domain.Movement) error {
	if movement.WalletID == nil {
		return nil
	}

	// Reverte o saldo: se amount era negativo (despesa), adiciona de volta; se era positivo (receita), subtrai
	revertAmount := -movement.Amount
	if err := u.walletRepo.UpdateAmount(ctx, tx, movement.WalletID, revertAmount); err != nil {
		return fmt.Errorf("error reverting wallet balance: %w", err)
	}

	return nil
}
