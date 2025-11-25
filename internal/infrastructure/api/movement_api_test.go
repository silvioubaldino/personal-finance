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
	"personal-finance/internal/domain/fixture"
	"personal-finance/internal/domain/output"
	"personal-finance/pkg/log"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupRouter() *gin.Engine {
	log.Initialize()
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestMovementHandler_Add(t *testing.T) {
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
				movements := domain.MovementList{
					fixture.MovementMock(
						fixture.WithMovementDescription("Movement 1"),
						fixture.AsMovementExpense(100.0),
					),
					fixture.MovementMock(
						fixture.WithMovementDescription("Movement 2"),
						fixture.AsMovementIncome(200.0),
					),
				}
				invoices := []domain.DetailedInvoice{}
				periodData := domain.PeriodData{
					Movements: movements,
					Invoices:  invoices,
				}
				mockMov.On("FindByPeriod", mock.Anything, period).Return(periodData, nil)
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
				invoices := []output.DetailedInvoiceOutput{}
				response := PeriodMovementsResponse{
					Movements: movements,
					Invoices:  invoices,
				}
				body, err := json.Marshal(response)
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
					Return(domain.PeriodData{}, errors.New("usecase error"))
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

func TestMovementHandler_Pay(t *testing.T) {
	validID := fixture.MovementMock().ID
	validDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		id             string
		date           string
		mockSetup      func(mock *MockMovementUseCase)
		expectedStatus int
		expectedBody   string
	}{
		"should pay movement successfully": {
			id:   validID.String(),
			date: "2025-01-01",
			mockSetup: func(mockMov *MockMovementUseCase) {
				movement := fixture.MovementMock(
					fixture.WithMovementDescription("Test movement"),
					fixture.AsMovementExpense(100.0),
				)
				mockMov.On("Pay", mock.Anything, *validID, validDate).Return(movement, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func() string {
				movement := output.ToMovementOutput(fixture.MovementMock(
					fixture.WithMovementDescription("Test movement"),
					fixture.AsMovementExpense(100.0),
				))
				body, err := json.Marshal(movement)
				assert.NoError(t, err)
				return string(body)
			}(),
		},
		"should fail with invalid id": {
			id:             "invalid-uuid",
			date:           "2025-01-01",
			mockSetup:      func(mockMov *MockMovementUseCase) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":{"code":400,"message":"Invalid data provided"}}`,
		},
		"should fail with invalid date format": {
			id:             validID.String(),
			date:           "invalid-date",
			mockSetup:      func(mockMov *MockMovementUseCase) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":{"code":400,"message":"Invalid data provided"}}`,
		},
		"should return error when usecase fails": {
			id:   validID.String(),
			date: "2025-01-01",
			mockSetup: func(mockMov *MockMovementUseCase) {
				mockMov.On("Pay", mock.Anything, *validID, validDate).
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
			tt.mockSetup(mockUseCase)

			NewMovementV2Handlers(router, mockUseCase)

			url := "/v2/movements/" + tt.id + "/pay"
			if tt.date != "" {
				url += "?date=" + tt.date
			}

			req := httptest.NewRequest(http.MethodPost, url, nil)
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)
			assert.Equal(t, tt.expectedBody, resp.Body.String())
			mockUseCase.AssertExpectations(t)
		})
	}
}

func TestMovementHandler_RevertPay(t *testing.T) {
	validID := fixture.MovementMock().ID

	tests := map[string]struct {
		id             string
		mockSetup      func(mock *MockMovementUseCase)
		expectedStatus int
		expectedBody   string
	}{
		"should revert pay movement successfully": {
			id: validID.String(),
			mockSetup: func(mockMov *MockMovementUseCase) {
				movement := fixture.MovementMock(
					fixture.WithMovementDescription("Test movement"),
					fixture.AsMovementExpense(100.0),
				)
				mockMov.On("RevertPay", mock.Anything, *validID).Return(movement, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func() string {
				movement := output.ToMovementOutput(fixture.MovementMock(
					fixture.WithMovementDescription("Test movement"),
					fixture.AsMovementExpense(100.0),
				))
				body, err := json.Marshal(movement)
				assert.NoError(t, err)
				return string(body)
			}(),
		},
		"should fail with invalid id": {
			id:             "invalid-uuid",
			mockSetup:      func(mockMov *MockMovementUseCase) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":{"code":400,"message":"Invalid data provided"}}`,
		},
		"should return error when usecase fails": {
			id: validID.String(),
			mockSetup: func(mockMov *MockMovementUseCase) {
				mockMov.On("RevertPay", mock.Anything, *validID).
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
			tt.mockSetup(mockUseCase)

			NewMovementV2Handlers(router, mockUseCase)

			url := "/v2/movements/" + tt.id + "/revert-pay"

			req := httptest.NewRequest(http.MethodPost, url, nil)
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)
			assert.Equal(t, tt.expectedBody, resp.Body.String())
			mockUseCase.AssertExpectations(t)
		})
	}
}

func TestMovementHandler_DeleteOne(t *testing.T) {
	validID := fixture.MovementMock().ID

	tests := map[string]struct {
		id             string
		mockSetup      func(mock *MockMovementUseCase)
		expectedStatus int
		expectedBody   string
	}{
		"should delete movement successfully": {
			id: validID.String(),
			mockSetup: func(mockMov *MockMovementUseCase) {
				mockMov.On("DeleteOne", mock.Anything, *validID).Return(nil)
			},
			expectedStatus: http.StatusNoContent,
			expectedBody:   "",
		},
		"should fail with invalid id": {
			id:             "invalid-uuid",
			mockSetup:      func(mockMov *MockMovementUseCase) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":{"code":400,"message":"Invalid data provided"}}`,
		},
		"should return error when usecase fails": {
			id: validID.String(),
			mockSetup: func(mockMov *MockMovementUseCase) {
				mockMov.On("DeleteOne", mock.Anything, *validID).
					Return(errors.New("usecase error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":{"code":500,"message":"Internal server error"}}`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			router := setupRouter()
			mockUseCase := new(MockMovementUseCase)
			tt.mockSetup(mockUseCase)

			NewMovementV2Handlers(router, mockUseCase)

			url := "/v2/movements/" + tt.id

			req := httptest.NewRequest(http.MethodDelete, url, nil)
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)
			assert.Equal(t, tt.expectedBody, resp.Body.String())
			mockUseCase.AssertExpectations(t)
		})
	}
}

func TestMovementHandler_DeleteAllNext(t *testing.T) {
	validID := fixture.MovementMock().ID
	validDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		id             string
		date           string
		mockSetup      func(mock *MockMovementUseCase)
		expectedStatus int
		expectedBody   string
	}{
		"should delete all next movements successfully": {
			id:   validID.String(),
			date: "2025-01-01",
			mockSetup: func(mockMov *MockMovementUseCase) {
				mockMov.On("DeleteAllNext", mock.Anything, *validID, validDate).Return(nil)
			},
			expectedStatus: http.StatusNoContent,
			expectedBody:   "",
		},
		"should delete all next movements successfully without date": {
			id:   validID.String(),
			date: "",
			mockSetup: func(mockMov *MockMovementUseCase) {
				mockMov.On("DeleteAllNext", mock.Anything, *validID, time.Time{}).Return(nil)
			},
			expectedStatus: http.StatusNoContent,
			expectedBody:   "",
		},
		"should fail with invalid id": {
			id:             "invalid-uuid",
			date:           "2025-01-01",
			mockSetup:      func(mockMov *MockMovementUseCase) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":{"code":400,"message":"Invalid data provided"}}`,
		},
		"should fail with invalid date format": {
			id:             validID.String(),
			date:           "invalid-date",
			mockSetup:      func(mockMov *MockMovementUseCase) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":{"code":400,"message":"Invalid data provided"}}`,
		},
		"should return error when usecase fails": {
			id:   validID.String(),
			date: "2025-01-01",
			mockSetup: func(mockMov *MockMovementUseCase) {
				mockMov.On("DeleteAllNext", mock.Anything, *validID, validDate).
					Return(errors.New("usecase error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":{"code":500,"message":"Internal server error"}}`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			router := setupRouter()
			mockUseCase := new(MockMovementUseCase)
			tt.mockSetup(mockUseCase)

			NewMovementV2Handlers(router, mockUseCase)

			url := "/v2/movements/" + tt.id + "/all-next"
			if tt.date != "" {
				url += "?date=" + tt.date
			}

			req := httptest.NewRequest(http.MethodDelete, url, nil)
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)
			assert.Equal(t, tt.expectedBody, resp.Body.String())
			mockUseCase.AssertExpectations(t)
		})
	}
}
