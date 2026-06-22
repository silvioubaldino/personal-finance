package usecase

import (
	"context"
	"math"

	"personal-finance/internal/plataform/authentication"
	"personal-finance/pkg/log"
	"personal-finance/pkg/metrics"
)

const mobileServiceName = "personal-finance-mobile"

// telemetryReservedLabelKeys are identifiers that must never become a metric
// label (cardinality guard-rail). The mobile client should not send them as
// metric labels, but we strip them defensively rather than trust the client.
var telemetryReservedLabelKeys = map[string]struct{}{
	"user_id":    {},
	"session_id": {},
	"update_id":  {},
	"device_id":  {},
}

// --- Input DTOs ---

type TelemetryContextInput struct {
	AppVersion     string
	RuntimeVersion string
	UpdateID       string
	OS             string
	OSVersion      string
	SessionID      string
}

type TelemetryMetricInput struct {
	Kind      string
	Name      string
	Value     float64
	Labels    map[string]string
	Timestamp int64
}

type TelemetryLogInput struct {
	Level     string
	Message   string
	Fields    map[string]any
	Timestamp int64
}

type TelemetryBatchInput struct {
	Context TelemetryContextInput
	Metrics []TelemetryMetricInput
	Logs    []TelemetryLogInput
}

// --- Use Case ---

type TelemetryUseCase struct{}

func NewTelemetryUseCase() *TelemetryUseCase {
	return &TelemetryUseCase{}
}

// Ingest translates a mobile telemetry batch into the existing OTel pipeline.
// It returns immediately so the caller can answer 202 without waiting on the
// Collector: processing runs in a goroutine on a context detached from the
// request (which gets canceled once the response is written), while keeping
// the logger/trace correlation already attached to it. A malformed individual
// event is skipped; it never fails the whole batch.
func (u *TelemetryUseCase) Ingest(ctx context.Context, input TelemetryBatchInput) {
	userID := authentication.UserIDFromContext(ctx)
	detached := context.WithoutCancel(ctx)

	go u.process(detached, userID, input)
}

func (u *TelemetryUseCase) process(ctx context.Context, userID string, input TelemetryBatchInput) {
	for _, m := range input.Metrics {
		recordTelemetryMetric(ctx, m)
	}
	for _, l := range input.Logs {
		recordTelemetryLog(ctx, userID, input.Context, l)
	}
}

func recordTelemetryMetric(ctx context.Context, m TelemetryMetricInput) {
	if m.Name == "" || math.IsNaN(m.Value) || math.IsInf(m.Value, 0) {
		return
	}

	labels := make([]metrics.Label, 0, len(m.Labels))
	for k, v := range m.Labels {
		if _, reserved := telemetryReservedLabelKeys[k]; reserved {
			continue
		}
		labels = append(labels, metrics.String(k, v))
	}

	switch m.Kind {
	case "counter":
		metrics.IncMobileCounter(ctx, m.Name, int64(m.Value), labels...)
	case "histogram":
		metrics.RecordMobileHistogram(ctx, m.Name, m.Value, labels...)
	}
}

func recordTelemetryLog(ctx context.Context, userID string, tctx TelemetryContextInput, l TelemetryLogInput) {
	if l.Message == "" {
		return
	}

	fields := []log.Field{
		log.String("service", mobileServiceName),
		log.String("app_version", tctx.AppVersion),
		log.String("runtime_version", tctx.RuntimeVersion),
		log.String("os", tctx.OS),
		log.String("os_version", tctx.OSVersion),
		log.String("session_id", tctx.SessionID),
	}
	if tctx.UpdateID != "" {
		fields = append(fields, log.String("update_id", tctx.UpdateID))
	}
	if userID != "" {
		fields = append(fields, log.String("user_id", userID))
	}
	for k, v := range l.Fields {
		fields = append(fields, log.Any(k, v))
	}

	switch l.Level {
	case "error":
		log.ErrorContext(ctx, l.Message, fields...)
	case "warn":
		log.WarnContext(ctx, l.Message, fields...)
	case "info":
		log.InfoContext(ctx, l.Message, fields...)
	}
}
