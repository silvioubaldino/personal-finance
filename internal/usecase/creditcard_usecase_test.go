package usecase

import (
	"context"
	"testing"

	"personal-finance/internal/domain"
	"personal-finance/internal/domain/fixture"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

func TestCreditCard_Add(t *testing.T) {
	tests := map[string]struct {
		creditCardInput    domain.CreditCard
		mockSetup          func(mockRepo *MockCreditCardRepository, mockTxManager *MockTransactionManager)
		expectedCreditCard domain.CreditCard
		expectedError      error
	}{
		"should add credit card with success": {
			creditCardInput: fixture.CreditCardMock(
				fixture.WithCreditCardName("Nubank"),
				fixture.WithCreditCardLimit(2000.0),
			),
			mockSetup: func(mockRepo *MockCreditCardRepository, mockTxManager *MockTransactionManager) {
				creditCard := fixture.CreditCardMock(
					fixture.WithCreditCardName("Nubank"),
					fixture.WithCreditCardLimit(2000.0),
				)

				mockTxManager.On("WithTransaction", mock.Anything, mock.Anything).Return(nil)

				mockRepo.On("Add", mock.Anything, mock.Anything, mock.Anything).Return(creditCard, nil)
			},
			expectedCreditCard: fixture.CreditCardMock(
				fixture.WithCreditCardName("Nubank"),
				fixture.WithCreditCardLimit(2000.0),
			),
			expectedError: nil,
		},
		"should fail with invalid closing day": {
			creditCardInput: fixture.CreditCardMock(
				fixture.WithCreditCardClosingDay(35),
			),
			mockSetup: func(mockRepo *MockCreditCardRepository, mockTxManager *MockTransactionManager) {
			},
			expectedCreditCard: domain.CreditCard{},
			expectedError:      ErrInvalidClosingDay,
		},
		"should fail with invalid due day": {
			creditCardInput: fixture.CreditCardMock(
				fixture.WithCreditCardDueDay(0),
			),
			mockSetup: func(mockRepo *MockCreditCardRepository, mockTxManager *MockTransactionManager) {
			},
			expectedCreditCard: domain.CreditCard{},
			expectedError:      ErrInvalidDueDay,
		},
		"should fail with negative credit limit": {
			creditCardInput: fixture.CreditCardMock(
				fixture.WithCreditCardLimit(-1000.0),
			),
			mockSetup: func(mockRepo *MockCreditCardRepository, mockTxManager *MockTransactionManager) {
			},
			expectedCreditCard: domain.CreditCard{},
			expectedError:      ErrInvalidCreditLimit,
		},
		"should fail when repo.Add returns error": {
			creditCardInput: fixture.CreditCardMock(
				fixture.WithCreditCardName("Nubank"),
				fixture.WithCreditCardLimit(2000.0),
			),
			mockSetup: func(mockRepo *MockCreditCardRepository, mockTxManager *MockTransactionManager) {
				mockRepo.On("Add", mock.Anything, mock.Anything).Return(domain.CreditCard{}, assert.AnError)

				mockTxManager.On("WithTransaction", mock.Anything).Run(func(args mock.Arguments) {
					fn := args.Get(0).(func(*gorm.DB) error)
					fn(nil)
				}).Return(assert.AnError)
			},
			expectedCreditCard: domain.CreditCard{},
			expectedError:      assert.AnError,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mockRepo := &MockCreditCardRepository{}
			mockTxManager := &MockTransactionManager{}

			tc.mockSetup(mockRepo, mockTxManager)

			useCase := NewCreditCard(mockRepo, mockTxManager)
			result, err := useCase.Add(context.Background(), tc.creditCardInput)

			assert.Equal(t, tc.expectedError, err)
			if tc.expectedError == nil {
				assert.Equal(t, tc.expectedCreditCard.Name, result.Name)
				assert.Equal(t, tc.expectedCreditCard.CreditLimit, result.CreditLimit)
				assert.Equal(t, tc.expectedCreditCard.ClosingDay, result.ClosingDay)
				assert.Equal(t, tc.expectedCreditCard.DueDay, result.DueDay)
			}

			mockRepo.AssertExpectations(t)
			mockTxManager.AssertExpectations(t)
		})
	}
}

