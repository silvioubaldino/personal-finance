package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/domain/fixture"
	"personal-finance/internal/domain/output"
	"personal-finance/internal/infrastructure/repository"
	"personal-finance/internal/usecase"
	"personal-finance/pkg/log"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
				mockInv.On("FindByMonth", testDate).Return(invoices, nil)
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
				mockInv.On("FindByMonth", zeroDate).Return([]domain.Invoice{}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `[]`,
		},
		"should return empty array when no invoices found": {
			queryParams: "date=2025-01-15",
			mockSetup: func(mockInv *MockInvoiceUseCase) {
				mockInv.On("FindByMonth", testDate).Return([]domain.Invoice{}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `[]`,
		},
		"should return error when usecase fails": {
			queryParams: "date=2025-01-15",
			mockSetup: func(mockInv *MockInvoiceUseCase) {
				mockInv.On("FindByMonth", testDate).
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
				inv := output.ToInvoiceWithAdvancePaymentsOutput(fixture.InvoiceMock())
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
				mockInv.On("Pay", mock.Anything, *validID, *validWalletID, &paymentDate, mock.Anything).Return(invoice, nil)
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
				mockInv.On("Pay", mock.Anything, *validID, *validWalletID, (*time.Time)(nil), mock.Anything).Return(invoice, nil)
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
				mockInv.On("Pay", mock.Anything, *validID, *validWalletID, (*time.Time)(nil), mock.Anything).
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

			req := httptest.NewRequest(http.MethodPost, "/v2/invoices/"+tt.id+"/revert-pay", nil)
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)
			assert.Equal(t, tt.expectedBody, resp.Body.String())
			mockUseCase.AssertExpectations(t)
		})
	}
}

func TestInvoiceHandler_AdvancePayment(t *testing.T) {
	validID := fixture.InvoiceMock().ID
	walletID := fixture.WalletMock().ID
	movementID := uuid.MustParse("99999999-9999-9999-9999-999999999999")
	paymentDate := time.Date(2026, 1, 11, 0, 0, 0, 0, time.UTC)

	invoiceWithAdvance := fixture.InvoiceMock().WithAdvancePayments([]domain.InvoiceAdvancePayment{
		{ID: movementID, PaymentDate: paymentDate, Amount: -150, WalletID: walletID, Description: "Adiantamento da fatura - Card"},
	})

	tests := map[string]struct {
		id             string
		body           string
		mockSetup      func(mockInv *MockInvoiceUseCase)
		expectedStatus int
		expectedType   string
		assertBody     func(t *testing.T, body map[string]any)
	}{
		"should create advance payment with yyyy-mm-dd date": {
			id:   validID.String(),
			body: `{"amount":150,"wallet_id":"` + walletID.String() + `","payment_date":"2026-01-11"}`,
			mockSetup: func(mockInv *MockInvoiceUseCase) {
				date := paymentDate
				mockInv.On("AdvancePayment", mock.Anything, *validID, usecase.AdvancePaymentInput{
					Amount: 150, WalletID: walletID, PaymentDate: &date,
				}).Return(invoiceWithAdvance, nil)
			},
			expectedStatus: http.StatusCreated,
			assertBody: func(t *testing.T, body map[string]any) {
				assert.Equal(t, -150.0, body["advance_paid_amount"])
				assert.Equal(t, -1350.0, body["remaining_amount"])
				assert.Equal(t, -1500.0, body["amount"])
				payments := body["advance_payments"].([]any)
				assert.Len(t, payments, 1)
				assert.Equal(t, "2026-01-11", payments[0].(map[string]any)["payment_date"])
			},
		},
		"should create advance payment with defaults": {
			id:   validID.String(),
			body: `{"amount":-150}`,
			mockSetup: func(mockInv *MockInvoiceUseCase) {
				mockInv.On("AdvancePayment", mock.Anything, *validID, usecase.AdvancePaymentInput{Amount: -150}).
					Return(invoiceWithAdvance, nil)
			},
			expectedStatus: http.StatusCreated,
		},
		"should reject invalid payment_date": {
			id:             validID.String(),
			body:           `{"amount":150,"payment_date":"11/01/2026"}`,
			mockSetup:      func(mockInv *MockInvoiceUseCase) {},
			expectedStatus: http.StatusBadRequest,
		},
		"should reject missing amount": {
			id:             validID.String(),
			body:           `{}`,
			mockSetup:      func(mockInv *MockInvoiceUseCase) {},
			expectedStatus: http.StatusBadRequest,
		},
		"should reject invalid id": {
			id:             "invalid-uuid",
			body:           `{"amount":150}`,
			mockSetup:      func(mockInv *MockInvoiceUseCase) {},
			expectedStatus: http.StatusBadRequest,
		},
		"should map paid invoice to 409 invoice_already_paid": {
			id:   validID.String(),
			body: `{"amount":150}`,
			mockSetup: func(mockInv *MockInvoiceUseCase) {
				mockInv.On("AdvancePayment", mock.Anything, *validID, mock.Anything).
					Return(domain.Invoice{}, fmt.Errorf("advance invoice payment: %w", usecase.ErrAdvanceInvoiceAlreadyPaid))
			},
			expectedStatus: http.StatusConflict,
			expectedType:   "invoice_already_paid",
		},
		"should map settling amount to 422 advance_payment_settles_invoice": {
			id:   validID.String(),
			body: `{"amount":1500}`,
			mockSetup: func(mockInv *MockInvoiceUseCase) {
				mockInv.On("AdvancePayment", mock.Anything, *validID, mock.Anything).
					Return(domain.Invoice{}, fmt.Errorf("advance invoice payment: %w", usecase.ErrAdvancePaymentSettlesInvoice))
			},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedType:   "advance_payment_settles_invoice",
		},
		"should map date after due date to 422 advance_payment_after_due_date": {
			id:   validID.String(),
			body: `{"amount":100}`,
			mockSetup: func(mockInv *MockInvoiceUseCase) {
				mockInv.On("AdvancePayment", mock.Anything, *validID, mock.Anything).
					Return(domain.Invoice{}, fmt.Errorf("advance invoice payment: %w", usecase.ErrAdvancePaymentAfterDueDate))
			},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedType:   "advance_payment_after_due_date",
		},
		"should map invoice not found to 404": {
			id:   validID.String(),
			body: `{"amount":100}`,
			mockSetup: func(mockInv *MockInvoiceUseCase) {
				mockInv.On("AdvancePayment", mock.Anything, *validID, mock.Anything).
					Return(domain.Invoice{}, fmt.Errorf("error finding invoice: %w", repository.ErrInvoiceNotFound))
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			router := setupInvoiceRouter()
			mockUseCase := new(MockInvoiceUseCase)
			tt.mockSetup(mockUseCase)
			NewInvoiceV2Handlers(router, mockUseCase)

			req := httptest.NewRequest(http.MethodPost, "/v2/invoices/"+tt.id+"/advance-payments", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)
			var body map[string]any
			assert.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
			if tt.expectedType != "" {
				assert.Equal(t, tt.expectedType, body["error"].(map[string]any)["type"])
			}
			if tt.assertBody != nil {
				tt.assertBody(t, body)
			}
			mockUseCase.AssertExpectations(t)
		})
	}
}

