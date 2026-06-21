# OpenTelemetry Collector sidecar

This directory holds the versioned configuration for the OpenTelemetry Collector
that runs as a **Cloud Run sidecar** alongside the personal-finance API. It is
the observability foundation decided in the AyD (§5.4).

## What it does

The API container exports OTLP metrics over HTTP to the collector on
`localhost:4318` (the collector also listens on `:4317` for gRPC). The collector
has **no external ingress** — it is only reachable from the API container in the
same Cloud Run service.

The collector splits metrics by name prefix and routes them to two backends:

| Metrics                         | Pipeline           | Backend                  |
| ------------------------------- | ------------------ | ------------------------ |
| operational (everything else)   | `metrics/health`   | Grafana Cloud (OTLP/HTTP) |
| business (`biz_*` prefix)       | `metrics/business` | Google Cloud Monitoring  |

Splitting is done with the `filter` processor using a regex on the metric name
(`^biz_.*`).

## How it ships

1. The collector image is built from a tag via **Cloud Build**; `config.yaml` in
   this directory is **baked into the image** at build time.
2. The image is deployed as a **Cloud Run sidecar** container (no ingress) next
   to the API.
3. The Grafana endpoint and token are provided as environment variables,
   sourced from **Secret Manager** at deploy time:
   - `GRAFANA_OTLP_ENDPOINT`
   - `GRAFANA_OTLP_TOKEN` (full `Authorization` header value, e.g. `Bearer <id>:<token>`)

   These are never hardcoded in the config or the image.
4. Google Cloud Monitoring export uses the Cloud Run service account; the GCP
   project is auto-detected via the `resourcedetection` processor.

## Environment variables

| Variable                | Source         | Purpose                                 |
| ----------------------- | -------------- | --------------------------------------- |
| `GRAFANA_OTLP_ENDPOINT` | Secret Manager | Grafana Cloud OTLP/HTTP endpoint        |
| `GRAFANA_OTLP_TOKEN`    | Secret Manager | `Authorization` header for Grafana Cloud |

## Local note

The API defaults to `OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318`. Set
`OTEL_SDK_DISABLED=true` to disable metric export entirely (e.g. local dev
without a collector running).
