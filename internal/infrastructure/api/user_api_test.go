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

func setupUserRouter() *gin.Engine {
	log.Initialize()
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestUserHandler_Get(t *testing.T) {
	now := time.Now()

	tests := map[string]struct {
		mockSetup      func(mock *MockUserUseCase)
		expectedStatus int
		expectedBody   string
	}{
		"should get preferences successfully": {
			mockSetup: func(mockUC *MockUserUseCase) {
				mockUC.On("Get", mock.Anything).Return(domain.User{
					ID:        "user-123",
					Language:  "pt-BR",
					Currency:  "BRL",
					CreatedAt: now,
					UpdatedAt: now,
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"language":"pt-BR","currency":"BRL"}`,
		},
		"should return defaults when no preferences exist": {
			mockSetup: func(mockUC *MockUserUseCase) {
				mockUC.On("Get", mock.Anything).Return(domain.User{
					ID:        "user-123",
					Language:  domain.DefaultLanguage,
					Currency:  domain.DefaultCurrency,
					CreatedAt: now,
					UpdatedAt: now,
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"language":"pt-BR","currency":"BRL"}`,
		},
		"should return error when usecase fails": {
			mockSetup: func(mockUC *MockUserUseCase) {
				mockUC.On("Get", mock.Anything).
					Return(domain.User{}, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":{"code":500,"message":"Internal server error"}}`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			router := setupUserRouter()
			mockUseCase := new(MockUserUseCase)
			if tt.mockSetup != nil {
				tt.mockSetup(mockUseCase)
			}

			NewUserHandlers(router, mockUseCase)

			req := httptest.NewRequest(http.MethodGet, "/me/preferences", nil)
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)
			assert.Equal(t, tt.expectedBody, resp.Body.String())
			mockUseCase.AssertExpectations(t)
		})
	}
}

func TestUserHandler_Update(t *testing.T) {
	now := time.Now()

	tests := map[string]struct {
		requestBody    interface{}
		mockSetup      func(mock *MockUserUseCase)
		expectedStatus int
		expectedBody   string
	}{
		"should update preferences successfully": {
			requestBody: UserPreferencesRequest{
				Language: "en-US",
				Currency: "USD",
			},
			mockSetup: func(mockUC *MockUserUseCase) {
				expectedInput := usecase.UserInput{
					Language: "en-US",
					Currency: "USD",
				}
				mockUC.On("Update", mock.Anything, expectedInput).Return(domain.User{
					ID:        "user-123",
					Language:  "en-US",
					Currency:  "USD",
					CreatedAt: now,
					UpdatedAt: now,
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"language":"en-US","currency":"USD"}`,
		},
		"should update with only currency": {
			requestBody: map[string]string{
				"currency": "USD",
			},
			mockSetup: func(mockUC *MockUserUseCase) {
				expectedInput := usecase.UserInput{
					Language: "",
					Currency: "USD",
				}
				mockUC.On("Update", mock.Anything, expectedInput).Return(domain.User{
					ID:        "user-123",
					Language:  "pt-BR",
					Currency:  "USD",
					CreatedAt: now,
					UpdatedAt: now,
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"language":"pt-BR","currency":"USD"}`,
		},
		"should return error when no fields provided": {
			requestBody:    map[string]string{},
			mockSetup:      func(mockUC *MockUserUseCase) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":{"code":400,"message":"Invalid data provided"}}`,
		},
		"should update with only language": {
			requestBody: map[string]string{
				"language": "en-US",
			},
			mockSetup: func(mockUC *MockUserUseCase) {
				expectedInput := usecase.UserInput{
					Language: "en-US",
					Currency: "",
				}
				mockUC.On("Update", mock.Anything, expectedInput).Return(domain.User{
					ID:        "user-123",
					Language:  "en-US",
					Currency:  "BRL",
					CreatedAt: now,
					UpdatedAt: now,
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"language":"en-US","currency":"BRL"}`,
		},
		"should return error when body is invalid json": {
			requestBody:    "invalid",
			mockSetup:      func(mockUC *MockUserUseCase) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":{"code":400,"message":"Invalid data provided"}}`,
		},
		"should return error when usecase fails with validation error": {
			requestBody: UserPreferencesRequest{
				Language: "invalid",
				Currency: "USD",
			},
			mockSetup: func(mockUC *MockUserUseCase) {
				expectedInput := usecase.UserInput{
					Language: "invalid",
					Currency: "USD",
				}
				mockUC.On("Update", mock.Anything, expectedInput).
					Return(domain.User{}, domain.WrapInvalidInput(errors.New("invalid language"), "language must be in BCP47 format"))
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":{"code":400,"message":"Invalid data provided"}}`,
		},
		"should return error when usecase fails with database error": {
			requestBody: UserPreferencesRequest{
				Language: "pt-BR",
				Currency: "BRL",
			},
			mockSetup: func(mockUC *MockUserUseCase) {
				expectedInput := usecase.UserInput{
					Language: "pt-BR",
					Currency: "BRL",
				}
				mockUC.On("Update", mock.Anything, expectedInput).
					Return(domain.User{}, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":{"code":500,"message":"Internal server error"}}`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			router := setupUserRouter()
			mockUseCase := new(MockUserUseCase)
			if tt.mockSetup != nil {
				tt.mockSetup(mockUseCase)
			}

			NewUserHandlers(router, mockUseCase)

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
