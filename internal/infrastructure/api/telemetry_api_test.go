package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"personal-finance/internal/usecase"
	"personal-finance/pkg/log"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupTelemetryRouter() *gin.Engine {
	log.Initialize()
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestTelemetryHandler_Ingest(t *testing.T) {
	validBody := `{
		"context": {
			"app_version": "1.1.0",
			"runtime_version": "54.0.0",
			"os": "ios",
			"os_version": "17.4",
			"session_id": "session-1"
		},
		"metrics": [
			{"kind": "counter", "name": "app_request_failed_total", "value": 1, "labels": {"reason": "timeout"}, "timestamp": 1719000000000},
			{"kind": "counter", "name": "app_malformed", "value": "not-a-number", "timestamp": 1719000000001}
		],
		"logs": [
			{"level": "error", "message": "boom", "fields": {"stack": "..."}, "timestamp": 1719000000200}
		]
	}`

	t.Run("should accept a valid batch and forward it to the use case", func(t *testing.T) {
		router := setupTelemetryRouter()
		mockUseCase := new(MockTelemetryUseCase)
		mockUseCase.On("Ingest", mock.Anything, mock.MatchedBy(func(input usecase.TelemetryBatchInput) bool {
			return input.Context.AppVersion == "1.1.0" &&
				input.Context.SessionID == "session-1" &&
				len(input.Metrics) == 1 &&
				input.Metrics[0].Name == "app_request_failed_total" &&
				len(input.Logs) == 1 &&
				input.Logs[0].Message == "boom"
		})).Return()

		NewTelemetryHandlers(router, mockUseCase)

		req := httptest.NewRequest(http.MethodPost, "/v1/telemetry", strings.NewReader(validBody))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusAccepted, resp.Code)
		mockUseCase.AssertExpectations(t)
	})

	t.Run("should reject malformed top-level json with 400", func(t *testing.T) {
		router := setupTelemetryRouter()
		mockUseCase := new(MockTelemetryUseCase)

		NewTelemetryHandlers(router, mockUseCase)

		req := httptest.NewRequest(http.MethodPost, "/v1/telemetry", strings.NewReader("not json"))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
		mockUseCase.AssertNotCalled(t, "Ingest")
	})

	t.Run("should reject oversized declared content-length with 413", func(t *testing.T) {
		router := setupTelemetryRouter()
		mockUseCase := new(MockTelemetryUseCase)

		NewTelemetryHandlers(router, mockUseCase)

		body := bytes.Repeat([]byte("a"), 10)
		req := httptest.NewRequest(http.MethodPost, "/v1/telemetry", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.ContentLength = 10 * 1024 * 1024 // lie about the size to trigger the guard

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusRequestEntityTooLarge, resp.Code)
		mockUseCase.AssertNotCalled(t, "Ingest")
	})

	t.Run("should reject a batch with too many events with 413", func(t *testing.T) {
		router := setupTelemetryRouter()
		mockUseCase := new(MockTelemetryUseCase)

		NewTelemetryHandlers(router, mockUseCase)

		var sb strings.Builder
		sb.WriteString(`{"context":{},"metrics":[`)
		for i := 0; i < 501; i++ {
			if i > 0 {
				sb.WriteString(",")
			}
			sb.WriteString(`{"kind":"counter","name":"app_x","value":1}`)
		}
		sb.WriteString(`],"logs":[]}`)

		req := httptest.NewRequest(http.MethodPost, "/v1/telemetry", strings.NewReader(sb.String()))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusRequestEntityTooLarge, resp.Code)
		mockUseCase.AssertNotCalled(t, "Ingest")
	})

	t.Run("should accept an empty batch", func(t *testing.T) {
		router := setupTelemetryRouter()
		mockUseCase := new(MockTelemetryUseCase)
		mockUseCase.On("Ingest", mock.Anything, mock.Anything).Return()

		NewTelemetryHandlers(router, mockUseCase)

		req := httptest.NewRequest(http.MethodPost, "/v1/telemetry", strings.NewReader(`{"context":{},"metrics":[],"logs":[]}`))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusAccepted, resp.Code)
		mockUseCase.AssertExpectations(t)
	})
}
