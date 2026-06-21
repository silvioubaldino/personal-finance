package log

import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/otel/trace"
)

type logCtxKey struct{}

func Context(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, logCtxKey{}, logger)
}

func FromContext(ctx context.Context) Logger {
	if ctx == nil {
		return Global
	}

	l, ok := ctx.Value(logCtxKey{}).(Logger)
	if ok {
		return l
	}
	return Global
}

// traceFields returns the trace correlation fields for the active OTel span in
// ctx, if any. When no span context is present it returns nil so callers can
// safely spread the result without any extra branching. The
// "logging.googleapis.com/trace" field lets Cloud Logging correlate logs with
// Cloud Trace spans.
func traceFields(ctx context.Context) []Field {
	if ctx == nil {
		return nil
	}

	spanCtx := trace.SpanContextFromContext(ctx)
	if !spanCtx.IsValid() {
		return nil
	}

	traceID := spanCtx.TraceID().String()
	spanID := spanCtx.SpanID().String()

	fields := []Field{
		String("trace_id", traceID),
		String("span_id", spanID),
	}

	if projectID := os.Getenv("GOOGLE_PROJECT_ID"); projectID != "" {
		fields = append(fields, String(
			"logging.googleapis.com/trace",
			fmt.Sprintf("projects/%s/traces/%s", projectID, traceID),
		))
	} else {
		fields = append(fields, String("logging.googleapis.com/trace", traceID))
	}

	return fields
}

func withTrace(ctx context.Context, fields []Field) []Field {
	tf := traceFields(ctx)
	if len(tf) == 0 {
		return fields
	}
	return append(fields, tf...)
}

func DebugContext(ctx context.Context, msg string, fields ...Field) {
	FromContext(ctx).Debug(msg, withTrace(ctx, fields)...)
}

func InfoContext(ctx context.Context, msg string, fields ...Field) {
	FromContext(ctx).Info(msg, withTrace(ctx, fields)...)
}

func WarnContext(ctx context.Context, msg string, fields ...Field) {
	FromContext(ctx).Warn(msg, withTrace(ctx, fields)...)
}

func ErrorContext(ctx context.Context, msg string, fields ...Field) {
	FromContext(ctx).Error(msg, withTrace(ctx, fields)...)
}

func FatalContext(ctx context.Context, msg string, fields ...Field) {
	FromContext(ctx).Fatal(msg, withTrace(ctx, fields)...)
}
