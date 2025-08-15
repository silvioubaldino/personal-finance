package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"personal-finance/internal/domain"
	"personal-finance/internal/domain/fixture"
	"personal-finance/pkg/log"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupCreditCardRouter() *gin.Engine {
	log.Initialize()
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestCreditCardHandler_Add(t *testing.T) {
	tests := map[string]struct {
		setupMock    func(mockUseCase *MockCreditCardUseCase)
		requestBody  interface{}
		expectedCode int
		expectedBody string
	}{
		"should_add_credit_card_successfully": {
			setupMock: func(mockUseCase *MockCreditCardUseCase) {
				creditCard := fixture.CreditCardMock()
				mockUseCase.On("Add", mock.Anything, mock.AnythingOfType("domain.CreditCard")).Return(creditCard, nil)
			},
			requestBody: map[string]interface{}{
				"name":              "Cartão Teste",
				"credit_limit":      5000.0,
				"closing_day":       15,
				"due_day":           22,
				"default_wallet_id": "77777777-7777-7777-7777-777777777777",
			},
			expectedCode: http.StatusCreated,
		},
		"should_fail_with_invalid_json": {
			setupMock: func(mockUseCase *MockCreditCardUseCase) {
				// No setup needed for this test
			},
			requestBody:  "invalid json",
			expectedCode: http.StatusBadRequest,
		},
		"should_fail_when_usecase_returns_error": {
			setupMock: func(mockUseCase *MockCreditCardUseCase) {
				mockUseCase.On("Add", mock.Anything, mock.AnythingOfType("domain.CreditCard")).Return(domain.CreditCard{}, fmt.Errorf("usecase error"))
			},
			requestBody: map[string]interface{}{
				"name":         "Cartão Teste",
				"credit_limit": 5000.0,
				"closing_day":  15,
				"due_day":      22,
			},
			expectedCode: http.StatusInternalServerError,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mockUseCase := &MockCreditCardUseCase{}
			tc.setupMock(mockUseCase)

			router := setupCreditCardRouter()
			NewCreditCardV2Handlers(router, mockUseCase)

			var body []byte
			if str, ok := tc.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, _ = json.Marshal(tc.requestBody)
			}

			req, _ := http.NewRequest("POST", "/v2/creditcards/", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedCode, w.Code)
			mockUseCase.AssertExpectations(t)
		})
	}
}

func TestCreditCardHandler_FindAll(t *testing.T) {
	tests := map[string]struct {
		setupMock    func(mockUseCase *MockCreditCardUseCase)
		expectedCode int
	}{
		"should_find_all_credit_cards_successfully": {
			setupMock: func(mockUseCase *MockCreditCardUseCase) {
				creditCards := []domain.CreditCard{
					fixture.CreditCardMock(),
					fixture.CreditCardMock(fixture.WithCreditCardName("Cartão 2")),
				}
				mockUseCase.On("FindAll", mock.Anything).Return(creditCards, nil)
			},
			expectedCode: http.StatusOK,
		},
		"should_return_empty_array_when_no_credit_cards_found": {
			setupMock: func(mockUseCase *MockCreditCardUseCase) {
				mockUseCase.On("FindAll", mock.Anything).Return([]domain.CreditCard{}, nil)
			},
			expectedCode: http.StatusOK,
		},
		"should_fail_when_usecase_returns_error": {
			setupMock: func(mockUseCase *MockCreditCardUseCase) {
				mockUseCase.On("FindAll", mock.Anything).Return([]domain.CreditCard{}, fmt.Errorf("usecase error"))
			},
			expectedCode: http.StatusInternalServerError,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mockUseCase := &MockCreditCardUseCase{}
			tc.setupMock(mockUseCase)

			router := setupCreditCardRouter()
			NewCreditCardV2Handlers(router, mockUseCase)

			req, _ := http.NewRequest("GET", "/v2/creditcards/", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedCode, w.Code)
			mockUseCase.AssertExpectations(t)
		})
	}
}

func TestCreditCardHandler_FindByID(t *testing.T) {
	tests := map[string]struct {
		setupMock    func(mockUseCase *MockCreditCardUseCase)
		id           string
		expectedCode int
	}{
		"should_find_credit_card_by_id_successfully": {
			setupMock: func(mockUseCase *MockCreditCardUseCase) {
				creditCard := fixture.CreditCardMock()
				mockUseCase.On("FindByID", mock.Anything, fixture.CreditCardID).Return(creditCard, nil)
			},
			id:           fixture.CreditCardID.String(),
			expectedCode: http.StatusOK,
		},
		"should_fail_with_invalid_id": {
			setupMock: func(mockUseCase *MockCreditCardUseCase) {
				// No setup needed for this test
			},
			id:           "invalid-uuid",
			expectedCode: http.StatusBadRequest,
		},
		"should_fail_when_usecase_returns_error": {
			setupMock: func(mockUseCase *MockCreditCardUseCase) {
				mockUseCase.On("FindByID", mock.Anything, fixture.CreditCardID).Return(domain.CreditCard{}, fmt.Errorf("usecase error"))
			},
			id:           fixture.CreditCardID.String(),
			expectedCode: http.StatusInternalServerError,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mockUseCase := &MockCreditCardUseCase{}
			tc.setupMock(mockUseCase)

			router := setupCreditCardRouter()
			NewCreditCardV2Handlers(router, mockUseCase)

			req, _ := http.NewRequest("GET", fmt.Sprintf("/v2/creditcards/%s", tc.id), nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedCode, w.Code)
			mockUseCase.AssertExpectations(t)
		})
	}
}

