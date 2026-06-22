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

// IncAITokens records LLM token usage on the unified biz_ai_tokens_total
// counter, split by kind ("input"/"output"). feature identifies the call site
// (e.g. "agent", "statement_extract", "statement_classify") and model is the
// LLM model name. All three labels are low-cardinality, so the series count
// stays bounded (features × kinds × models). Non-positive counts are skipped so
// a kind that produced no tokens does not create an empty series.
func IncAITokens(ctx context.Context, feature, model string, inputTokens, outputTokens int) {
	if inputTokens > 0 {
		IncBusiness(ctx, "biz_ai_tokens_total", int64(inputTokens),
			String("feature", feature),
			String("kind", "input"),
			String("model", model),
		)
	}
	if outputTokens > 0 {
		IncBusiness(ctx, "biz_ai_tokens_total", int64(outputTokens),
			String("feature", feature),
			String("kind", "output"),
			String("model", model),
		)
	}
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
