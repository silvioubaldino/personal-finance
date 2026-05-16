package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"personal-finance/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Get(ctx context.Context) (domain.User, error) {
	args := m.Called()
	return args.Get(0).(domain.User), args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, user domain.User) (domain.User, error) {
	args := m.Called(user)
	return args.Get(0).(domain.User), args.Error(1)
}

func TestUser_Get(t *testing.T) {
	now := time.Now()

	tests := map[string]struct {
		mockSetup    func(mock *MockUserRepository)
		expectedUser domain.User
		expectedErr  error
	}{
		"should get user successfully": {
			mockSetup: func(mockRepo *MockUserRepository) {
				mockRepo.On("Get").Return(domain.User{
					ID:        "user-123",
					Language:  "pt-BR",
					Currency:  "BRL",
					CreatedAt: now,
					UpdatedAt: now,
				}, nil)
			},
			expectedUser: domain.User{
				ID:        "user-123",
				Language:  "pt-BR",
				Currency:  "BRL",
				CreatedAt: now,
				UpdatedAt: now,
			},
			expectedErr: nil,
		},
		"should return error when repository fails": {
			mockSetup: func(mockRepo *MockUserRepository) {
				mockRepo.On("Get").Return(domain.User{}, errors.New("database error"))
			},
			expectedUser: domain.User{},
			expectedErr:  errors.New("database error"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockRepo := new(MockUserRepository)
			if tt.mockSetup != nil {
				tt.mockSetup(mockRepo)
			}

			uc := NewUser(mockRepo)
			ctx := context.Background()

			result, err := uc.Get(ctx)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedUser, result)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestUser_Update(t *testing.T) {
	now := time.Now()

	tests := map[string]struct {
		input        UserInput
		mockSetup    func(mock *MockUserRepository)
		expectedUser domain.User
		expectedErr  bool
		errContains  string
	}{
		"should update successfully": {
			input: UserInput{Language: "en-US", Currency: "USD"},
			mockSetup: func(mockRepo *MockUserRepository) {
				mockRepo.On("Update", domain.User{
					Language: "en-US",
					Currency: "USD",
				}).Return(domain.User{
					ID:        "user-123",
					Language:  "en-US",
					Currency:  "USD",
					CreatedAt: now,
					UpdatedAt: now,
				}, nil)
			},
			expectedUser: domain.User{
				ID:        "user-123",
				Language:  "en-US",
				Currency:  "USD",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		"should normalize currency to uppercase": {
			input: UserInput{Language: "pt-BR", Currency: "brl"},
			mockSetup: func(mockRepo *MockUserRepository) {
				mockRepo.On("Update", domain.User{
					Language: "pt-BR",
					Currency: "BRL",
				}).Return(domain.User{
					ID:        "user-123",
					Language:  "pt-BR",
					Currency:  "BRL",
					CreatedAt: now,
					UpdatedAt: now,
				}, nil)
			},
			expectedUser: domain.User{
				ID:        "user-123",
				Language:  "pt-BR",
				Currency:  "BRL",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		"should update with only currency": {
			input: UserInput{Currency: "USD"},
			mockSetup: func(mockRepo *MockUserRepository) {
				mockRepo.On("Update", domain.User{
					Language: "",
					Currency: "USD",
				}).Return(domain.User{
					ID:        "user-123",
					Language:  "pt-BR",
					Currency:  "USD",
					CreatedAt: now,
					UpdatedAt: now,
				}, nil)
			},
			expectedUser: domain.User{
				ID:        "user-123",
				Language:  "pt-BR",
				Currency:  "USD",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		"should update with only language": {
			input: UserInput{Language: "pt-BR"},
			mockSetup: func(mockRepo *MockUserRepository) {
				mockRepo.On("Update", domain.User{
					Language: "pt-BR",
					Currency: "",
				}).Return(domain.User{
					ID:        "user-123",
					Language:  "pt-BR",
					Currency:  "BRL",
					CreatedAt: now,
					UpdatedAt: now,
				}, nil)
			},
			expectedUser: domain.User{
				ID:        "user-123",
				Language:  "pt-BR",
				Currency:  "BRL",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		"should error on invalid language": {
			input:       UserInput{Language: "invalid-language-format", Currency: "BRL"},
			mockSetup:   func(mockRepo *MockUserRepository) {},
			expectedErr: true,
			errContains: "BCP47 format",
		},
		"should error on invalid currency": {
			input:       UserInput{Language: "pt-BR", Currency: "INVALID"},
			mockSetup:   func(mockRepo *MockUserRepository) {},
			expectedErr: true,
			errContains: "ISO 4217",
		},
		"should accept language without region": {
			input: UserInput{Language: "pt", Currency: "BRL"},
			mockSetup: func(mockRepo *MockUserRepository) {
				mockRepo.On("Update", domain.User{
					Language: "pt",
					Currency: "BRL",
				}).Return(domain.User{
					ID:        "user-123",
					Language:  "pt",
					Currency:  "BRL",
					CreatedAt: now,
					UpdatedAt: now,
				}, nil)
			},
			expectedUser: domain.User{
				ID:        "user-123",
				Language:  "pt",
				Currency:  "BRL",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		"should accept three letter language code": {
			input: UserInput{Language: "por", Currency: "BRL"},
			mockSetup: func(mockRepo *MockUserRepository) {
				mockRepo.On("Update", domain.User{
					Language: "por",
					Currency: "BRL",
				}).Return(domain.User{
					ID:        "user-123",
					Language:  "por",
					Currency:  "BRL",
					CreatedAt: now,
					UpdatedAt: now,
				}, nil)
			},
			expectedUser: domain.User{
				ID:        "user-123",
				Language:  "por",
				Currency:  "BRL",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		"should return error when repository fails": {
			input: UserInput{Language: "pt-BR", Currency: "BRL"},
			mockSetup: func(mockRepo *MockUserRepository) {
				mockRepo.On("Update", domain.User{
					Language: "pt-BR",
					Currency: "BRL",
				}).Return(domain.User{}, errors.New("database error"))
			},
			expectedErr: true,
			errContains: "database error",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockRepo := new(MockUserRepository)
			if tt.mockSetup != nil {
				tt.mockSetup(mockRepo)
			}

			uc := NewUser(mockRepo)
			ctx := context.Background()

			result, err := uc.Update(ctx, tt.input)

			if tt.expectedErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedUser, result)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}
