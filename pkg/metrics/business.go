package metrics

import (
	"context"
	"strings"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	businessMeterName = "personal-finance/business"
	businessPrefix    = "biz_"
)

var (
	businessCountersMu sync.Mutex
	businessCounters   = map[string]metric.Int64Counter{}
)

// BusinessCounter increments a named business KPI counter by value.
//
// The "biz_" prefix is enforced (prepended when missing) so the OTel Collector
// can route business metrics to Google Cloud Monitoring via a name-prefix
// filter, separately from operational metrics. Counters are created lazily and
// cached by name.
func BusinessCounter(ctx context.Context, name string, value int64, attrs ...attribute.KeyValue) {
	counter := businessCounter(name)
	counter.Add(ctx, value, metric.WithAttributes(attrs...))
}

func businessCounter(name string) metric.Int64Counter {
	name = withBizPrefix(name)

	businessCountersMu.Lock()
	defer businessCountersMu.Unlock()

	if counter, ok := businessCounters[name]; ok {
		return counter
	}

	meter := otel.GetMeterProvider().Meter(businessMeterName)
	counter, _ := meter.Int64Counter(name)
	businessCounters[name] = counter
	return counter
}

func withBizPrefix(name string) string {
	if strings.HasPrefix(name, businessPrefix) {
		return name
	}
	return businessPrefix + name
}
