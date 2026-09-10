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
| `blob` | `batch.operations` |
| `cloud.aws.kinesis` | `consumed.messages`; `sent.messages`; `process.duration`; `reads`; `lag`; `acquire.duration`; `sleep.duration`; `wait.duration`; `shard.count`; `client.count`; `batch.records` |
| `conc.scheduler` | `batch.tasks`; `task.queue.duration` |
| `db.client` | `connection.count`; `connections` |
| `db.repo` | `operation.duration`; `model_event.notifications` |
| `ddb` | `operation.duration` |
| `http.client` | `request.duration` |
| `http.server` | `request.duration`; `rejected.requests`; `active_request.count`; `connection.count` |
| `kafka` | `connects`; `throttles`; `throttle.duration`; `produce.batch.records`; `produce.batch.size`; `produce.batch.compressed.size`; `fetch.batch.records`; `fetch.batch.size`; `fetch.batch.compressed.size` |
| `kafka.consumer` | `consumed.messages`; `process.duration`; `polls`; `poll.duration`; `commit.duration`; `wait.duration`; `rebalances`; `consume.errors` |
| `kafka.producer` | `sent.messages`; `produce.duration`; `batch.records` |
| `kvstore` | `reads`; `writes`; `deletes`; `item.count` |
| `limit` | `takes` |
| `mdlsub` | `events` |
| `metric` | `log.records` |
| `rpc.server` | `request.duration` |
| `smpl` | `decisions` |
| `stream` | `consumed.messages`; `process.duration`; `retry.operations`; `produced.messages`; `batch.messages`; `aggregate.messages`; `idle.duration`; `message.count`; `reads`; `writes` |

#### Focused contract update status

`metric.SchemaVersion` is **`v2.0`**: this revision is one contract change, published under one
version, rather than a version increment per edit inside it. There is still no dual emission.

| Status | Metric or behavior | Consumer action |
|---|---|---|
| Unchanged | Every final inventory entry not called out below | Keep using its canonical namespace and leaf. Moving namespace constants into emitting packages changes source ownership, not those emitted names. |
| Deleted | `autoscaling.per_runner.stream.messages`; `autoscaling.per_runner.http.server.requests` | Remove the per-runner calculator configuration and replace scaling policies with application-specific signals; the calculator and both handlers are no longer present. |
| Deleted | `stream.available.messages`; `stream.sent.messages` | Their only emitter was the messages-per-runner handler, so nothing has written them since it was removed; drop them from dashboards and alerts. |
| Renamed | Log records: `metric.records` → `metric.log.records` | Re-key log-volume dashboards and alerts. The leaf now says what is counted, so it cannot be mistaken for the metric records the daemon writes. |
| Renamed | SQL repository operations: `db.client.operation.duration` → `db.repo.operation.duration` | Re-key SQL repository dashboards, alerts, and queries to `db.repo`. |
| Renamed | DynamoDB repository operations: `db.client.operation.duration` → `ddb.operation.duration` | Re-key DynamoDB repository dashboards, alerts, and queries to `ddb`. |
| Deleted and consolidated | `db.repo.model_event.notify.errors` | Use `db.repo.model_event.notifications` for both outcomes; the former error-only metric is not emitted. |
| Added outcome coverage | A cancelled HTTP client request | `http.client.request.duration` is emitted with `error.type=metric.ErrorType(context.Canceled)` before the original cancellation error is returned. |
| Deleted aggregate series | HTTP server-only, Kafka topic-only, and Kinesis stream-only aggregate data | Query or aggregate the retained detailed series in the backend rather than searching for a framework-emitted `KindTotal` datum. |
| Dimension value changed | An unknown model on a stream consumer | `stream.errors` no longer carries `error.type="unknown_model"`, and the failure is no longer counted twice on the single-message consumer. Both consumers now count it once, through the shared error path, with the normalized Go error type. |
| Help text added | Every metric in the inventory above | Each metric is exported with a description of what it counts instead of `unit: <unit>`. Emitting packages register it through `metric.RegisterHelp`; `Kind.WithHelp` still overrides it per datum. |
| Renamed | **Your own** metrics written through `metric.NewWriter("")`, on CloudWatch | A namespace-less leaf is now PascalCased exactly like a namespaced one, so `my.custom.metric` exports as `MyCustomMetric` and `my-metric-name` as `My-metric-name`. This hits application-authored metrics, not only gosoline's: re-key every CloudWatch dashboard and alarm built on one. See "Metrics authored outside gosoline" below for the Prometheus and OTEL effect. |

#### Failures and outcomes are attributes, not metrics

Eleven metrics were folded into the metric recording the operation, so a failure or an alternative
outcome is a series on that metric rather than a metric of its own. Attempts are the sum over the
attribute; failures are the non-`{{default}}` `error.type`.

