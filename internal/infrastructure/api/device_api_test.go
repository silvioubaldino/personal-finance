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
	"personal-finance/internal/infrastructure/repository"
	"personal-finance/internal/usecase"
	"personal-finance/pkg/log"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupDeviceRouter() *gin.Engine {
	log.Initialize()
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestDeviceHandler_Upsert(t *testing.T) {
	fixedTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	fixedID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	tests := map[string]struct {
		requestBody    interface{}
		mockSetup      func(m *MockDeviceUseCase)
		expectedStatus int
		expectedBody   string
	}{
		"should upsert device successfully": {
			requestBody: DeviceRequest{
				ExpoPushToken: "ExponentPushToken[abc123]",
				Platform:      "ios",
			},
			mockSetup: func(m *MockDeviceUseCase) {
				expectedInput := usecase.DeviceInput{
					ExpoPushToken: "ExponentPushToken[abc123]",
					Platform:      "ios",
				}
				m.On("Upsert", mock.Anything, expectedInput).Return(domain.Device{
					ID:            fixedID,
					UserID:        "user-123",
					ExpoPushToken: "ExponentPushToken[abc123]",
					Platform:      domain.PlatformIOS,
					DateCreate:    fixedTime,
					DateUpdate:    fixedTime,
					LastSeenAt:    &fixedTime,
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"id":"11111111-1111-1111-1111-111111111111","expo_push_token":"ExponentPushToken[abc123]","platform":"ios","date_create":"2024-01-15T10:00:00Z","date_update":"2024-01-15T10:00:00Z","last_seen_at":"2024-01-15T10:00:00Z"}`,
		},
		"should upsert android device successfully": {
			requestBody: DeviceRequest{
				ExpoPushToken: "ExponentPushToken[xyz789]",
				Platform:      "android",
			},
			mockSetup: func(m *MockDeviceUseCase) {
				expectedInput := usecase.DeviceInput{
					ExpoPushToken: "ExponentPushToken[xyz789]",
					Platform:      "android",
				}
				m.On("Upsert", mock.Anything, expectedInput).Return(domain.Device{
					ID:            fixedID,
					UserID:        "user-123",
					ExpoPushToken: "ExponentPushToken[xyz789]",
					Platform:      domain.PlatformAndroid,
					DateCreate:    fixedTime,
					DateUpdate:    fixedTime,
					LastSeenAt:    &fixedTime,
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"id":"11111111-1111-1111-1111-111111111111","expo_push_token":"ExponentPushToken[xyz789]","platform":"android","date_create":"2024-01-15T10:00:00Z","date_update":"2024-01-15T10:00:00Z","last_seen_at":"2024-01-15T10:00:00Z"}`,
		},
		"should return error when expo_push_token is missing": {
			requestBody: map[string]string{
				"platform": "ios",
			},
			mockSetup:      func(_ *MockDeviceUseCase) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":{"code":400,"message":"Invalid data provided"}}`,
		},
		"should return error when platform is missing": {
			requestBody: map[string]string{
				"expo_push_token": "ExponentPushToken[abc123]",
			},
			mockSetup:      func(_ *MockDeviceUseCase) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":{"code":400,"message":"Invalid data provided"}}`,
		},
		"should return error when body is invalid json": {
			requestBody:    "invalid",
			mockSetup:      func(_ *MockDeviceUseCase) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":{"code":400,"message":"Invalid data provided"}}`,
		},
		"should return error when usecase fails with empty token": {
			requestBody: DeviceRequest{
				ExpoPushToken: "",
				Platform:      "ios",
			},
			mockSetup:      func(_ *MockDeviceUseCase) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":{"code":400,"message":"Invalid data provided"}}`,
		},
		"should return error when usecase fails with invalid platform": {
			requestBody: DeviceRequest{
				ExpoPushToken: "ExponentPushToken[abc123]",
				Platform:      "invalid",
			},
			mockSetup: func(m *MockDeviceUseCase) {
				expectedInput := usecase.DeviceInput{
					ExpoPushToken: "ExponentPushToken[abc123]",
					Platform:      "invalid",
				}
				m.On("Upsert", mock.Anything, expectedInput).Return(domain.Device{}, usecase.ErrInvalidPlatform)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":{"code":400,"message":"Invalid data provided"}}`,
		},
		"should return error when usecase fails with database error": {
			requestBody: DeviceRequest{
				ExpoPushToken: "ExponentPushToken[abc123]",
				Platform:      "ios",
			},
			mockSetup: func(m *MockDeviceUseCase) {
				expectedInput := usecase.DeviceInput{
					ExpoPushToken: "ExponentPushToken[abc123]",
					Platform:      "ios",
				}
				m.On("Upsert", mock.Anything, expectedInput).Return(domain.Device{}, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":{"code":500,"message":"Internal server error"}}`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			router := setupDeviceRouter()
			mockUseCase := new(MockDeviceUseCase)
			tt.mockSetup(mockUseCase)

			NewDeviceHandlers(router, mockUseCase)

			var body []byte
			var err error
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				assert.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPost, "/devices", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)
			assert.Equal(t, tt.expectedBody, resp.Body.String())
			mockUseCase.AssertExpectations(t)
		})
	}
}

func TestDeviceHandler_List(t *testing.T) {
	fixedTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	fixedID1 := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	fixedID2 := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	tests := map[string]struct {
		mockSetup      func(m *MockDeviceUseCase)
		expectedStatus int
		expectedBody   string
	}{
		"should list devices successfully": {
			mockSetup: func(m *MockDeviceUseCase) {
				m.On("List", mock.Anything).Return([]domain.Device{
					{
						ID:            fixedID1,
						UserID:        "user-123",
						ExpoPushToken: "ExponentPushToken[abc123]",
						Platform:      domain.PlatformIOS,
						DateCreate:    fixedTime,
						DateUpdate:    fixedTime,
						LastSeenAt:    &fixedTime,
					},
					{
						ID:            fixedID2,
						UserID:        "user-123",
						ExpoPushToken: "ExponentPushToken[xyz789]",
						Platform:      domain.PlatformAndroid,
						DateCreate:    fixedTime,
						DateUpdate:    fixedTime,
						LastSeenAt:    nil,
					},
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `[{"id":"11111111-1111-1111-1111-111111111111","expo_push_token":"ExponentPushToken[abc123]","platform":"ios","date_create":"2024-01-15T10:00:00Z","date_update":"2024-01-15T10:00:00Z","last_seen_at":"2024-01-15T10:00:00Z"},{"id":"22222222-2222-2222-2222-222222222222","expo_push_token":"ExponentPushToken[xyz789]","platform":"android","date_create":"2024-01-15T10:00:00Z","date_update":"2024-01-15T10:00:00Z"}]`,
		},
		"should return empty array when no devices": {
			mockSetup: func(m *MockDeviceUseCase) {
				m.On("List", mock.Anything).Return([]domain.Device{}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `[]`,
		},
		"should return error when usecase fails": {
			mockSetup: func(m *MockDeviceUseCase) {
				m.On("List", mock.Anything).Return([]domain.Device{}, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":{"code":500,"message":"Internal server error"}}`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			router := setupDeviceRouter()
			mockUseCase := new(MockDeviceUseCase)
			tt.mockSetup(mockUseCase)

			NewDeviceHandlers(router, mockUseCase)

			req := httptest.NewRequest(http.MethodGet, "/devices", nil)
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)
			assert.Equal(t, tt.expectedBody, resp.Body.String())
			mockUseCase.AssertExpectations(t)
		})
	}
}

func TestDeviceHandler_Delete(t *testing.T) {
	tests := map[string]struct {
		token          string
		mockSetup      func(m *MockDeviceUseCase)
		expectedStatus int
		expectedBody   string
	}{
		"should delete device successfully": {
			token: "ExponentPushToken[abc123]",
			mockSetup: func(m *MockDeviceUseCase) {
				m.On("Delete", mock.Anything, "ExponentPushToken[abc123]").Return(nil)
			},
			expectedStatus: http.StatusNoContent,
			expectedBody:   "",
		},
		"should return error when device not found": {
			token: "ExponentPushToken[notfound]",
			mockSetup: func(m *MockDeviceUseCase) {
				m.On("Delete", mock.Anything, "ExponentPushToken[notfound]").Return(repository.ErrDeviceNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"error":{"code":404,"message":"Resource not found"}}`,
		},
		"should return error when usecase fails": {
			token: "ExponentPushToken[abc123]",
			mockSetup: func(m *MockDeviceUseCase) {
				m.On("Delete", mock.Anything, "ExponentPushToken[abc123]").Return(errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":{"code":500,"message":"Internal server error"}}`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			router := setupDeviceRouter()
			mockUseCase := new(MockDeviceUseCase)
			tt.mockSetup(mockUseCase)

			NewDeviceHandlers(router, mockUseCase)

			req := httptest.NewRequest(http.MethodDelete, "/devices/"+tt.token, nil)
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)
			assert.Equal(t, tt.expectedBody, resp.Body.String())
			mockUseCase.AssertExpectations(t)
		})
	}
}
