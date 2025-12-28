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

type MockUserPreferencesRepository struct {
	mock.Mock
}

func (m *MockUserPreferencesRepository) GetOrCreateDefaults(ctx context.Context) (domain.UserPreferences, error) {
	args := m.Called()
	return args.Get(0).(domain.UserPreferences), args.Error(1)
}

func (m *MockUserPreferencesRepository) Upsert(ctx context.Context, prefs domain.UserPreferences) (domain.UserPreferences, error) {
	args := m.Called(prefs)
	return args.Get(0).(domain.UserPreferences), args.Error(1)
}

func TestUserPreferences_Get(t *testing.T) {
	now := time.Now()

	tests := map[string]struct {
		mockSetup     func(mock *MockUserPreferencesRepository)
		expectedPrefs domain.UserPreferences
		expectedErr   error
	}{
		"should get preferences successfully": {
			mockSetup: func(mockRepo *MockUserPreferencesRepository) {
				mockRepo.On("GetOrCreateDefaults").Return(domain.UserPreferences{
					UserID:     "user-123",
					Language:   "pt-BR",
					Currency:   "BRL",
					DateCreate: now,
					DateUpdate: now,
				}, nil)
			},
			expectedPrefs: domain.UserPreferences{
				UserID:     "user-123",
				Language:   "pt-BR",
				Currency:   "BRL",
				DateCreate: now,
				DateUpdate: now,
			},
			expectedErr: nil,
		},
		"should return error when repository fails": {
			mockSetup: func(mockRepo *MockUserPreferencesRepository) {
				mockRepo.On("GetOrCreateDefaults").
					Return(domain.UserPreferences{}, errors.New("database error"))
			},
			expectedPrefs: domain.UserPreferences{},
			expectedErr:   errors.New("database error"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockRepo := new(MockUserPreferencesRepository)
			if tt.mockSetup != nil {
				tt.mockSetup(mockRepo)
			}

			uc := NewUserPreferences(mockRepo)
			ctx := context.Background()

			result, err := uc.Get(ctx)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedPrefs, result)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestUserPreferences_Update(t *testing.T) {
	now := time.Now()

	tests := map[string]struct {
		input         UserPreferencesInput
		mockSetup     func(mock *MockUserPreferencesRepository)
		expectedPrefs domain.UserPreferences
		expectedErr   bool
		errContains   string
	}{
		"should update preferences successfully": {
			input: UserPreferencesInput{
				Language: "en-US",
				Currency: "USD",
			},
			mockSetup: func(mockRepo *MockUserPreferencesRepository) {
				mockRepo.On("Upsert", domain.UserPreferences{
					Language: "en-US",
					Currency: "USD",
				}).Return(domain.UserPreferences{
					UserID:     "user-123",
					Language:   "en-US",
					Currency:   "USD",
					DateCreate: now,
					DateUpdate: now,
				}, nil)
			},
			expectedPrefs: domain.UserPreferences{
				UserID:     "user-123",
				Language:   "en-US",
				Currency:   "USD",
				DateCreate: now,
				DateUpdate: now,
			},
			expectedErr: false,
		},
		"should normalize currency to uppercase": {
			input: UserPreferencesInput{
				Language: "pt-BR",
				Currency: "brl",
			},
			mockSetup: func(mockRepo *MockUserPreferencesRepository) {
				mockRepo.On("Upsert", domain.UserPreferences{
					Language: "pt-BR",
					Currency: "BRL",
				}).Return(domain.UserPreferences{
					UserID:     "user-123",
					Language:   "pt-BR",
					Currency:   "BRL",
					DateCreate: now,
					DateUpdate: now,
				}, nil)
			},
			expectedPrefs: domain.UserPreferences{
				UserID:     "user-123",
				Language:   "pt-BR",
				Currency:   "BRL",
				DateCreate: now,
				DateUpdate: now,
			},
			expectedErr: false,
		},
		"should return error when language is empty": {
			input: UserPreferencesInput{
				Language: "",
				Currency: "BRL",
			},
			mockSetup:   func(mockRepo *MockUserPreferencesRepository) {},
			expectedErr: true,
			errContains: "language is required",
		},
		"should update with empty currency (optional)": {
			input: UserPreferencesInput{
				Language: "pt-BR",
				Currency: "",
			},
			mockSetup: func(mockRepo *MockUserPreferencesRepository) {
				mockRepo.On("Upsert", domain.UserPreferences{
					Language: "pt-BR",
					Currency: "",
				}).Return(domain.UserPreferences{
					UserID:     "user-123",
					Language:   "pt-BR",
					Currency:   "BRL",
					DateCreate: now,
					DateUpdate: now,
				}, nil)
			},
			expectedPrefs: domain.UserPreferences{
				UserID:     "user-123",
				Language:   "pt-BR",
				Currency:   "BRL",
				DateCreate: now,
				DateUpdate: now,
			},
			expectedErr: false,
		},
		"should return error when language format is invalid": {
			input: UserPreferencesInput{
				Language: "invalid-language-format",
				Currency: "BRL",
			},
			mockSetup:   func(mockRepo *MockUserPreferencesRepository) {},
			expectedErr: true,
			errContains: "BCP47 format",
		},
		"should return error when currency format is invalid": {
			input: UserPreferencesInput{
				Language: "pt-BR",
				Currency: "INVALID",
			},
			mockSetup:   func(mockRepo *MockUserPreferencesRepository) {},
			expectedErr: true,
			errContains: "ISO 4217",
		},
		"should accept language without region": {
			input: UserPreferencesInput{
				Language: "pt",
				Currency: "BRL",
			},
			mockSetup: func(mockRepo *MockUserPreferencesRepository) {
				mockRepo.On("Upsert", domain.UserPreferences{
					Language: "pt",
					Currency: "BRL",
				}).Return(domain.UserPreferences{
					UserID:     "user-123",
					Language:   "pt",
					Currency:   "BRL",
					DateCreate: now,
					DateUpdate: now,
				}, nil)
			},
			expectedPrefs: domain.UserPreferences{
				UserID:     "user-123",
				Language:   "pt",
				Currency:   "BRL",
				DateCreate: now,
				DateUpdate: now,
			},
			expectedErr: false,
		},
		"should accept three letter language code": {
			input: UserPreferencesInput{
				Language: "por",
				Currency: "BRL",
			},
			mockSetup: func(mockRepo *MockUserPreferencesRepository) {
				mockRepo.On("Upsert", domain.UserPreferences{
					Language: "por",
					Currency: "BRL",
				}).Return(domain.UserPreferences{
					UserID:     "user-123",
					Language:   "por",
					Currency:   "BRL",
					DateCreate: now,
					DateUpdate: now,
				}, nil)
			},
			expectedPrefs: domain.UserPreferences{
				UserID:     "user-123",
				Language:   "por",
				Currency:   "BRL",
				DateCreate: now,
				DateUpdate: now,
			},
			expectedErr: false,
		},
		"should return error when repository fails": {
			input: UserPreferencesInput{
				Language: "pt-BR",
				Currency: "BRL",
			},
			mockSetup: func(mockRepo *MockUserPreferencesRepository) {
				mockRepo.On("Upsert", domain.UserPreferences{
					Language: "pt-BR",
					Currency: "BRL",
				}).Return(domain.UserPreferences{}, errors.New("database error"))
			},
			expectedErr: true,
			errContains: "database error",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockRepo := new(MockUserPreferencesRepository)
			if tt.mockSetup != nil {
				tt.mockSetup(mockRepo)
			}

			uc := NewUserPreferences(mockRepo)
			ctx := context.Background()

			result, err := uc.Update(ctx, tt.input)

			if tt.expectedErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedPrefs, result)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}
