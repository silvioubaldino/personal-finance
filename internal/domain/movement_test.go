package domain_test

import (
	"testing"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/domain/fixture"

	"github.com/stretchr/testify/assert"
)

func TestMovement_GenerateInstallmentMovements(t *testing.T) {
	tests := map[string]struct {
		movementInput      domain.Movement
		expectedCount      int
		expectedError      bool
		validateInstalment func(t *testing.T, installments domain.MovementList, input domain.Movement)
	}{
		"should generate installment movements with valid data": {
			movementInput: fixture.MovementMock(
				fixture.WithMovementDescription("Compra Parcelada"),
				fixture.AsMovementExpense(100.50),
				fixture.WithMovementIsPaid(false),
				fixture.WithMovementTypePayment(string(domain.TypePaymentCreditCard)),
				fixture.WithMovementCreditCardID(&fixture.CreditCardID),
				fixture.WithMovementInstallment(1, 5),
				fixture.WithMovementDate(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)),
			),
			expectedCount: 5,
			expectedError: false,
			validateInstalment: func(t *testing.T, installments domain.MovementList, input domain.Movement) {
				baseDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

				groupID := installments[0].CreditCardInfo.InstallmentGroupID
				for i, installment := range installments {
					assert.NotNil(t, installment.ID)
					assert.Equal(t, input.Description, installment.Description)
					assert.Equal(t, input.Amount, installment.Amount)
					assert.Equal(t, input.UserID, installment.UserID)
					assert.Equal(t, input.TypePayment, installment.TypePayment)
					assert.Equal(t, input.CategoryID, installment.CategoryID)
					assert.Equal(t, input.SubCategoryID, installment.SubCategoryID)

					assert.False(t, installment.IsPaid)
					assert.False(t, installment.IsRecurrent)

					expectedDate := baseDate.AddDate(0, i, 0)
					assert.Equal(t, expectedDate, *installment.Date)

					assert.NotNil(t, installment.CreditCardInfo)
					assert.Equal(t, input.CreditCardInfo.CreditCardID, installment.CreditCardInfo.CreditCardID)
					assert.Equal(t, input.CreditCardInfo.TotalInstallments, installment.CreditCardInfo.TotalInstallments)
					assert.Equal(t, i+1, *installment.CreditCardInfo.InstallmentNumber)
					assert.Equal(t, groupID, installment.CreditCardInfo.InstallmentGroupID)
				}
			},
		},
		"should return empty list when movement is not installment movement": {
			movementInput: fixture.MovementMock(
				fixture.WithMovementDescription("Compra no cartão à vista"),
				fixture.AsMovementExpense(100.50),
				fixture.WithMovementTypePayment(string(domain.TypePaymentCreditCard)),
				fixture.WithMovementCreditCardID(&fixture.CreditCardID),
				fixture.WithMovementDate(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)),
			),
			expectedCount: 0,
			expectedError: false,
			validateInstalment: func(t *testing.T, installments domain.MovementList, input domain.Movement) {
				assert.Empty(t, installments)
				assert.False(t, input.IsInstallmentMovement())
			},
		},
		"should return empty list when CreditCardInfo is nil": {
			movementInput: fixture.MovementMock(
				fixture.WithMovementDescription("Compra sem cartão"),
				fixture.AsMovementExpense(100.50),
				fixture.WithMovementDate(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)),
			),
			expectedCount: 0,
			expectedError: false,
			validateInstalment: func(t *testing.T, installments domain.MovementList, input domain.Movement) {
				assert.Empty(t, installments)
			},
		},
		"should return empty list when InstallmentNumber is nil": {
			movementInput: func() domain.Movement {
				m := fixture.MovementMock(
					fixture.WithMovementDescription("Compra Parcelada"),
					fixture.AsMovementExpense(100.50),
					fixture.WithMovementCreditCardID(&fixture.CreditCardID),
					fixture.WithMovementDate(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)),
				)
				m.CreditCardInfo = &domain.CreditCardMovement{
					CreditCardID:      &fixture.CreditCardID,
					InstallmentNumber: nil,
					TotalInstallments: intPtr(5),
				}
				return m
			}(),
			expectedCount: 0,
			expectedError: false,
			validateInstalment: func(t *testing.T, installments domain.MovementList, input domain.Movement) {
				assert.Empty(t, installments)
			},
		},
		"should return empty list when TotalInstallments is nil": {
			movementInput: func() domain.Movement {
				m := fixture.MovementMock(
					fixture.WithMovementDescription("Compra Parcelada"),
					fixture.AsMovementExpense(100.50),
					fixture.WithMovementCreditCardID(&fixture.CreditCardID),
					fixture.WithMovementDate(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)),
				)
				m.CreditCardInfo = &domain.CreditCardMovement{
					CreditCardID:      &fixture.CreditCardID,
					InstallmentNumber: intPtr(1),
					TotalInstallments: nil,
				}
				return m
			}(),
			expectedCount: 0,
			expectedError: false,
			validateInstalment: func(t *testing.T, installments domain.MovementList, input domain.Movement) {
				assert.Empty(t, installments)
			},
		},
		"should return last installment when on final installment": {
			movementInput: fixture.MovementMock(
				fixture.WithMovementDescription("Compra Parcelada - Última parcela"),
				fixture.AsMovementExpense(100.50),
				fixture.WithMovementTypePayment(string(domain.TypePaymentCreditCard)),
				fixture.WithMovementCreditCardID(&fixture.CreditCardID),
				fixture.WithMovementInstallment(5, 5),
				fixture.WithMovementDate(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)),
			),
			expectedCount: 1,
			expectedError: false,
			validateInstalment: func(t *testing.T, installments domain.MovementList, input domain.Movement) {
				assert.Len(t, installments, 1)
				assert.Equal(t, 5, *installments[0].CreditCardInfo.InstallmentNumber)

				baseDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
				assert.Equal(t, baseDate, *installments[0].Date)
			},
		},
		"should return empty list when installment number exceeds total": {
			movementInput: fixture.MovementMock(
				fixture.WithMovementDescription("Compra Parcelada - Número inválido"),
				fixture.AsMovementExpense(100.50),
				fixture.WithMovementTypePayment(string(domain.TypePaymentCreditCard)),
				fixture.WithMovementCreditCardID(&fixture.CreditCardID),
				fixture.WithMovementInstallment(6, 5),
				fixture.WithMovementDate(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)),
			),
			expectedCount: 0,
			expectedError: false,
			validateInstalment: func(t *testing.T, installments domain.MovementList, input domain.Movement) {
				assert.Empty(t, installments)
			},
		},
		"should return empty list when remaining installments is negative": {
			movementInput: fixture.MovementMock(
				fixture.WithMovementDescription("Compra Parcelada - Parcela além do limite"),
				fixture.AsMovementExpense(100.50),
				fixture.WithMovementTypePayment(string(domain.TypePaymentCreditCard)),
				fixture.WithMovementCreditCardID(&fixture.CreditCardID),
				fixture.WithMovementInstallment(8, 5),
				fixture.WithMovementDate(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)),
			),
			expectedCount: 0,
			expectedError: false,
			validateInstalment: func(t *testing.T, installments domain.MovementList, input domain.Movement) {
				assert.Empty(t, installments)
				remainingInstallments := *input.CreditCardInfo.TotalInstallments - *input.CreditCardInfo.InstallmentNumber
				assert.Less(t, remainingInstallments, 0)
			},
		},
		"should generate remaining installments for second-to-last installment": {
			movementInput: fixture.MovementMock(
				fixture.WithMovementDescription("Compra Parcelada - Penúltima"),
				fixture.AsMovementExpense(100.50),
				fixture.WithMovementTypePayment(string(domain.TypePaymentCreditCard)),
				fixture.WithMovementCreditCardID(&fixture.CreditCardID),
				fixture.WithMovementInstallment(4, 5),
				fixture.WithMovementDate(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)),
			),
			expectedCount: 2,
			expectedError: false,
			validateInstalment: func(t *testing.T, installments domain.MovementList, input domain.Movement) {
				assert.Len(t, installments, 2)

				assert.Equal(t, 4, *installments[0].CreditCardInfo.InstallmentNumber)
				baseDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
				assert.Equal(t, baseDate, *installments[0].Date)

				assert.Equal(t, 5, *installments[1].CreditCardInfo.InstallmentNumber)
				expectedDate2 := baseDate.AddDate(0, 1, 0)
				assert.Equal(t, expectedDate2, *installments[1].Date)
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			installments := tt.movementInput.GenerateInstallmentMovements()

			assert.Len(t, installments, tt.expectedCount)

			if tt.validateInstalment != nil {
				tt.validateInstalment(t, installments, tt.movementInput)
			}
		})
	}
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}
