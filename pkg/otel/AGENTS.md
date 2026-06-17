# OTEL Package Agent Guide

## Scope
- Shared OpenTelemetry core used by `pkg/tracing`, `pkg/metric`, and `pkg/log`.
- Builds the OTEL **resource** from the application identity (so all signals correlate).
- Builds OTLP **exporters** (gRPC/HTTP) for traces, metrics, and logs, with TLS/mTLS support.
- Driven entirely by the native gosoline `cfg` system under the `otel` root key.

## Key files
- `settings.go` — `Settings`, `ResourceSettings`, `ExporterSettings` (incl. `Address()` host/port vs endpoint), `TLSSettings`, `RetrySettings`.
- `resource.go` — `BuildResource` / `ProvideResource` (identity → resource attributes).
- `exporter.go` — `BuildTraceExporter`, `BuildMetricExporter`, `BuildLogExporter` + TLS/mTLS config builder.

## Configuration
A single `otel` block is shared by all three signals. Per-signal toggles live in their own packages
(`tracing.otel`, `metric.writer_settings.otel`, `log.handlers.<name>`).

```yaml
otel:
  resource:
    service_name_pattern: "{app.name}"            # -> service.name
    service_namespace_pattern: "{app.namespace}"  # -> service.namespace
    delimiter: "-"
    attributes:                                   # extra resource attributes (values may use placeholders)
      deployment.environment: "{app.env}"
      organization: "acme"
  exporter:
    protocol: grpc        # grpc | http
    host: "localhost"     # override via env from pod metadata (status.hostIP)
    port: 4317
    endpoint: ""          # optional full override; wins over host:port
    url_path: ""          # http only; shared fallback for all signals
    traces_url_path: ""   # http only; per-signal override for traces (e.g. /otel/v1/traces)
    metrics_url_path: ""  # http only; per-signal override for metrics (e.g. /otel/v1/metrics)
    logs_url_path: ""     # http only; per-signal override for logs (e.g. /otel/v1/logs)
    insecure: true        # set false to enable TLS/mTLS
    compression: gzip
    timeout: 10s
    headers: {}           # static headers (auth, tenant, ...)
    tls:                  # used when insecure=false
      ca_file: ""
      cert_file: ""       # client cert (mTLS)
      key_file: ""        # client key (mTLS)
      server_name: ""
      insecure_skip_verify: false
      min_version: "1.3"  # minimum TLS version (1.0, 1.1, 1.2, 1.3)
    retry:
      enabled: true
      initial_interval: 5s
      max_interval: 30s
      max_elapsed_time: 300s
```

### Host/port injection (Kubernetes)
The host is split from the port so only the host needs to be injected from pod metadata:

```yaml
env:
  - name: NODE_IP
    valueFrom: { fieldRef: { fieldPath: status.hostIP } }
  - name: OTEL_EXPORTER_HOST   # -> otel.exporter.host
    value: "$(NODE_IP)"
```

`Address()` composes the endpoint with `net.JoinHostPort` (correct IPv6 bracketing) unless
`endpoint` is set explicitly.

## Design notes
- **Resource is identical across signals** — identity (service name/namespace, environment, extra
  attributes) lives in resource attributes, never in metric or span names. This is what enables
  trace ↔ metric ↔ log correlation by resource attributes.
- `pkg/otel` intentionally does **not** import `appctx` to avoid an import cycle
  (`log → otel → appctx → conc → exec → log`). Each signal provider builds the resource once at
  startup; the attribute values are identical.

## Common tasks
- Add a new exporter knob: extend `ExporterSettings` and thread it through the three `Build*Exporter` functions.
- Add a resource attribute: prefer config via `otel.resource.attributes`; only add a semconv attribute in `BuildResource` if it is universal.