func TestCreditCardHandler_Update(t *testing.T) {
	tests := map[string]struct {
		setupMock    func(mockUseCase *MockCreditCardUseCase)
		id           string
		requestBody  interface{}
		expectedCode int
	}{
		"should_update_credit_card_successfully": {
			setupMock: func(mockUseCase *MockCreditCardUseCase) {
				creditCard := fixture.CreditCardMock(fixture.WithCreditCardName("Cartão Atualizado"))
				mockUseCase.On("Update", mock.Anything, fixture.CreditCardID, mock.AnythingOfType("domain.CreditCard")).Return(creditCard, nil)
			},
			id: fixture.CreditCardID.String(),
			requestBody: map[string]interface{}{
				"name":         "Cartão Atualizado",
				"credit_limit": 7000.0,
				"closing_day":  20,
				"due_day":      25,
			},
			expectedCode: http.StatusOK,
		},
		"should_fail_with_invalid_id": {
			setupMock: func(mockUseCase *MockCreditCardUseCase) {
				// No setup needed for this test
			},
			id: "invalid-uuid",
			requestBody: map[string]interface{}{
				"name": "Cartão Teste",
			},
			expectedCode: http.StatusBadRequest,
		},
		"should_fail_with_invalid_json": {
			setupMock: func(mockUseCase *MockCreditCardUseCase) {
				// No setup needed for this test
			},
			id:           fixture.CreditCardID.String(),
			requestBody:  "invalid json",
			expectedCode: http.StatusBadRequest,
		},
		"should_fail_when_usecase_returns_error": {
			setupMock: func(mockUseCase *MockCreditCardUseCase) {
				mockUseCase.On("Update", mock.Anything, fixture.CreditCardID, mock.AnythingOfType("domain.CreditCard")).Return(domain.CreditCard{}, fmt.Errorf("usecase error"))
			},
			id: fixture.CreditCardID.String(),
			requestBody: map[string]interface{}{
				"name": "Cartão Teste",
			},
			expectedCode: http.StatusInternalServerError,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mockUseCase := &MockCreditCardUseCase{}
			tc.setupMock(mockUseCase)

			router := setupCreditCardRouter()
			NewCreditCardV2Handlers(router, mockUseCase)

			var body []byte
			if str, ok := tc.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, _ = json.Marshal(tc.requestBody)
			}

			req, _ := http.NewRequest("PUT", fmt.Sprintf("/v2/creditcards/%s", tc.id), bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedCode, w.Code)
			mockUseCase.AssertExpectations(t)
		})
	}
}

func TestCreditCardHandler_Delete(t *testing.T) {
	tests := map[string]struct {
		setupMock    func(mockUseCase *MockCreditCardUseCase)
		id           string
		expectedCode int
		expectedBody string
	}{
		"should_delete_credit_card_successfully": {
			setupMock: func(mockUseCase *MockCreditCardUseCase) {
				mockUseCase.On("Delete", mock.Anything, fixture.CreditCardID).Return(nil)
			},
			id:           fixture.CreditCardID.String(),
			expectedCode: http.StatusNoContent,
			expectedBody: "",
		},
		"should_fail_with_invalid_id": {
			setupMock: func(mockUseCase *MockCreditCardUseCase) {
				// No setup needed for this test
			},
			id:           "invalid-uuid",
			expectedCode: http.StatusBadRequest,
		},
		"should_fail_when_usecase_returns_error": {
			setupMock: func(mockUseCase *MockCreditCardUseCase) {
				mockUseCase.On("Delete", mock.Anything, fixture.CreditCardID).Return(fmt.Errorf("usecase error"))
			},
			id:           fixture.CreditCardID.String(),
			expectedCode: http.StatusInternalServerError,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mockUseCase := &MockCreditCardUseCase{}
			tc.setupMock(mockUseCase)

			router := setupCreditCardRouter()
			NewCreditCardV2Handlers(router, mockUseCase)

			req, _ := http.NewRequest("DELETE", fmt.Sprintf("/v2/creditcards/%s", tc.id), nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedCode, w.Code)
			if tc.expectedBody != "" {
				assert.Equal(t, tc.expectedBody, w.Body.String())
			}
			mockUseCase.AssertExpectations(t)
		})
	}
}