func TestCreditCard_FindByID(t *testing.T) {
	tests := map[string]struct {
		creditCardID       uuid.UUID
		mockSetup          func(mockRepo *MockCreditCardRepository)
		expectedCreditCard domain.CreditCard
		expectedError      error
	}{
		"should find credit card by id with success": {
			creditCardID: fixture.CreditCardID,
			mockSetup: func(mockRepo *MockCreditCardRepository) {
				creditCard := fixture.CreditCardMock()
				mockRepo.On("FindByID", fixture.CreditCardID).Return(creditCard, nil)
			},
			expectedCreditCard: fixture.CreditCardMock(),
			expectedError:      nil,
		},
		"should fail when credit card not found": {
			creditCardID: fixture.CreditCardID,
			mockSetup: func(mockRepo *MockCreditCardRepository) {
				mockRepo.On("FindByID", fixture.CreditCardID).Return(domain.CreditCard{}, assert.AnError)
			},
			expectedCreditCard: domain.CreditCard{},
			expectedError:      assert.AnError,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mockRepo := &MockCreditCardRepository{}
			mockTxManager := &MockTransactionManager{}

			tc.mockSetup(mockRepo)

			useCase := NewCreditCard(mockRepo, mockTxManager)
			result, err := useCase.FindByID(context.Background(), tc.creditCardID)

			if tc.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.Equal(t, tc.expectedError, err)
				assert.Equal(t, tc.expectedCreditCard.Name, result.Name)
				assert.Equal(t, tc.expectedCreditCard.CreditLimit, result.CreditLimit)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestCreditCard_FindAll(t *testing.T) {
	tests := map[string]struct {
		mockSetup           func(mockRepo *MockCreditCardRepository)
		expectedCreditCards []domain.CreditCard
		expectedError       error
	}{
		"should find all credit cards with success": {
			mockSetup: func(mockRepo *MockCreditCardRepository) {
				creditCards := []domain.CreditCard{
					fixture.CreditCardMock(fixture.WithCreditCardName("Nubank")),
					fixture.CreditCardMock(fixture.WithCreditCardName("Inter")),
				}
				mockRepo.On("FindAll").Return(creditCards, nil)
			},
			expectedCreditCards: []domain.CreditCard{
				fixture.CreditCardMock(fixture.WithCreditCardName("Nubank")),
				fixture.CreditCardMock(fixture.WithCreditCardName("Inter")),
			},
			expectedError: nil,
		},
		"should return empty list when no credit cards found": {
			mockSetup: func(mockRepo *MockCreditCardRepository) {
				mockRepo.On("FindAll").Return([]domain.CreditCard{}, nil)
			},
			expectedCreditCards: []domain.CreditCard{},
			expectedError:       nil,
		},
		"should fail when repo.FindAll returns error": {
			mockSetup: func(mockRepo *MockCreditCardRepository) {
				mockRepo.On("FindAll").Return([]domain.CreditCard{}, assert.AnError)
			},
			expectedCreditCards: []domain.CreditCard{},
			expectedError:       assert.AnError,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mockRepo := &MockCreditCardRepository{}
			mockTxManager := &MockTransactionManager{}

			tc.mockSetup(mockRepo)

			useCase := NewCreditCard(mockRepo, mockTxManager)
			result, err := useCase.FindAll(context.Background())

			if tc.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.Equal(t, tc.expectedError, err)
				assert.Equal(t, len(tc.expectedCreditCards), len(result))
				for i, expectedCard := range tc.expectedCreditCards {
					assert.Equal(t, expectedCard.Name, result[i].Name)
				}
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestCreditCard_Update(t *testing.T) {
	tests := map[string]struct {
		creditCardID       uuid.UUID
		creditCardInput    domain.CreditCard
		mockSetup          func(mockRepo *MockCreditCardRepository, mockTxManager *MockTransactionManager)
		expectedCreditCard domain.CreditCard
		expectedError      error
	}{
		"should update credit card with success": {
			creditCardID: fixture.CreditCardID,
			creditCardInput: fixture.CreditCardMock(
				fixture.WithCreditCardName("Nubank Updated"),
			),
			mockSetup: func(mockRepo *MockCreditCardRepository, mockTxManager *MockTransactionManager) {
				updatedCreditCard := fixture.CreditCardMock(
					fixture.WithCreditCardName("Nubank Updated"),
				)

				mockTxManager.On("WithTransaction", mock.Anything, mock.Anything).Return(nil)

				mockRepo.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(updatedCreditCard, nil)
			},
			expectedCreditCard: fixture.CreditCardMock(
				fixture.WithCreditCardName("Nubank Updated"),
			),
			expectedError: nil,
		},
		"should fail with invalid closing day": {
			creditCardID: fixture.CreditCardID,
			creditCardInput: fixture.CreditCardMock(
				fixture.WithCreditCardClosingDay(40),
			),
			mockSetup: func(mockRepo *MockCreditCardRepository, mockTxManager *MockTransactionManager) {
			},
			expectedCreditCard: domain.CreditCard{},
			expectedError:      ErrInvalidClosingDay,
		},
		"should fail when repo.Update returns error": {
			creditCardID: fixture.CreditCardID,
			creditCardInput: fixture.CreditCardMock(
				fixture.WithCreditCardName("Nubank Updated"),
			),
			mockSetup: func(mockRepo *MockCreditCardRepository, mockTxManager *MockTransactionManager) {
				mockRepo.On("Update", mock.Anything, fixture.CreditCardID, mock.Anything).Return(domain.CreditCard{}, assert.AnError)

				mockTxManager.On("WithTransaction", mock.Anything).Run(func(args mock.Arguments) {
					fn := args.Get(0).(func(*gorm.DB) error)
					fn(nil)
				}).Return(assert.AnError)
			},
			expectedCreditCard: domain.CreditCard{},
			expectedError:      assert.AnError,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mockRepo := &MockCreditCardRepository{}
			mockTxManager := &MockTransactionManager{}

			tc.mockSetup(mockRepo, mockTxManager)

			useCase := NewCreditCard(mockRepo, mockTxManager)
			result, err := useCase.Update(context.Background(), tc.creditCardID, tc.creditCardInput)

			assert.Equal(t, tc.expectedError, err)
			if tc.expectedError == nil {
				assert.Equal(t, tc.expectedCreditCard.Name, result.Name)
			}

			mockRepo.AssertExpectations(t)
			mockTxManager.AssertExpectations(t)
		})
	}
}

func TestCreditCard_Delete(t *testing.T) {
	tests := map[string]struct {
		creditCardID  uuid.UUID
		mockSetup     func(mockRepo *MockCreditCardRepository, mockTxManager *MockTransactionManager)
		expectedError error
	}{
		"should delete credit card with success": {
			creditCardID: fixture.CreditCardID,
			mockSetup: func(mockRepo *MockCreditCardRepository, mockTxManager *MockTransactionManager) {
				mockTxManager.On("WithTransaction", mock.Anything, mock.Anything).Return(nil)

				mockRepo.On("Delete", mock.Anything, mock.Anything).Return(nil)
			},
			expectedError: nil,
		},
		"should fail when repo.Delete returns error": {
			creditCardID: fixture.CreditCardID,
			mockSetup: func(mockRepo *MockCreditCardRepository, mockTxManager *MockTransactionManager) {
				mockRepo.On("Delete", mock.Anything, fixture.CreditCardID).Return(assert.AnError)

				mockTxManager.On("WithTransaction", mock.Anything).Run(func(args mock.Arguments) {
					fn := args.Get(0).(func(*gorm.DB) error)
					fn(nil)
				}).Return(assert.AnError)
			},
			expectedError: assert.AnError,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mockRepo := &MockCreditCardRepository{}
			mockTxManager := &MockTransactionManager{}

			tc.mockSetup(mockRepo, mockTxManager)

			useCase := NewCreditCard(mockRepo, mockTxManager)
			err := useCase.Delete(context.Background(), tc.creditCardID)

			if tc.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.Equal(t, tc.expectedError, err)
			}

			mockRepo.AssertExpectations(t)
			mockTxManager.AssertExpectations(t)
		})
	}
}
