package metrics

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const mobileMeterName = "personal-finance/mobile"

var (
	mobileInstrumentsMu sync.Mutex
	mobileCounters      = map[string]metric.Int64Counter{}
	mobileHistograms    = map[string]metric.Float64Histogram{}
)

// IncMobileCounter increments a counter metric reported by the mobile app.
// Unlike IncBusiness, the name is used as-is (the mobile client already
// prefixes it with "app_" or "biz_") so the Collector's existing
// prefix-based routing keeps applying. Instruments are created lazily and
// cached by name.
func IncMobileCounter(ctx context.Context, name string, value int64, labels ...Label) {
	mobileCounter(name).Add(ctx, value, metric.WithAttributes(toAttributes(labels)...))
}

// RecordMobileHistogram records a histogram measurement reported by the
// mobile app. See IncMobileCounter for the naming convention.
func RecordMobileHistogram(ctx context.Context, name string, value float64, labels ...Label) {
	mobileHistogram(name).Record(ctx, value, metric.WithAttributes(toAttributes(labels)...))
}

func toAttributes(labels []Label) []attribute.KeyValue {
	attrs := make([]attribute.KeyValue, len(labels))
	for i, l := range labels {
		attrs[i] = attribute.String(l.key, l.value)
	}
	return attrs
}

func mobileCounter(name string) metric.Int64Counter {
	mobileInstrumentsMu.Lock()
	defer mobileInstrumentsMu.Unlock()

	if counter, ok := mobileCounters[name]; ok {
		return counter
	}

	meter := otel.GetMeterProvider().Meter(mobileMeterName)
	counter, _ := meter.Int64Counter(name)
	mobileCounters[name] = counter
	return counter
}

func mobileHistogram(name string) metric.Float64Histogram {
	mobileInstrumentsMu.Lock()
	defer mobileInstrumentsMu.Unlock()

	if histogram, ok := mobileHistograms[name]; ok {
		return histogram
	}

	meter := otel.GetMeterProvider().Meter(mobileMeterName)
	histogram, _ := meter.Float64Histogram(name)
	mobileHistograms[name] = histogram
	return histogram
}
