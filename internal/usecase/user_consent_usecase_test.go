package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/plataform/authentication"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockUserConsentRepository struct {
	mock.Mock
}

func (m *MockUserConsentRepository) Save(ctx context.Context, consent domain.UserConsent) (domain.UserConsent, error) {
	args := m.Called(mock.Anything, consent)
	return args.Get(0).(domain.UserConsent), args.Error(1)
}

func (m *MockUserConsentRepository) FindByUserID(ctx context.Context) ([]domain.UserConsent, error) {
	args := m.Called()
	return args.Get(0).([]domain.UserConsent), args.Error(1)
}

func (m *MockUserConsentRepository) FindLatestByTermVersion(ctx context.Context, termVersion string) (domain.UserConsent, error) {
	args := m.Called(termVersion)
	return args.Get(0).(domain.UserConsent), args.Error(1)
}

func (m *MockUserConsentRepository) HasConsentedToVersion(ctx context.Context, termVersion string) (bool, error) {
	args := m.Called(termVersion)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserConsentRepository) FindByID(ctx context.Context, id uuid.UUID) (domain.UserConsent, error) {
	args := m.Called(id)
	return args.Get(0).(domain.UserConsent), args.Error(1)
}

func TestUserConsent_RecordConsent(t *testing.T) {
	testCases := []struct {
		name            string
		input           UserConsentInput
		mockSetup       func(mock *MockUserConsentRepository)
		expectedConsent domain.UserConsent
		expectError     bool
	}{
		{
			name: "should record consent successfully",
			input: UserConsentInput{
				TermVersion: "v1.0",
				IPAddress:   "192.168.1.1",
				UserAgent:   "Mozilla/5.0",
			},
			mockSetup: func(mockRepo *MockUserConsentRepository) {
				mockRepo.On("Save", mock.Anything, mock.MatchedBy(func(c domain.UserConsent) bool {
					return c.TermVersion == "v1.0" && c.UserID == "test-user"
				})).Return(domain.UserConsent{
					ID:          uuid.MustParse("12345678-1234-1234-1234-123456789012"),
					UserID:      "test-user",
					TermVersion: "v1.0",
					AgreedAt:    time.Now(),
					IPAddress:   "192.168.1.1",
					UserAgent:   "Mozilla/5.0",
				}, nil)
			},
			expectedConsent: domain.UserConsent{
				ID:          uuid.MustParse("12345678-1234-1234-1234-123456789012"),
				UserID:      "test-user",
				TermVersion: "v1.0",
				IPAddress:   "192.168.1.1",
				UserAgent:   "Mozilla/5.0",
			},
			expectError: false,
		},
		{
			name: "should fail when term_version is empty",
			input: UserConsentInput{
				TermVersion: "",
			},
			mockSetup:       func(mockRepo *MockUserConsentRepository) {},
			expectedConsent: domain.UserConsent{},
			expectError:     true,
		},
		{
			name: "should fail when repository returns error",
			input: UserConsentInput{
				TermVersion: "v1.0",
			},
			mockSetup: func(mockRepo *MockUserConsentRepository) {
				mockRepo.On("Save", mock.Anything, mock.Anything).
					Return(domain.UserConsent{}, errors.New("database error"))
			},
			expectedConsent: domain.UserConsent{},
			expectError:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := new(MockUserConsentRepository)
			tc.mockSetup(mockRepo)

			ctx := context.WithValue(context.Background(), authentication.UserID, "test-user")

			uc := NewUserConsent(mockRepo)

			result, err := uc.RecordConsent(ctx, tc.input)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedConsent.TermVersion, result.TermVersion)
				assert.Equal(t, tc.expectedConsent.UserID, result.UserID)
			}
		})
	}
}

func TestUserConsent_GetAllConsents(t *testing.T) {
	testCases := []struct {
		name             string
		mockSetup        func(mock *MockUserConsentRepository)
		expectedConsents []domain.UserConsent
		expectError      bool
	}{
		{
			name: "should return all consents",
			mockSetup: func(mockRepo *MockUserConsentRepository) {
				mockRepo.On("FindByUserID").Return([]domain.UserConsent{
					{
						ID:          uuid.MustParse("12345678-1234-1234-1234-123456789012"),
						UserID:      "test-user",
						TermVersion: "v1.0",
					},
					{
						ID:          uuid.MustParse("22345678-1234-1234-1234-123456789012"),
						UserID:      "test-user",
						TermVersion: "v2.0",
					},
				}, nil)
			},
			expectedConsents: []domain.UserConsent{
				{TermVersion: "v1.0"},
				{TermVersion: "v2.0"},
			},
			expectError: false,
		},
		{
			name: "should return empty list when no consents",
			mockSetup: func(mockRepo *MockUserConsentRepository) {
				mockRepo.On("FindByUserID").Return([]domain.UserConsent{}, nil)
			},
			expectedConsents: []domain.UserConsent{},
			expectError:      false,
		},
		{
			name: "should fail when repository returns error",
			mockSetup: func(mockRepo *MockUserConsentRepository) {
				mockRepo.On("FindByUserID").Return([]domain.UserConsent(nil), errors.New("database error"))
			},
			expectedConsents: nil,
			expectError:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := new(MockUserConsentRepository)
			tc.mockSetup(mockRepo)

			ctx := context.WithValue(context.Background(), authentication.UserID, "test-user")

			uc := NewUserConsent(mockRepo)

			result, err := uc.GetAllConsents(ctx)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, result, len(tc.expectedConsents))
			}
		})
	}
}

func TestUserConsent_HasConsentedToVersion(t *testing.T) {
	testCases := []struct {
		name           string
		termVersion    string
		mockSetup      func(mock *MockUserConsentRepository)
		expectedResult bool
		expectError    bool
	}{
		{
			name:        "should return true when user has consented",
			termVersion: "v1.0",
			mockSetup: func(mockRepo *MockUserConsentRepository) {
				mockRepo.On("HasConsentedToVersion", "v1.0").Return(true, nil)
			},
			expectedResult: true,
			expectError:    false,
		},
		{
			name:        "should return false when user has not consented",
			termVersion: "v2.0",
			mockSetup: func(mockRepo *MockUserConsentRepository) {
				mockRepo.On("HasConsentedToVersion", "v2.0").Return(false, nil)
			},
			expectedResult: false,
			expectError:    false,
		},
		{
			name:           "should fail when term_version is empty",
			termVersion:    "",
			mockSetup:      func(mockRepo *MockUserConsentRepository) {},
			expectedResult: false,
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := new(MockUserConsentRepository)
			tc.mockSetup(mockRepo)

			ctx := context.WithValue(context.Background(), authentication.UserID, "test-user")

			uc := NewUserConsent(mockRepo)

			result, err := uc.HasConsentedToVersion(ctx, tc.termVersion)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedResult, result)
			}
		})
	}
}
