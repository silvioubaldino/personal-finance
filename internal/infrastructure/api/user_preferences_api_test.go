package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/usecase"
	"personal-finance/pkg/log"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupUserPreferencesRouter() *gin.Engine {
	log.Initialize()
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestUserPreferencesHandler_Get(t *testing.T) {
	now := time.Now()

	tests := map[string]struct {
		mockSetup      func(mock *MockUserPreferencesUseCase)
		expectedStatus int
		expectedBody   string
	}{
		"should get preferences successfully": {
			mockSetup: func(mockUC *MockUserPreferencesUseCase) {
				mockUC.On("Get", mock.Anything).Return(domain.UserPreferences{
					UserID:     "user-123",
					Language:   "pt-BR",
					Currency:   "BRL",
					DateCreate: now,
					DateUpdate: now,
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"language":"pt-BR","currency":"BRL"}`,
		},
		"should return defaults when no preferences exist": {
			mockSetup: func(mockUC *MockUserPreferencesUseCase) {
				mockUC.On("Get", mock.Anything).Return(domain.UserPreferences{
					UserID:     "user-123",
					Language:   domain.DefaultLanguage,
					Currency:   domain.DefaultCurrency,
					DateCreate: now,
					DateUpdate: now,
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"language":"pt-BR","currency":"BRL"}`,
		},
		"should return error when usecase fails": {
			mockSetup: func(mockUC *MockUserPreferencesUseCase) {
				mockUC.On("Get", mock.Anything).
					Return(domain.UserPreferences{}, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":{"code":500,"message":"Internal server error"}}`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			router := setupUserPreferencesRouter()
			mockUseCase := new(MockUserPreferencesUseCase)
			if tt.mockSetup != nil {
				tt.mockSetup(mockUseCase)
			}

			NewUserPreferencesHandlers(router, mockUseCase)

			req := httptest.NewRequest(http.MethodGet, "/me/preferences", nil)
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)
			assert.Equal(t, tt.expectedBody, resp.Body.String())
			mockUseCase.AssertExpectations(t)
		})
	}
}

func TestUserPreferencesHandler_Update(t *testing.T) {
	now := time.Now()

	tests := map[string]struct {
		requestBody    interface{}
		mockSetup      func(mock *MockUserPreferencesUseCase)
		expectedStatus int
		expectedBody   string
	}{
		"should update preferences successfully": {
			requestBody: UserPreferencesRequest{
				Language: "en-US",
				Currency: "USD",
			},
			mockSetup: func(mockUC *MockUserPreferencesUseCase) {
				expectedInput := usecase.UserPreferencesInput{
					Language: "en-US",
					Currency: "USD",
				}
				mockUC.On("Update", mock.Anything, expectedInput).Return(domain.UserPreferences{
					UserID:     "user-123",
					Language:   "en-US",
					Currency:   "USD",
					DateCreate: now,
					DateUpdate: now,
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"language":"en-US","currency":"USD"}`,
		},
		"should return error when language is missing": {
			requestBody: map[string]string{
				"currency": "USD",
			},
			mockSetup:      func(mockUC *MockUserPreferencesUseCase) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":{"code":400,"message":"Invalid data provided"}}`,
		},
		"should update preferences without currency": {
			requestBody: map[string]string{
				"language": "en-US",
			},
			mockSetup: func(mockUC *MockUserPreferencesUseCase) {
				expectedInput := usecase.UserPreferencesInput{
					Language: "en-US",
					Currency: "",
				}
				mockUC.On("Update", mock.Anything, expectedInput).Return(domain.UserPreferences{
					UserID:     "user-123",
					Language:   "en-US",
					Currency:   "BRL",
					DateCreate: now,
					DateUpdate: now,
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"language":"en-US","currency":"BRL"}`,
		},
		"should return error when body is invalid json": {
			requestBody:    "invalid",
			mockSetup:      func(mockUC *MockUserPreferencesUseCase) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":{"code":400,"message":"Invalid data provided"}}`,
		},
		"should return error when usecase fails with validation error": {
			requestBody: UserPreferencesRequest{
				Language: "invalid",
				Currency: "USD",
			},
			mockSetup: func(mockUC *MockUserPreferencesUseCase) {
				expectedInput := usecase.UserPreferencesInput{
					Language: "invalid",
					Currency: "USD",
				}
				mockUC.On("Update", mock.Anything, expectedInput).
					Return(domain.UserPreferences{}, domain.WrapInvalidInput(errors.New("invalid language"), "language must be in BCP47 format"))
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":{"code":400,"message":"Invalid data provided"}}`,
		},
		"should return error when usecase fails with database error": {
			requestBody: UserPreferencesRequest{
				Language: "pt-BR",
				Currency: "BRL",
			},
			mockSetup: func(mockUC *MockUserPreferencesUseCase) {
				expectedInput := usecase.UserPreferencesInput{
					Language: "pt-BR",
					Currency: "BRL",
				}
				mockUC.On("Update", mock.Anything, expectedInput).
					Return(domain.UserPreferences{}, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":{"code":500,"message":"Internal server error"}}`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			router := setupUserPreferencesRouter()
			mockUseCase := new(MockUserPreferencesUseCase)
			if tt.mockSetup != nil {
				tt.mockSetup(mockUseCase)
			}

			NewUserPreferencesHandlers(router, mockUseCase)

			var body []byte
			var err error
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				assert.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPut, "/me/preferences", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)
			assert.Equal(t, tt.expectedBody, resp.Body.String())
			mockUseCase.AssertExpectations(t)
		})
	}
}