func TestInvoiceHandler_RevertAdvancePayment(t *testing.T) {
	validID := fixture.InvoiceMock().ID
	movementID := uuid.MustParse("99999999-9999-9999-9999-999999999999")

	tests := map[string]struct {
		id             string
		movementID     string
		mockSetup      func(mockInv *MockInvoiceUseCase)
		expectedStatus int
		expectedType   string
	}{
		"should revert advance payment": {
			id:         validID.String(),
			movementID: movementID.String(),
			mockSetup: func(mockInv *MockInvoiceUseCase) {
				mockInv.On("RevertAdvancePayment", mock.Anything, *validID, movementID).
					Return(fixture.InvoiceMock().WithAdvancePayments(nil), nil)
			},
			expectedStatus: http.StatusOK,
		},
		"should reject invalid movement id": {
			id:             validID.String(),
			movementID:     "invalid",
			mockSetup:      func(mockInv *MockInvoiceUseCase) {},
			expectedStatus: http.StatusBadRequest,
		},
		"should map not an advance payment of the invoice to 404": {
			id:         validID.String(),
			movementID: movementID.String(),
			mockSetup: func(mockInv *MockInvoiceUseCase) {
				mockInv.On("RevertAdvancePayment", mock.Anything, *validID, movementID).
					Return(domain.Invoice{}, fmt.Errorf("revert advance invoice payment: %w", usecase.ErrAdvancePaymentNotFound))
			},
			expectedStatus: http.StatusNotFound,
			expectedType:   "advance_payment_not_found",
		},
		"should map paid invoice to 409": {
			id:         validID.String(),
			movementID: movementID.String(),
			mockSetup: func(mockInv *MockInvoiceUseCase) {
				mockInv.On("RevertAdvancePayment", mock.Anything, *validID, movementID).
					Return(domain.Invoice{}, fmt.Errorf("revert advance invoice payment: %w", usecase.ErrAdvanceInvoiceAlreadyPaid))
			},
			expectedStatus: http.StatusConflict,
			expectedType:   "invoice_already_paid",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			router := setupInvoiceRouter()
			mockUseCase := new(MockInvoiceUseCase)
			tt.mockSetup(mockUseCase)
			NewInvoiceV2Handlers(router, mockUseCase)

			req := httptest.NewRequest(http.MethodDelete, "/v2/invoices/"+tt.id+"/advance-payments/"+tt.movementID, nil)
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)
			if tt.expectedType != "" {
				var body map[string]any
				assert.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
				assert.Equal(t, tt.expectedType, body["error"].(map[string]any)["type"])
			}
			mockUseCase.AssertExpectations(t)
		})
	}
}
