// Package metrics wires up the OpenTelemetry metrics SDK and exposes small,
// idiomatic helpers for the rest of the application: an HTTP server middleware
// (http.go) and a business KPI helper (business.go).
//
// The OTLP HTTP exporter ships metrics to a local OpenTelemetry Collector
// sidecar (see deploy/otel-collector/), which fans them out to Grafana Cloud
// (operational metrics) and Google Cloud Monitoring (business metrics).
package metrics

import (
	"context"
	"os"
	"time"

	"personal-finance/internal/bootstrap/environment"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

const (
	serviceName           = "personal-finance"
	defaultOTLPEndpoint   = "http://localhost:4318"
	defaultExportInterval = 60 * time.Second
)

// InitMeterProvider initializes the global OTel MeterProvider.
//
// Behavior is controlled by environment variables:
//   - OTEL_SDK_DISABLED=true installs a no-op provider and returns immediately.
//   - OTEL_EXPORTER_OTLP_ENDPOINT overrides the collector endpoint
//     (default http://localhost:4318).
//   - SERVICE_VERSION / K_REVISION provide service.version.
//   - ENVIRONMENT provides deployment.environment.
//
// The returned shutdown func flushes and stops the exporter and must be
// deferred by the caller.
func InitMeterProvider(ctx context.Context) (func(context.Context) error, error) {
	if os.Getenv("OTEL_SDK_DISABLED") == "true" {
		// No-op provider: instruments still work, they just record nothing.
		otel.SetMeterProvider(noopmetric.NewMeterProvider())
		return func(context.Context) error { return nil }, nil
	}

	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = defaultOTLPEndpoint
	}

	exporter, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpointURL(endpoint),
	)
	if err != nil {
		return nil, err
	}

	res, err := buildResource(ctx)
	if err != nil {
		return nil, err
	}

	provider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(metric.NewPeriodicReader(
			exporter,
			metric.WithInterval(defaultExportInterval),
		)),
	)

	otel.SetMeterProvider(provider)

	return provider.Shutdown, nil
}

func buildResource(ctx context.Context) (*resource.Resource, error) {
	version := os.Getenv("SERVICE_VERSION")
	if version == "" {
		version = os.Getenv("K_REVISION")
	}
	if version == "" {
		version = "unknown"
	}

	return resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(version),
			semconv.DeploymentEnvironment(environment.GetEnvironment()),
		),
	)
}
