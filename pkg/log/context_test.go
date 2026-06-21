package log

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

func TestTraceFields_NoSpan(t *testing.T) {
	if fields := traceFields(context.Background()); fields != nil {
		t.Errorf("expected no trace fields without a span, got %v", fields)
	}
}

func spanContext(t *testing.T) context.Context {
	t.Helper()
	traceID, _ := trace.TraceIDFromHex("0102030405060708090a0b0c0d0e0f10")
	spanID, _ := trace.SpanIDFromHex("0102030405060708")
	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{TraceID: traceID, SpanID: spanID})
	return trace.ContextWithSpanContext(context.Background(), spanCtx)
}

func TestInfoContext_IncludesSeverityAndTrace(t *testing.T) {
	t.Setenv("GOOGLE_PROJECT_ID", "my-project")

	baseCtx := spanContext(t)
	saida := capturarSaida(func() {
		ctx := Context(baseCtx, New(WithFormat("json")))
		InfoContext(ctx, "mensagem com trace")
	})

	var entry map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(saida)), &entry); err != nil {
		t.Fatalf("failed to parse json log: %v (%q)", err, saida)
	}

	if entry["severity"] != "INFO" {
		t.Errorf("expected severity INFO, got %v", entry["severity"])
	}
	if entry["trace_id"] != "0102030405060708090a0b0c0d0e0f10" {
		t.Errorf("expected trace_id in log, got %v", entry["trace_id"])
	}
	if entry["span_id"] != "0102030405060708" {
		t.Errorf("expected span_id in log, got %v", entry["span_id"])
	}
	want := "projects/my-project/traces/0102030405060708090a0b0c0d0e0f10"
	if entry["logging.googleapis.com/trace"] != want {
		t.Errorf("expected cloud trace field %q, got %v", want, entry["logging.googleapis.com/trace"])
	}
}

func TestErrorContext_SeverityMapping(t *testing.T) {
	saida := capturarSaida(func() {
		ctx := Context(context.Background(), New(WithFormat("json")))
		ErrorContext(ctx, "boom")
	})

	if !strings.Contains(saida, `"severity":"ERROR"`) {
		t.Errorf("expected ERROR severity in output: %s", saida)
	}
}
