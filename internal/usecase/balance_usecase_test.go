package usecase

import (
	"context"
	"testing"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/domain/fixture"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestBalance_CalculateBalance(t *testing.T) {
	type (
		input struct {
			period domain.Period
		}
		expected struct {
			output domain.Balance
			err    error
		}
	)

	categoryID := uuid.New()

	tests := map[string]struct {
		input     input
		mockSetup func(mockMovRepo *MockMovementRepository, mockEstRepo *MockEstimateRepository)
		expected  expected
	}{
		"should sum paid expenses and income when there is no internal transfer": {
			input: input{period: domain.Period{
				From: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC),
			}},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockEstRepo *MockEstimateRepository) {
				period := domain.Period{
					From: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
					To:   time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC),
				}
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
			input: input{period: domain.Period{
				From: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC),
			}},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockEstRepo *MockEstimateRepository) {
				period := domain.Period{
					From: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
					To:   time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC),
				}
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
		"should accumulate two paid movements of the same category before applying the estimate ceiling": {
			input: input{period: domain.Period{
				From: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2026, 8, 31, 23, 59, 59, 0, time.UTC),
			}},
			mockSetup: func(mockMovRepo *MockMovementRepository, mockEstRepo *MockEstimateRepository) {
				period := domain.Period{
					From: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
					To:   time.Date(2026, 8, 31, 23, 59, 59, 0, time.UTC),
				}
				mockMovRepo.On("FindByPeriod", period).Return(domain.MovementList{
					fixture.MovementMock(fixture.AsMovementExpense(700), fixture.WithMovementIsPaid(true)),
					fixture.MovementMock(fixture.AsMovementExpense(600), fixture.WithMovementIsPaid(true)),
				}, nil)
				mockEstRepo.On("FindCategoriesByMonth", 8, 2026).Return([]domain.EstimateCategories{
					{CategoryID: &fixture.CategoryID, Amount: -1000},
				}, nil)
			},
			expected: expected{
				output: domain.Balance{Expense: -1300, Income: 0, PeriodBalance: -1300},
				err:    nil,
			},
		},
		"should return error when the movement repository fails": {
			input: input{period: domain.Period{
				From: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2026, 8, 31, 23, 59, 59, 0, time.UTC),
			}},
			mockSetup: func(mockMovRepo *MockMovementRepository, _ *MockEstimateRepository) {
				period := domain.Period{
					From: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
					To:   time.Date(2026, 8, 31, 23, 59, 59, 0, time.UTC),
				}
				mockMovRepo.On("FindByPeriod", period).Return(domain.MovementList{}, assert.AnError)
			},
			expected: expected{
				output: domain.Balance{},
				err:    assert.AnError,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Arrange
			var (
				mockMovRepo = &MockMovementRepository{}
				mockEstRepo = &MockEstimateRepository{}
				svc         = NewBalance(mockMovRepo, mockEstRepo)
			)
			defer mockMovRepo.AssertExpectations(t)
			defer mockEstRepo.AssertExpectations(t)
			tc.mockSetup(mockMovRepo, mockEstRepo)

			// Act
			output, err := svc.CalculateBalance(context.Background(), tc.input.period)

			// Assert
			assert.ErrorIs(t, err, tc.expected.err)
			assert.Equal(t, tc.expected.output, output)
		})
	}
}
