# Release notes: native OpenTelemetry support

**Target:** next release
**Feature branch:** `feat/complete-otel-implementation`

This release adds native OpenTelemetry support across tracing, metrics, and structured logging, together with shared exporter configuration and more reliable application shutdown.

## Highlights

### Native OpenTelemetry for all telemetry signals

Gosoline can now export traces, metrics, and logs through OpenTelemetry Protocol (OTLP):

- **Traces** support the existing `otel_http` exporter plus the new `otel_grpc` and `stdout` exporters.
- **Metrics** support a new `otel` writer with periodic OTLP export.
- **Logs** support a new `otel` handler. Structured fields are preserved, and the active trace and span context is attached for trace-to-log correlation.
- All three signals can share the same OpenTelemetry resource, including service name, service namespace, application identity, and custom attributes.

### Shared OTLP exporter configuration

The new `otel` configuration block is shared by the metric, log, and new gRPC trace exporters. It supports:

- OTLP gRPC and HTTP transports
- Explicit `endpoint` or `host`/`port` configuration, including IPv6-safe endpoint construction
- Per-signal HTTP paths
- Static export headers
- Gzip compression
- Export timeouts and retry/backoff settings
- TLS and mutual TLS, including custom CA and client certificates
- Resource attributes with gosoline identity placeholder expansion

Example configuration:

```yaml
otel:
  resource:
    service_name_pattern: "{app.name}"
    service_namespace_pattern: "{app.namespace}"
    attributes:
      deployment.environment: "{app.env}"
  exporter:
    protocol: grpc
    host: localhost
    port: 4317
    insecure: true
    compression: gzip
    timeout: 10s
    retry:
      enabled: true

tracing:
  provider: otel
  otel:
    exporter: otel_grpc
    propagators: [tracecontext, baggage]

metric:
  enabled: true
  writers: [otel]
  writer_settings:
    otel:
      interval: 15s

log:
  handlers:
    otel:
      type: otel
      level: info
```

The legacy `tracing.otel.exporter: otel_http` path and its `tracing.otel.http.*` settings remain available.

### OpenTelemetry metric behavior

The OTEL metric writer:

- Maps gosoline counters, gauges, histograms, and summaries to the corresponding OTEL instruments.
- Converts gosoline units to UCUM-compatible OTEL units.
- Exports dimensions as metric attributes.
- Honors explicit histogram bucket boundaries.
- Uses OTEL semantic-convention naming and shared resource attributes.
- Can run alongside the existing Prometheus writer.

Raw Prometheus and OTEL writers do not aggregate metrics in gosoline; aggregation is still available through writers that support it.

### Lifecycle and shutdown

- Add `application.WithOtelShutdown` to register metric and tracing provider shutdown with the kernel.
- Shutdown handlers run in registration order, continue after individual handler errors, and receive a bounded live context based on `kernel.killTimeout`.
- The root logger now closes resource-owning handlers, including the OTEL log provider, and reports close errors.
- Forced kernel exit remains available when graceful cleanup blocks.

### Examples and verification

The branch adds a runnable `examples/otel` application and OTEL collector configuration, plus OTEL integration coverage for trace, metric, and log export.

## Breaking changes

### `log.GosoLogger` now requires `Close`

`log.GosoLogger` has a new method:

```go
Close(ctx context.Context) error
```

Custom implementations, fakes, and mocks of `log.GosoLogger` must implement `Close`. The logger closes all handlers implementing the new optional `log.ClosingHandler` interface.

### Kernel builders now require `log.GosoLogger`

The logger parameter of the following exported functions changed from `log.Logger` to `log.GosoLogger`:

- `kernel.BuildFactory`
- `kernel.BuildKernel`
- `kernel.NewFactory`

Callers that pass a value typed only as `log.Logger` must provide a `log.GosoLogger` instead.

### `OtelExporterFactory` returns the exporter interface

`tracing.OtelExporterFactory` and `tracing.NewOtelHttpTracer` now return `sdktrace.SpanExporter` instead of the concrete `*otlptrace.Exporter`. Custom exporter factories must update their return type:

```go
func(ctx context.Context, config cfg.Config, logger log.Logger) (sdktrace.SpanExporter, error)
```

### Aggregation is rejected for raw writers

Applications that set either of these options to `true` now fail configuration/startup validation:

```yaml
metric:
  writer_settings:
    prometheus:
      aggregate: true
    otel:
      aggregate: true
```

Remove those settings or set them to `false`. Use a metric writer that supports gosoline-side aggregation when aggregation is required.

### Every gosoline metric is renamed, and the metric schema version is `v2.0`

