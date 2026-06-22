package api

import (
	"context"
	"encoding/json"
	"net/http"

	"personal-finance/internal/domain"
	"personal-finance/internal/usecase"

	"github.com/gin-gonic/gin"
)

type (
	TelemetryUseCase interface {
		Ingest(ctx context.Context, input usecase.TelemetryBatchInput)
	}

	TelemetryHandler struct {
		usecase TelemetryUseCase
	}

	telemetryContextRequest struct {
		AppVersion     string `json:"app_version"`
		RuntimeVersion string `json:"runtime_version"`
		UpdateID       string `json:"update_id"`
		OS             string `json:"os"`
		OSVersion      string `json:"os_version"`
		SessionID      string `json:"session_id"`
	}

	telemetryMetricRequest struct {
		Kind      string            `json:"kind"`
		Name      string            `json:"name"`
		Value     float64           `json:"value"`
		Labels    map[string]string `json:"labels"`
		Timestamp int64             `json:"timestamp"`
	}

	telemetryLogRequest struct {
		Level     string         `json:"level"`
		Message   string         `json:"message"`
		Fields    map[string]any `json:"fields"`
		Timestamp int64          `json:"timestamp"`
	}

	telemetryRequest struct {
		Context telemetryContextRequest `json:"context"`
		Metrics []json.RawMessage       `json:"metrics"`
		Logs    []json.RawMessage       `json:"logs"`
	}
)

func NewTelemetryHandlers(r *gin.Engine, srv TelemetryUseCase) {
	handler := TelemetryHandler{usecase: srv}
	r.POST("/v1/telemetry", handler.Ingest())
}

// Ingest accepts a mobile telemetry batch (metrics + logs) and forwards it to
// the OTel pipeline asynchronously, answering 202 as soon as the batch is
// structurally valid and within size limits. A malformed individual metric or
// log entry is discarded by the use case rather than rejecting the batch:
// telemetry must never surface as a 4xx because of one bad event.
func (h TelemetryHandler) Ingest() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		if c.Request.ContentLength > domain.MaxTelemetryBodyBytes {
			HandleErr(c, ctx, domain.ErrTelemetryPayloadTooLarge)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, domain.MaxTelemetryBodyBytes)

		var req telemetryRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "invalid telemetry payload"))
			return
		}

		if len(req.Metrics) > domain.MaxTelemetryBatchEvents || len(req.Logs) > domain.MaxTelemetryBatchEvents {
			HandleErr(c, ctx, domain.ErrTelemetryPayloadTooLarge)
			return
		}

		input := usecase.TelemetryBatchInput{
			Context: usecase.TelemetryContextInput{
				AppVersion:     req.Context.AppVersion,
				RuntimeVersion: req.Context.RuntimeVersion,
				UpdateID:       req.Context.UpdateID,
				OS:             req.Context.OS,
				OSVersion:      req.Context.OSVersion,
				SessionID:      req.Context.SessionID,
			},
			Metrics: decodeTelemetryMetrics(req.Metrics),
			Logs:    decodeTelemetryLogs(req.Logs),
		}

		h.usecase.Ingest(ctx, input)

		c.Status(http.StatusAccepted)
	}
}

func decodeTelemetryMetrics(raw []json.RawMessage) []usecase.TelemetryMetricInput {
	metrics := make([]usecase.TelemetryMetricInput, 0, len(raw))
	for _, r := range raw {
		var m telemetryMetricRequest
		if err := json.Unmarshal(r, &m); err != nil {
			continue
		}
		metrics = append(metrics, usecase.TelemetryMetricInput{
			Kind:      m.Kind,
			Name:      m.Name,
			Value:     m.Value,
			Labels:    m.Labels,
			Timestamp: m.Timestamp,
		})
	}
	return metrics
}

func decodeTelemetryLogs(raw []json.RawMessage) []usecase.TelemetryLogInput {
	logs := make([]usecase.TelemetryLogInput, 0, len(raw))
	for _, r := range raw {
		var l telemetryLogRequest
		if err := json.Unmarshal(r, &l); err != nil {
			continue
		}
		logs = append(logs, usecase.TelemetryLogInput{
			Level:     l.Level,
			Message:   l.Message,
			Fields:    l.Fields,
			Timestamp: l.Timestamp,
		})
	}
	return logs
}