| Removed | Now query | Attribute that tells them apart |
|---|---|---|
| `stream.errors` | `stream.consumed.messages` | `error.type` |
| `kafka.consumer.commit.errors` | `kafka.consumer.commit.duration` | `error.type` |
| `kafka.producer.send.errors` | `kafka.producer.sent.messages` | `error.type` |
| `cloud.aws.kinesis.send.errors` | `cloud.aws.kinesis.sent.messages` | `error.type`, carrying the reason Kinesis reported |
| `cloud.aws.kinesis.consume.errors` | `cloud.aws.kinesis.consumed.messages` | `error.type` |
| `kvstore.hits` | `kvstore.reads` | `hit` (`true` / `false`) |
| `limit.releases`, `limit.throttles`, `limit.errors` | `limit.takes` | `outcome` (`allowed` / `throttled` / `error`) plus `error.type` |
| `mdlsub.consumed.events`, `mdlsub.skipped.events`, `mdlsub.consume.errors` | `mdlsub.events` | `outcome` (`applied` / `skipped`) plus `error.type` |

`limit.takes` is now recorded when a take **ends** rather than when it starts, because its outcome is
not known at the start. `kafka.consumer.consume.errors` was deliberately not folded - see
`pkg/metric/SEMCONV.md`.

The full specification, including every metric's attributes and the semantic-convention metric it
derives from, is `pkg/metric/SEMCONV.md`.

#### Prometheus writer: exported names no longer carry the application

The Prometheus writer's namespace is now the fixed string `gosoline` rather than a namespace formatted
from the application identity. `gosoline` names the framework that authored the metric; the application
belongs in the labels the scrape target is discovered with, so carrying it in the metric name kept one
query from spanning several applications.

| | Before | After |
|---|---|---|
| Exported name | `<app.namespace>_<app.name>_http_server_request_duration_seconds` | `gosoline_http_server_request_duration_seconds` |

**Every Prometheus query, recording rule, dashboard and alert has to be re-keyed**, and the
`metric.writer_settings.prometheus.naming` configuration block - `namespace_pattern` and
`namespace_delimiter` - is removed. Leaving it in a configuration file is harmless; it has no effect.
`metric.NewPrometheusWriterWithInterfaces` lost its `namespace` parameter for the same reason.

#### Metrics authored outside gosoline

A datum written through a writer with no namespace is now rendered by the same rules as a namespaced
one, minus the namespace, because its leaf can carry canonical separators just the same:

| Writer | Leaf `my.custom.metric`, before | After |
|---|---|---|
| CloudWatch | `my.custom.metric` | `MyCustomMetric` |
| Prometheus | `my.custom.metric`, rejected at registration | `my_custom_metric_total` on a counter |
| OTEL | `my.custom.metric` | `my.custom.metric`, unchanged |

The Prometheus rendering fixes an outright defect - a dotted name is not a valid Prometheus metric
name, so those metrics were never exported. **The CloudWatch rendering renames custom metrics**, since
a leaf is now PascalCased whether or not a namespace precedes it; re-key any CloudWatch dashboard built
on a `metric.NewWriter("")` metric.

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

#### Attribute-key policy

**No gosoline metric carries a canonical OpenTelemetry semantic-convention name, and no attribute key
carries a namespace prefix.** The conventions are followed for grammar, units and attribute shape; the
names stay gosoline's own, and the OTEL renderer prefixes every one with `gosoline.`. `error.type` is
the single key taken verbatim from a convention, because its values are what make an operation metric
self-describing.

An attribute key never repeats the namespace of the metric it is attached to - the metric already names
its subsystem.

| Former key | Key now |
|---|---|
| `Consumer` (stream) | `consumer.name` |
| `ProducerDaemon` | `producer.name` |
| `Scheduler` | `name` |
| `ModelId` | `model.id` |
| `model`, `store` (kvstore) | `model.id`, `store.type` |
| `Operation` (blob) | `operation` |
| `Operation` (db, ddb) | `operation.name` |
| `Type` (db connections) | `connection.state`, values `used` and `idle` |
| `Topic` | `topic.name` |
| `StreamName` | `stream.name` |
| redis list destination | `list.name` |
| `ShardId`, `Partition` | `partition.id` |
| `ClientType`, `Client`, `Broker` | `client.type`, `client.name`, `broker.address` |
| `Method`, `Path`, `ServerName` | `request.method`, `route`, `name` |
| `full_method` | `service` and `method` |
| `name`, `prefix` (limit) | `name`, `prefix` |
| `sampled` | `sampled` |
| `trace_id` (limit) | **removed** - a trace id is unbounded cardinality and belongs on a span |

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
10. Remove `metric.calculator` and `stream.metrics.messages_per_runner` configuration, and replace ECS scaling policies that reference `PerRunner*`, `StreamMessages`, `HttpServerRequests` or `ShardTaskRatio`; none of those per-runner metrics are emitted.
