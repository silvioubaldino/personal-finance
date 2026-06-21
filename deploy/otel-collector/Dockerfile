# OpenTelemetry Collector sidecar image for Cloud Run.
#
# Versioned per AyD §5.4: the config is baked into the image (reproducible),
# while the Grafana endpoint/token are injected as env vars from Secret Manager
# at deploy time — never baked in.
#
# We use the OpenTelemetry Collector *contrib* distribution: it ships the
# `googlecloud` exporter as a first-class component (the reason the AyD chose the
# Collector over Grafana Alloy). The Google-built collector
# (us-docker.pkg.dev/cloud-ops-agents-artifacts/google-cloud-opentelemetry-collector/otelcol-google)
# is an equivalent alternative; pin whichever you standardize on.
#
# Pin an explicit version for reproducibility (bump deliberately).
FROM otel/opentelemetry-collector-contrib:0.119.0

# Bake the versioned config. The contrib image reads this path by default
# (default CMD is ["--config", "/etc/otelcol-contrib/config.yaml"]).
COPY config.yaml /etc/otelcol-contrib/config.yaml

# OTLP receivers (no external ingress on Cloud Run; reachable only from the
# app container via localhost). gRPC 4317 / HTTP 4318.
EXPOSE 4317 4318