Gosoline metrics no longer bake identity, units or dimensions into the name. Each metric is authored
once as a canonical namespace plus a leaf, and every writer renders that one name into its own
convention:

| Writer | Rendering of `http.server` + `request.duration` |
|--------|--------------------------------------------------|
| CloudWatch | `HttpServerRequestDuration`, value in milliseconds |
| Prometheus | `<app>_http_server_request_duration_seconds`, value in seconds |
| OTEL | `http.server.request.duration`, unit `s` on the instrument |

Dimension keys follow the same rule: they are authored canonically and the Prometheus writer renders
them into valid label names, so `http.route` is exported as the label `http_route`.

`metric.SchemaVersion` is `v2.0`. Read it from the metadata document under `metric.schema_version` to
decide which names to expect - there is no overlap window and no dual emission, so alarms, dashboards
and queries keyed on a former name break at this release by design.

Further consequences of the rename:

- **Durations are seconds in Prometheus and OTEL.** CloudWatch keeps milliseconds. Call sites still
  record milliseconds; the writers scale.
- **Prometheus summaries became histograms.** Quantiles move from summary quantiles to
  `histogram_quantile` over buckets.
- **`UnitMillisecondsAverage` and its eight siblings now resolve to their base unit** before the unit
  is rendered, so millisecond histograms no longer report UCUM `1`.
- **The four Kafka byte metrics changed from a count unit to bytes**, which is what they always
  measured.
- **`metric.Datum` gained a `Namespace` field** and `metric.NewWriter` gained a leading `namespace`
  parameter. Application code that writes its own metrics passes `metric.NewWriter("")` to keep its
  names unchanged.

#### Final canonical inventory (schema `v2.0`)

This is the complete final framework-authored inventory. It is grouped by canonical namespace; the
backend writers apply their rendering rules after these names are authored.

| Namespace | Canonical leaves |
|---|---|
| `autoscaling.per_runner` | `stream.messages`; `http.server.requests` |
| `blob` | `batch.operations` |
| `cloud.aws.kinesis` | `reads`; `consume.errors`; `lag`; `acquire.delay`; `sleep.duration`; `wait.duration`; `shard.count`; `client.count`; `batch.size`; `send.errors` |
| `conc.scheduler` | `batch.size`; `task.delay` |
| `db.client` | `connection.count`; `connections` |
| `db.repo` | `operation.duration`; `model_event.notifications` |
| `ddb` | `operation.duration` |
| `http.client` | `request.duration` |
| `http.server` | `request.duration`; `rejected.requests`; `active_requests`; `connection.count` |
| `kafka` | `connects`; `throttles`; `throttle.duration`; `produce.batch.records`; `produce.batch.size`; `produce.batch.compressed.size`; `fetch.batch.records`; `fetch.batch.size`; `fetch.batch.compressed.size` |
| `kafka.consumer` | `polls`; `poll.duration`; `commit.duration`; `commit.errors`; `wait.duration`; `rebalances`; `consume.errors` |
| `kafka.producer` | `batch.size`; `send.errors` |
| `kvstore` | `reads`; `writes`; `deletes`; `hits`; `item.count` |
| `limit` | `rate_limit.takes`; `rate_limit.releases`; `rate_limit.throttles`; `rate_limit.errors` |
| `mdlsub` | `consumed.events`; `skipped.events`; `consume.errors` |
| `messaging` | `process.duration`; `client.consumed.messages`; `client.sent.messages`; `client.operation.duration` |
| `metric` | `records` |
| `rpc.server` | `duration` |
| `smpl` | `decisions` |
| `stream` | `errors`; `retry.operations`; `messages`; `batch.size`; `aggregate.size`; `idle.duration`; `available.messages`; `sent.messages`; `message.count`; `reads`; `writes` |

#### Focused contract update status

`metric.SchemaVersion` remains **`v2.0`** by explicit maintainer direction. This is an exception to
the normal version-increment policy: do not change the marker to `v3.0` for this focused revision.
There is still no dual emission.

| Status | Metric or behavior | Consumer action |
|---|---|---|
| Unchanged | Every final inventory entry not called out below | Keep using its canonical namespace and leaf. Moving namespace constants into emitting packages changes source ownership, not those emitted names. |
| Renamed | SQL repository operations: `db.client.operation.duration` → `db.repo.operation.duration` | Re-key SQL repository dashboards, alerts, and queries to `db.repo`. |
| Renamed | DynamoDB repository operations: `db.client.operation.duration` → `ddb.operation.duration` | Re-key DynamoDB repository dashboards, alerts, and queries to `ddb`. |
| Deleted and consolidated | `db.repo.model_event.notify.errors` | Use `db.repo.model_event.notifications` for both outcomes; the former error-only metric is not emitted. |
| Added outcome coverage | A cancelled HTTP client request | `http.client.request.duration` is emitted with `error.type=metric.ErrorType(context.Canceled)` before the original cancellation error is returned. |
| Deleted aggregate series | HTTP server-only, Kafka topic-only, and Kinesis stream-only aggregate data | Query or aggregate the retained detailed series in the backend rather than searching for a framework-emitted `KindTotal` datum. |

