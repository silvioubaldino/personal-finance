package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"personal-finance/pkg/log"

	"github.com/stretchr/testify/assert"
)

// captureStdout swaps os.Stdout for the duration of f, which must build any
// logger it needs inside the closure: zapLogger captures the write syncer at
// construction time, so redirecting stdout after the logger exists has no
// effect on where it writes.
func captureStdout(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

type fakeLogEntry struct {
	level  string
	msg    string
	fields []log.Field
}

type fakeLogger struct {
	mu      sync.Mutex
	entries []fakeLogEntry
}

func (f *fakeLogger) record(level, msg string, fields []log.Field) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.entries = append(f.entries, fakeLogEntry{level: level, msg: msg, fields: fields})
}

func (f *fakeLogger) Debug(msg string, fields ...log.Field) { f.record("debug", msg, fields) }
func (f *fakeLogger) Info(msg string, fields ...log.Field)  { f.record("info", msg, fields) }
func (f *fakeLogger) Warn(msg string, fields ...log.Field)  { f.record("warn", msg, fields) }
func (f *fakeLogger) Error(msg string, fields ...log.Field) { f.record("error", msg, fields) }
func (f *fakeLogger) Fatal(msg string, fields ...log.Field) { f.record("fatal", msg, fields) }
func (f *fakeLogger) With(_ ...log.Field) log.Logger        { return f }

func TestRecordTelemetryLog(t *testing.T) {
	tctx := TelemetryContextInput{
		AppVersion:     "1.1.0",
		RuntimeVersion: "54.0.0",
		UpdateID:       "update-1",
		OS:             "ios",
		OSVersion:      "17.4",
		SessionID:      "session-1",
	}

	t.Run("routes error level to Error and includes context/user fields in the emitted log", func(t *testing.T) {
		saida := captureStdout(func() {
			ctx := log.Context(context.Background(), log.New(log.WithFormat("json")))
			recordTelemetryLog(ctx, "user-123", tctx, TelemetryLogInput{
				Level:   "error",
				Message: "boom",
				Fields:  map[string]any{"stack": "trace..."},
			})
		})

		var entry map[string]interface{}
		assert.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(saida)), &entry))

		assert.Equal(t, "ERROR", entry["severity"])
		assert.Equal(t, "boom", entry["message"])
		assert.Equal(t, "personal-finance-mobile", entry["service"])
		assert.Equal(t, "user-123", entry["user_id"])
		assert.Equal(t, "session-1", entry["session_id"])
		assert.Equal(t, "update-1", entry["update_id"])
		assert.Equal(t, "trace...", entry["stack"])
	})

	t.Run("routes warn and info levels accordingly", func(t *testing.T) {
		fl := &fakeLogger{}
		ctx := log.Context(context.Background(), fl)

		recordTelemetryLog(ctx, "", tctx, TelemetryLogInput{Level: "warn", Message: "careful"})
		recordTelemetryLog(ctx, "", tctx, TelemetryLogInput{Level: "info", Message: "fyi"})

		assert.Len(t, fl.entries, 2)
		assert.Equal(t, "warn", fl.entries[0].level)
		assert.Equal(t, "info", fl.entries[1].level)
	})

	t.Run("drops events with empty message or unknown level", func(t *testing.T) {
		fl := &fakeLogger{}
		ctx := log.Context(context.Background(), fl)

		recordTelemetryLog(ctx, "", tctx, TelemetryLogInput{Level: "error", Message: ""})
		recordTelemetryLog(ctx, "", tctx, TelemetryLogInput{Level: "trace", Message: "ignored"})

		assert.Empty(t, fl.entries)
	})

	t.Run("omits user_id and update_id fields when absent", func(t *testing.T) {
		saida := captureStdout(func() {
			ctx := log.Context(context.Background(), log.New(log.WithFormat("json")))
			recordTelemetryLog(ctx, "", TelemetryContextInput{}, TelemetryLogInput{Level: "info", Message: "no extras"})
		})

		var entry map[string]interface{}
		assert.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(saida)), &entry))

		_, ok := entry["user_id"]
		assert.False(t, ok)
		_, ok = entry["update_id"]
		assert.False(t, ok)
	})
}

func TestRecordTelemetryMetric_DoesNotPanicOnInvalidInput(t *testing.T) {
	ctx := context.Background()

	cases := []TelemetryMetricInput{
		{Kind: "counter", Name: ""},
		{Kind: "counter", Name: "app_x", Value: math.NaN()},
		{Kind: "counter", Name: "app_x", Value: math.Inf(1)},
		{Kind: "unknown_kind", Name: "app_x", Value: 1},
		{
			Kind: "counter", Name: "app_x", Value: 1,
			Labels: map[string]string{"user_id": "should-be-stripped", "reason": "timeout"},
		},
		{Kind: "histogram", Name: "app_duration_seconds", Value: 0.5},
	}

	for _, c := range cases {
		assert.NotPanics(t, func() {
			recordTelemetryMetric(ctx, c)
		})
	}
}

func TestTelemetryUseCase_Ingest_ProcessesAsynchronously(t *testing.T) {
	fl := &fakeLogger{}
	ctx := log.Context(context.Background(), fl)

	u := NewTelemetryUseCase()
	u.Ingest(ctx, TelemetryBatchInput{
		Logs: []TelemetryLogInput{{Level: "info", Message: "hello"}},
	})

	assert.Eventually(t, func() bool {
		fl.mu.Lock()
		defer fl.mu.Unlock()
		return len(fl.entries) == 1
	}, time.Second, time.Millisecond, "expected the log entry to be processed asynchronously")
}
