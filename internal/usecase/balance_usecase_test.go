package usecase

import (
	"context"
	"testing"
	"time"

	"personal-finance/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestBalance_CalculateBalance(t *testing.T) {
	period := domain.Period{
		From: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC),
	}
	categoryID := uuid.New()

	type (
		input struct {
			period domain.Period
		}
		expected struct {
			output domain.Balance
			err    error
		}
	)

	tests := map[string]struct {
		input     input
		mockSetup func(mockMovRepo *MockMovementRepository, mockEstRepo *MockEstimateRepository)
		expected  expected
	}{
		"should sum paid expenses and income when there is no internal transfer": {
			input: input{period: period},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockEstRepo *MockEstimateRepository) {
				movements := domain.MovementList{
					dashboardMovementWithCategory(-300, dashboardDate(2026, 2, 5), true, &categoryID),
					dashboardMovement(1000, dashboardDate(2026, 2, 6), true),
				}
				mockMovRepo.On("FindByPeriod", period).Return(movements, nil)
				mockEstRepo.On("FindCategoriesByMonth", 2, 2026).Return([]domain.EstimateCategories{}, nil)
			},
			expected: expected{
				output: domain.Balance{Expense: -300, Income: 1000, PeriodBalance: 700},
				err:    nil,
			},
		},
		"should exclude internal transfer movements from expense and income": {
			input: input{period: period},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockEstRepo *MockEstimateRepository) {
				movements := domain.MovementList{
					dashboardMovementWithCategory(-300, dashboardDate(2026, 2, 5), true, &categoryID),
					dashboardMovement(1000, dashboardDate(2026, 2, 6), true),
					dashboardMovementWithPayment(-500, dashboardDate(2026, 2, 7), true, domain.TypePaymentInternalTransfer),
					dashboardMovementWithPayment(500, dashboardDate(2026, 2, 7), true, domain.TypePaymentInternalTransfer),
				}
				mockMovRepo.On("FindByPeriod", period).Return(movements, nil)
				mockEstRepo.On("FindCategoriesByMonth", 2, 2026).Return([]domain.EstimateCategories{}, nil)
			},
			expected: expected{
				output: domain.Balance{Expense: -300, Income: 1000, PeriodBalance: 700},
				err:    nil,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Arrange
			mockMovRepo := new(MockMovementRepository)
			mockEstRepo := new(MockEstimateRepository)
			tc.mockSetup(mockMovRepo, mockEstRepo)

			svc := NewBalance(mockMovRepo, mockEstRepo)

			// Act
			output, err := svc.CalculateBalance(context.Background(), tc.input.period)

			// Assert
			assert.ErrorIs(t, err, tc.expected.err)
			assert.Equal(t, tc.expected.output, output)
			mockMovRepo.AssertExpectations(t)
			mockEstRepo.AssertExpectations(t)
		})
	}
}
