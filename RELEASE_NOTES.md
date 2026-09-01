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

#### Metric name mapping

| Former name | Namespace | Leaf |
|---|---|---|
| `Duration` (stream consumer), `ProcessDuration` | `messaging` | `process.duration` |
| `ProcessedCount`, `RecordsConsumed`, `ReadRecords` | `messaging` | `client.consumed.messages` |
| `RecordsSent`, `PutRecords` | `messaging` | `client.sent.messages` |
| `ProduceDuration` | `messaging` | `client.operation.duration` |
| `Error`, `UnknownModelError` | `stream.consumer` | `errors` |
| `RetryGetCount`, `RetryPutCount` | `stream.consumer` | `retry.operations` |
| `MessageCount` | `stream.producer` | `messages` |
| `BatchSize` | `stream.producer` | `batch.size` |
| `AggregateSize` | `stream.producer` | `aggregate.size` |
| `IdleDuration` | `stream.producer` | `idle.duration` |
| `StreamRedisListInputLength` | `stream.input.redis_list` | `message.count` |
| `StreamRedisListInputReads` | `stream.input.redis_list` | `reads` |
| `StreamRedisListOutputWrites` | `stream.output.redis_list` | `writes` |
| `StreamMessagesAvailable` | `stream.input` | `available.messages` |
| `StreamMessagesSent` | `stream.output` | `sent.messages` |
| `StreamMessages` | `autoscaling.per_runner` | `stream.messages` |
| `HttpServerRequests` | `autoscaling.per_runner` | `http.server.requests` |
| `PerRunner<Name>` | `autoscaling.per_runner` | `<name>` |
| `PollCount` | `kafka.consumer` | `polls` |
| `PollDuration` | `kafka.consumer` | `poll.duration` |
| `CommitDuration` | `kafka.consumer` | `commit.duration` |
| `CommitFailures` | `kafka.consumer` | `commit.errors` |
| `WaitDuration` (kafka) | `kafka.consumer` | `wait.duration` |
| `RebalanceCount` | `kafka.consumer` | `rebalances` |
| `RecordsConsumedFailed` | `kafka.consumer` | `consume.errors` |
| `ProduceBatchSize` | `kafka.producer` | `batch.size` |
| `RecordsSentFailed` | `kafka.producer` | `send.errors` |
| `BrokerConnects`, `BrokerConnectsFailed` | `kafka.broker` | `connects` |
| `BrokerThrottleCount` | `kafka.broker` | `throttles` |
| `BrokerThrottleTime` | `kafka.broker` | `throttle.duration` |
| `ProduceBatchRecords` | `kafka.broker` | `produce.batch.records` |
| `ProduceBatchBytes` | `kafka.broker` | `produce.batch.size` (bytes) |
| `ProduceBatchBytesCompressed` | `kafka.broker` | `produce.batch.compressed.size` (bytes) |
| `FetchBatchRecords` | `kafka.broker` | `fetch.batch.records` |
| `FetchBatchBytes` | `kafka.broker` | `fetch.batch.size` (bytes) |
| `FetchBatchBytesCompressed` | `kafka.broker` | `fetch.batch.compressed.size` (bytes) |
| `ReadCount` | `aws.kinesis.shard` | `reads` |
| `FailedRecords` | `aws.kinesis.shard` | `consume.errors` |
| `MillisecondsBehind` | `aws.kinesis.shard` | `lag` |
| `AcquireShardDelaySeconds` | `aws.kinesis.shard` | `acquire.delay` |
| `SleepDuration` | `aws.kinesis.shard` | `sleep.duration` |
| `WaitDuration` (kinesis) | `aws.kinesis.shard` | `wait.duration` |
| `PutRecordsBatchSize` | `aws.kinesis.producer` | `batch.size` |
| `PutRecordsFailure` | `aws.kinesis.producer` | `send.errors` |
| `kvStoreRead` | `kvstore` | `reads` |
| `kvStoreWrite` | `kvstore` | `writes` |
| `kvStoreDelete` | `kvstore` | `deletes` |
| `kvStoreHit` | `kvstore` | `hits` |
| `kvStoreSize` | `kvstore` | `item.count` |
| `DbConnectionCount` (`inUse`, `idle`) | `db.client` | `connection.count` |
| `DbConnectionCount` (`new`) | `db.client` | `connections` |
| `DbAccessSuccess`, `DbAccessFailure`, `DbAccessLatency` | `db.client` | `operation.duration` |
| `DdbAccessSuccess`, `DdbAccessFailure`, `DdbAccessLatency` | `db.client` | `operation.duration` |
| `ModelEventNotifySuccess` | `db.repo` | `model_event.notifications` |
| `ModelEventNotifyFailure` | `db.repo` | `model_event.notify.errors` |
| `ModelEventConsumeSuccess` | `mdlsub` | `consumed.events` |
| `ModelEventConsumeSkipped` | `mdlsub` | `skipped.events` |
| `ModelEventConsumeFailure` | `mdlsub` | `consume.errors` |
| `HttpRequestCount`, `HttpRequestCountPerRoute`, `HttpRequestResponseTime`, `HttpStatus2XX`–`HttpStatus5XX` | `http.server` | `request.duration` |
| `HttpRequestsRejected` | `http.server` | `rejected.requests` |
| `HttpConcurrentRequests` | `http.server` | `active_requests` |
| `HttpOpenConnections` | `http.server` | `connection.count` |
| `HttpClientRequest`, `HttpRequestDuration`, `HttpClientResponseCode*`, `HttpClientError` | `http.client` | `request.duration` |
| `ApiRequestCount`, `ApiRequestResponseTime` | `rpc.server` | `duration` |
| `BlobBatchRunner` | `blob` | `batch.operations` |
| `schedulerBatchSize` | `conc.scheduler` | `batch.size` |
| `schedulerTaskDelay` | `conc.scheduler` | `task.delay` |
| `rate_limit_take` | `limit` | `rate_limit.takes` |
| `rate_limit_release` | `limit` | `rate_limit.releases` |
| `rate_limit_throttle` | `limit` | `rate_limit.throttles` |
| `rate_limit_error` | `limit` | `rate_limit.errors` |
| `sampling_decision` | `smpl` | `decisions` |
| `debug`, `info`, `warn`, `error` | `log` | `records` |

