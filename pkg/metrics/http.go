package metrics

import (
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const meterName = "personal-finance/http"

var (
	httpInstrumentsOnce sync.Once

	httpRequestsTotal   metric.Int64Counter
	httpRequestDuration metric.Float64Histogram
	httpActiveRequests  metric.Int64UpDownCounter
)

// initHTTPInstruments creates the HTTP server instruments exactly once, after
// the global MeterProvider has been configured.
func initHTTPInstruments() {
	httpInstrumentsOnce.Do(func() {
		meter := otel.GetMeterProvider().Meter(meterName)

		httpRequestsTotal, _ = meter.Int64Counter(
			"http_server_requests_total",
			metric.WithDescription("Total number of HTTP requests handled."),
		)
		httpRequestDuration, _ = meter.Float64Histogram(
			"http_server_request_duration_seconds",
			metric.WithDescription("HTTP request duration in seconds."),
			metric.WithUnit("s"),
		)
		httpActiveRequests, _ = meter.Int64UpDownCounter(
			"http_server_active_requests",
			metric.WithDescription("Number of in-flight HTTP requests."),
		)
	})
}

// HTTPMetricsMiddleware records request count, duration and active-request
// gauges per matched route. It uses c.FullPath() (the route template, e.g.
// "/movements/:id") rather than the raw URL path to keep label cardinality
// bounded; unmatched routes are bucketed under route="unmatched".
func HTTPMetricsMiddleware() gin.HandlerFunc {
	initHTTPInstruments()

	return func(c *gin.Context) {
		ctx := c.Request.Context()
		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}

		routeAttr := attribute.String("route", route)
		methodAttr := attribute.String("method", c.Request.Method)

		httpActiveRequests.Add(ctx, 1, metric.WithAttributes(routeAttr))
		start := time.Now()

		c.Next()

		elapsed := time.Since(start).Seconds()
		httpActiveRequests.Add(ctx, -1, metric.WithAttributes(routeAttr))

		statusAttr := attribute.String("status_class", statusClass(c.Writer.Status()))

		httpRequestsTotal.Add(ctx, 1, metric.WithAttributes(methodAttr, routeAttr, statusAttr))
		httpRequestDuration.Record(ctx, elapsed, metric.WithAttributes(methodAttr, routeAttr))
	}
}

// statusClass buckets an HTTP status code into its class (2xx, 3xx, ...) to
// avoid one time series per individual status code.
func statusClass(status int) string {
	if status < 100 || status >= 600 {
		return "unknown"
	}
	return strconv.Itoa(status/100) + "xx"
}
