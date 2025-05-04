package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"personal-finance/pkg/log"
	"testing"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/domain/fixture"
	"personal-finance/internal/domain/output"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupRouter() *gin.Engine {
	log.Initialize()
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestMovementHandler_AddSimple(t *testing.T) {
	tests := map[string]struct {
		input          any
		mockSetup      func(mock *MockMovementUseCase)
		expectedStatus int
		expectedBody   any
	}{
		"should add movement successfully": {
			input: fixture.MovementMock(
				fixture.WithMovementDescription("Test movement"),
				fixture.AsMovementExpense(100.0),
			),
			mockSetup: func(mockMov *MockMovementUseCase) {
				movement := fixture.MovementMock(
					fixture.WithMovementDescription("Test movement"),
					fixture.AsMovementExpense(100.0),
				)
				mockMov.On("Add", mock.Anything, movement).Return(movement, nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody: func() string {
				a := output.ToMovementOutput(
					fixture.MovementMock(
						fixture.WithMovementDescription("Test movement"),
						fixture.AsMovementExpense(100.0),
					),
				)
				body, err := json.Marshal(a)
				assert.NoError(t, err)
				return string(body)
			}(),
		},
		"should fail binding json": {
			input:          "{",
			mockSetup:      func(mockMov *MockMovementUseCase) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":{"code":400,"message":"Invalid data provided"}}`,
		},
		"should return error when usecase fails": {
			input: fixture.MovementMock(),
			mockSetup: func(mockMov *MockMovementUseCase) {
				mockMov.On("Add", mock.Anything, mock.Anything).
					Return(domain.Movement{}, errors.New("usecase error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":{"code":500,"message":"Internal server error"}}`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			router := setupRouter()
			mockUseCase := new(MockMovementUseCase)
			if tt.mockSetup != nil {
				tt.mockSetup(mockUseCase)
			}

			NewMovementV2Handlers(router, mockUseCase)

			body, _ := json.Marshal(tt.input)
			req := httptest.NewRequest(http.MethodPost, "/v2/movements/", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)
			assert.Equal(t, tt.expectedBody, resp.Body.String())

			mockUseCase.AssertExpectations(t)
		})
	}
}

func TestMovementHandler_FindByPeriod(t *testing.T) {
	tests := map[string]struct {
		queryParams    string
		mockSetup      func(mockMov *MockMovementUseCase)
		expectedStatus int
		expectedBody   string
	}{
		"should find movements by period successfully": {
			queryParams: "from=2025-01-01&to=2025-01-31",
			mockSetup: func(mockMov *MockMovementUseCase) {
				period := domain.Period{
					From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
					To:   time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC),
				}
				movements := []domain.Movement{
					fixture.MovementMock(
						fixture.WithMovementDescription("Movement 1"),
						fixture.AsMovementExpense(100.0),
					),
					fixture.MovementMock(
						fixture.WithMovementDescription("Movement 2"),
						fixture.AsMovementIncome(200.0),
					),
				}
				mockMov.On("FindByPeriod", mock.Anything, period).Return(movements, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func() string {
				movements := []output.MovementOutput{
					*output.ToMovementOutput(fixture.MovementMock(
						fixture.WithMovementDescription("Movement 1"),
						fixture.AsMovementExpense(100.0),
					)),
					*output.ToMovementOutput(fixture.MovementMock(
						fixture.WithMovementDescription("Movement 2"),
						fixture.AsMovementIncome(200.0),
					)),
				}
				body, err := json.Marshal(movements)
				assert.NoError(t, err)
				return string(body)
			}(),
		},
		"should return error when from period is invalid": {
			queryParams:    "from=2025-01-01&to=invalid-date",
			mockSetup:      func(mockMov *MockMovementUseCase) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":{"code":400,"message":"Invalid data provided"}}`,
		},
		"should return error when to period is invalid": {
			queryParams:    "from=invalid-date",
			mockSetup:      func(mockMov *MockMovementUseCase) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":{"code":400,"message":"Invalid data provided"}}`,
		},
		"should return error when usecase fails": {
			queryParams: "from=2025-01-01&to=2025-01-31",
			mockSetup: func(mockMov *MockMovementUseCase) {
				mockMov.On("FindByPeriod", mock.Anything, mock.Anything).
					Return([]domain.Movement{}, errors.New("usecase error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":{"code":500,"message":"Internal server error"}}`,
		},
		"should return error when period validation fails": {
			queryParams:    "from=2025-01-31&to=2025-01-01",
			mockSetup:      func(mockMov *MockMovementUseCase) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":{"code":400,"message":"Invalid data provided"}}`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			router := setupRouter()
			mockUseCase := new(MockMovementUseCase)
			if tt.mockSetup != nil {
				tt.mockSetup(mockUseCase)
			}

			NewMovementV2Handlers(router, mockUseCase)

			req := httptest.NewRequest(http.MethodGet, "/v2/movements/?"+tt.queryParams, nil)
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)
			assert.Equal(t, tt.expectedBody, resp.Body.String())
			mockUseCase.AssertExpectations(t)
		})
	}
}