#### Metrics that no longer exist

| Former name | Obtain it from |
|---|---|
| `HttpRequestCount`, `HttpRequestCountPerRoute`, `HttpClientRequest`, `ApiRequestCount` | the observation count of the corresponding duration histogram |
| `HttpStatus2XX`–`HttpStatus5XX`, `HttpClientResponseCode*` | the `http.response.status_code` dimension of the duration histogram |
| `HttpClientError` | the `error.type` dimension of `http.client` `request.duration` |
| `DbAccessSuccess`/`Failure`, `DdbAccessSuccess`/`Failure`, `BrokerConnectsFailed` | the `error.type` dimension of the corresponding metric |
| `ShardTaskRatio`, `ShardTaskRatioMax` | `aws.kinesis.consumer` `shard.count` divided by `client.count`, multiplied by 100 for the former scale, maximum taken across the stream dimension |
| `DbConnectionCount` with `Type=open` | the sum of `db.client` `connection.count` across `db.client.connection.state` |

#### Dimension key mapping

Every key an OpenTelemetry semantic convention defines is now spelled the way the convention spells
it; every other key is canonical.

| Former key | New key |
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
7. Re-key every alarm, dashboard and query off the metric name mapping above, selecting by the `metric.schema_version` of `v2.0`.
8. Update ECS scaling policies that reference `PerRunner*`, `StreamMessages`, `HttpServerRequests` or `ShardTaskRatio`.
