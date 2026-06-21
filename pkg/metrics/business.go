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

// Label is a low-cardinality string key/value attached to a business KPI.
// It keeps callers from depending on go.opentelemetry.io/otel/attribute.
type Label struct {
	key   string
	value string
}

// String builds a Label, mirroring the pkg/log field constructors.
func String(key, value string) Label {
	return Label{key: key, value: value}
}

// IncBusiness increments a named business KPI counter by value.
//
// The "biz_" prefix is enforced (prepended when missing) so the OTel Collector
// can route business metrics to Google Cloud Monitoring via a name-prefix
// filter, separately from operational metrics. Counters are created lazily and
// cached by name.
func IncBusiness(ctx context.Context, name string, value int64, labels ...Label) {
	attrs := make([]attribute.KeyValue, len(labels))
	for i, l := range labels {
		attrs[i] = attribute.String(l.key, l.value)
	}

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
