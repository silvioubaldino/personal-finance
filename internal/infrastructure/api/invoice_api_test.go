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

func setupInvoiceRouter() *gin.Engine {
	log.Initialize()
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestInvoiceHandler_FindByMonth(t *testing.T) {
	testDate := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		queryParams    string
		mockSetup      func(mock *MockInvoiceUseCase)
		expectedStatus int
		expectedBody   string
	}{
		"should find invoices by month successfully": {
			queryParams: "date=2025-01-15",
			mockSetup: func(mockInv *MockInvoiceUseCase) {
				invoices := []domain.Invoice{
					fixture.InvoiceMock(),
				}
				mockInv.On("FindOpenByMonth", mock.Anything, testDate).Return(invoices, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func() string {
				invoices := []output.InvoiceOutput{
					output.ToInvoiceOutput(fixture.InvoiceMock()),
				}
				body, err := json.Marshal(invoices)
				assert.NoError(t, err)
				return string(body)
			}(),
		},
		"should find invoices successfully without date": {
			queryParams: "",
			mockSetup: func(mockInv *MockInvoiceUseCase) {
				zeroDate := time.Time{}
				mockInv.On("FindOpenByMonth", mock.Anything, zeroDate).Return([]domain.Invoice{}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `[]`,
		},
		"should return empty array when no invoices found": {
			queryParams: "date=2025-01-15",
			mockSetup: func(mockInv *MockInvoiceUseCase) {
				mockInv.On("FindOpenByMonth", mock.Anything, testDate).Return([]domain.Invoice{}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `[]`,
		},
		"should return error when usecase fails": {
			queryParams: "date=2025-01-15",
			mockSetup: func(mockInv *MockInvoiceUseCase) {
				mockInv.On("FindOpenByMonth", mock.Anything, testDate).
					Return([]domain.Invoice{}, errors.New("usecase error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":{"code":500,"message":"Internal server error"}}`,
		},
		"should return error when date format is invalid": {
			queryParams:    "date=invalid-date",
			mockSetup:      func(mockInv *MockInvoiceUseCase) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":{"code":400,"message":"Invalid data provided"}}`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			router := setupInvoiceRouter()
			mockUseCase := new(MockInvoiceUseCase)
			if tt.mockSetup != nil {
				tt.mockSetup(mockUseCase)
			}

			NewInvoiceV2Handlers(router, mockUseCase)

			req := httptest.NewRequest(http.MethodGet, "/v2/invoices/date?"+tt.queryParams, nil)
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)
			assert.Equal(t, tt.expectedBody, resp.Body.String())
			mockUseCase.AssertExpectations(t)
		})
	}
}

func TestInvoiceHandler_FindByID(t *testing.T) {
	validID := fixture.InvoiceMock().ID

	tests := map[string]struct {
		id             string
		mockSetup      func(mock *MockInvoiceUseCase)
		expectedStatus int
		expectedBody   string
	}{
		"should find invoice by ID successfully": {
			id: validID.String(),
			mockSetup: func(mockInv *MockInvoiceUseCase) {
				invoice := fixture.InvoiceMock()
				mockInv.On("FindByID", mock.Anything, *validID).Return(invoice, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func() string {
				inv := output.ToInvoiceOutput(fixture.InvoiceMock())
				body, err := json.Marshal(inv)
				assert.NoError(t, err)
				return string(body)
			}(),
		},
		"should fail with invalid id": {
			id:             "invalid-uuid",
			mockSetup:      func(mockInv *MockInvoiceUseCase) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":{"code":400,"message":"Invalid data provided"}}`,
		},
		"should return error when usecase fails": {
			id: validID.String(),
			mockSetup: func(mockInv *MockInvoiceUseCase) {
				mockInv.On("FindByID", mock.Anything, *validID).
					Return(domain.Invoice{}, errors.New("usecase error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":{"code":500,"message":"Internal server error"}}`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			router := setupInvoiceRouter()
			mockUseCase := new(MockInvoiceUseCase)
			tt.mockSetup(mockUseCase)

			NewInvoiceV2Handlers(router, mockUseCase)

			req := httptest.NewRequest(http.MethodGet, "/v2/invoices/"+tt.id, nil)
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)
			assert.Equal(t, tt.expectedBody, resp.Body.String())
			mockUseCase.AssertExpectations(t)
		})
	}
}

func TestInvoiceHandler_Pay(t *testing.T) {
	validID := fixture.InvoiceMock().ID
	validWalletID := fixture.WalletMock().ID
	paymentDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		id             string
		input          any
		mockSetup      func(mock *MockInvoiceUseCase)
		expectedStatus int
		expectedBody   string
	}{
		"should pay invoice successfully": {
			id: validID.String(),
			input: PayInvoiceRequest{
				WalletID:    *validWalletID,
				PaymentDate: &paymentDate,
			},
			mockSetup: func(mockInv *MockInvoiceUseCase) {
				invoice := fixture.InvoiceMock()
				mockInv.On("Pay", mock.Anything, *validID, *validWalletID, &paymentDate).Return(invoice, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func() string {
				inv := output.ToInvoiceOutput(fixture.InvoiceMock())
				body, err := json.Marshal(inv)
				assert.NoError(t, err)
				return string(body)
			}(),
		},
		"should pay invoice successfully without payment date": {
			id: validID.String(),
			input: PayInvoiceRequest{
				WalletID: *validWalletID,
			},
			mockSetup: func(mockInv *MockInvoiceUseCase) {
				invoice := fixture.InvoiceMock()
				mockInv.On("Pay", mock.Anything, *validID, *validWalletID, (*time.Time)(nil)).Return(invoice, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func() string {
				inv := output.ToInvoiceOutput(fixture.InvoiceMock())
				body, err := json.Marshal(inv)
				assert.NoError(t, err)
				return string(body)
			}(),
		},
		"should fail with invalid id": {
			id: "invalid-uuid",
			input: PayInvoiceRequest{
				WalletID: *validWalletID,
			},
			mockSetup:      func(mockInv *MockInvoiceUseCase) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":{"code":400,"message":"Invalid data provided"}}`,
		},
		"should fail binding json": {
			id:             validID.String(),
			input:          "{",
			mockSetup:      func(mockInv *MockInvoiceUseCase) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":{"code":400,"message":"Invalid data provided"}}`,
		},
		"should fail with missing wallet_id": {
			id: validID.String(),
			input: map[string]interface{}{
				"payment_date": "2025-01-01T00:00:00Z",
			},
			mockSetup:      func(mockInv *MockInvoiceUseCase) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":{"code":400,"message":"Invalid data provided"}}`,
		},
		"should return error when usecase fails": {
			id: validID.String(),
			input: PayInvoiceRequest{
				WalletID: *validWalletID,
			},
			mockSetup: func(mockInv *MockInvoiceUseCase) {
				mockInv.On("Pay", mock.Anything, *validID, *validWalletID, (*time.Time)(nil)).
					Return(domain.Invoice{}, errors.New("usecase error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":{"code":500,"message":"Internal server error"}}`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			router := setupInvoiceRouter()
			mockUseCase := new(MockInvoiceUseCase)
			tt.mockSetup(mockUseCase)

			NewInvoiceV2Handlers(router, mockUseCase)

			body, _ := json.Marshal(tt.input)
			req := httptest.NewRequest(http.MethodPost, "/v2/invoices/"+tt.id+"/pay", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)
			assert.Equal(t, tt.expectedBody, resp.Body.String())
			mockUseCase.AssertExpectations(t)
		})
	}
}

func TestInvoiceHandler_FindDetailedInvoicesByPeriod(t *testing.T) {
	fromDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	toDate := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)
	period := domain.Period{From: fromDate, To: toDate}

	tests := map[string]struct {
		queryParams    string
		mockSetup      func(mock *MockInvoiceUseCase)
		expectedStatus int
		expectedBody   string
	}{
		"should find detailed invoices by period successfully": {
			queryParams: "from=2025-01-01&to=2025-01-31",
			mockSetup: func(mockInv *MockInvoiceUseCase) {
				detailedInvoices := []domain.DetailedInvoice{
					fixture.DetailedInvoiceMock(),
				}
				mockInv.On("FindDetailedInvoicesByPeriod", mock.Anything, period).Return(detailedInvoices, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func() string {
				invoices := []output.DetailedInvoiceOutput{
					output.ToDetailedInvoiceOutput(fixture.DetailedInvoiceMock()),
				}
				body, err := json.Marshal(invoices)
				assert.NoError(t, err)
				return string(body)
			}(),
		},
		"should return empty array when no detailed invoices found": {
			queryParams: "from=2025-01-01&to=2025-01-31",
			mockSetup: func(mockInv *MockInvoiceUseCase) {
				mockInv.On("FindDetailedInvoicesByPeriod", mock.Anything, period).Return([]domain.DetailedInvoice{}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `[]`,
		},
		"should return error when usecase fails": {
			queryParams: "from=2025-01-01&to=2025-01-31",
			mockSetup: func(mockInv *MockInvoiceUseCase) {
				mockInv.On("FindDetailedInvoicesByPeriod", mock.Anything, period).
					Return([]domain.DetailedInvoice{}, errors.New("usecase error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":{"code":500,"message":"Internal server error"}}`,
		},
		"should return error when from date format is invalid": {
			queryParams:    "from=invalid-date&to=2025-01-31",
			mockSetup:      func(mockInv *MockInvoiceUseCase) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":{"code":400,"message":"Invalid data provided"}}`,
		},
		"should return error when to date format is invalid": {
			queryParams:    "from=2025-01-01&to=invalid-date",
			mockSetup:      func(mockInv *MockInvoiceUseCase) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":{"code":400,"message":"Invalid data provided"}}`,
		},
		"should return error when period is invalid (from after to)": {
			queryParams:    "from=2025-01-31&to=2025-01-01",
			mockSetup:      func(mockInv *MockInvoiceUseCase) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":{"code":400,"message":"Invalid data provided"}}`,
		},
		"should return error when no query params provided": {
			queryParams:    "",
			mockSetup:      func(mockInv *MockInvoiceUseCase) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":{"code":400,"message":"Invalid data provided"}}`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			router := setupInvoiceRouter()
			mockUseCase := new(MockInvoiceUseCase)
			if tt.mockSetup != nil {
				tt.mockSetup(mockUseCase)
			}

			NewInvoiceV2Handlers(router, mockUseCase)

			req := httptest.NewRequest(http.MethodGet, "/v2/invoices/detailed?"+tt.queryParams, nil)
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)
			assert.Equal(t, tt.expectedBody, resp.Body.String())
			mockUseCase.AssertExpectations(t)
		})
	}
}

func TestInvoiceHandler_RevertPayment(t *testing.T) {
	validID := fixture.InvoiceMock().ID

	tests := map[string]struct {
		id             string
		mockSetup      func(mock *MockInvoiceUseCase)
		expectedStatus int
		expectedBody   string
	}{
		"should revert invoice payment successfully": {
			id: validID.String(),
			mockSetup: func(mockInv *MockInvoiceUseCase) {
				invoice := fixture.InvoiceMock()
				mockInv.On("RevertPayment", mock.Anything, *validID).Return(invoice, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func() string {
				inv := output.ToInvoiceOutput(fixture.InvoiceMock())
				body, err := json.Marshal(inv)
				assert.NoError(t, err)
				return string(body)
			}(),
		},
		"should fail with invalid id": {
			id:             "invalid-uuid",
			mockSetup:      func(mockInv *MockInvoiceUseCase) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":{"code":400,"message":"Invalid data provided"}}`,
		},
		"should return error when usecase fails": {
			id: validID.String(),
			mockSetup: func(mockInv *MockInvoiceUseCase) {
				mockInv.On("RevertPayment", mock.Anything, *validID).
					Return(domain.Invoice{}, errors.New("usecase error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":{"code":500,"message":"Internal server error"}}`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			router := setupInvoiceRouter()
			mockUseCase := new(MockInvoiceUseCase)
			tt.mockSetup(mockUseCase)

			NewInvoiceV2Handlers(router, mockUseCase)

			req := httptest.NewRequest(http.MethodPost, "/v2/invoices/"+tt.id+"/revert-payment", nil)
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)
			assert.Equal(t, tt.expectedBody, resp.Body.String())
			mockUseCase.AssertExpectations(t)
		})
	}
}