##### DB-repository notification outcomes

The single counter is `db.repo.model_event.notifications`, with `UnitCount` and `KindCounter` for
both paths. There is no `success` label.

- Successful notification selector: ``db.repo.model_event.notifications{error.type="{{default}}"}``.
- Failed notification selector: ``db.repo.model_event.notifications{error.type!="{{default}}"}``, or
  filter for the concrete normalized `metric.ErrorType(err)` value.

This consolidation replaces—not supplements—`db.repo.model_event.notify.errors`.

##### HTTP cancellation duration

If a client request ends with an error matching `context.Canceled`, gosoline records the total request
duration under `http.client.request.duration` before returning that original error. The cancellation
outcome is normalized as `metric.ErrorType(context.Canceled)`, rather than using a transport wrapper's
type, and retains the normal default response-status dimension. Cancellations remain excluded from
application-error logging.

##### Aggregate-query guidance

Gosoline no longer produces redundant aggregate `KindTotal` data:

- HTTP server middleware emits the route/method/status duration and route/method rejection series
  only; aggregate across those dimensions in the backend for server-level views.
- Kafka emits its topic-and-partition series only; aggregate topic partitions in the backend.
- Kinesis emits its stream-and-shard series only; aggregate stream shards in the backend.

Prometheus may still render counter instruments with its conventional `_total` suffix. That naming
rule is distinct from the removed framework-authored `KindTotal` aggregate datum.

#### Dimension-key policy

Every key an OpenTelemetry semantic convention defines remains spelled the way that convention
spells it; all existing custom dimensions remain unchanged in this focused revision. Additional
semantic-convention attributes may be added where appropriate. Future custom metric attributes must
use a unique owned prefix such as `gosoline.*`; do not rename existing custom keys solely to apply
that future convention.

| Former key | Canonical key |
|---|---|
| `Consumer` (stream) | `stream.consumer.name` |
| `ProducerDaemon` | `stream.producer.name` |
| `Scheduler` | `scheduler.name` |
| `ModelId` | `model.id` |
| `model`, `store` (kvstore) | `model.id`, `store.type` |
| `Operation` (blob) | `blob.operation` |
| `Operation` (db, ddb) | `db.operation.name` |
| `Type` (db connections) | `db.client.connection.state`, values `used` and `idle` |
| `StreamName`, `Topic` | `messaging.destination.name` |
| `ShardId`, `Partition` | `messaging.destination.partition.id` |
| `ClientType`, `Client`, `Broker` | `kafka.client.type`, `kafka.client.name`, `kafka.broker.address` |
| `Method`, `Path`, `ServerName` | `http.request.method`, `http.route`, `http.server.name` |
| `full_method` | `rpc.service` and `rpc.method` |
| `trace_id`, `name`, `prefix` (limit) | `trace.id`, `limit.name`, `limit.prefix` |
| `sampled` | `sampling.sampled` |

## Upgrade checklist

1. Update custom `log.GosoLogger` implementations with `Close(context.Context) error`.
2. Update direct calls to the kernel builder functions to pass a `log.GosoLogger`.
3. Update custom `tracing.OtelExporterFactory` functions to return `sdktrace.SpanExporter`.
4. Remove `aggregate: true` from Prometheus or OTEL writer settings.
5. Add `application.WithOtelShutdown` when OTEL trace and/or metric providers must be flushed during application shutdown.
6. Pass a namespace to `metric.NewWriter`; use `metric.NewWriter("")` to keep application-authored metric names unchanged.
7. Re-key every alarm, dashboard, query, and metric-calculator configuration using the final inventory above; select by `metric.schema_version` (`v2.0` for this release).
8. Re-key SQL repository operation telemetry to `db.repo.operation.duration` and DynamoDB repository operation telemetry to `ddb.operation.duration`.
9. Replace notification-failure queries with `db.repo.model_event.notifications` filtered by `error.type`, and aggregate detailed HTTP, Kafka, and Kinesis series in the backend.
10. Update ECS scaling policies that reference `PerRunner*`, `StreamMessages`, `HttpServerRequests` or `ShardTaskRatio`.
