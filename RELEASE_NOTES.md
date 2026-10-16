# Release notes: metric v2 emission contract

**Target:** next release

**Metric schema:** `v2.0`

This release establishes Gosoline's metric v2 emission contract: canonical namespace and leaf names, instrument kinds, units, and dimensions. Read `metric.schema_version` from the metadata document to identify the contract. There is no overlap period or dual emission; dashboards, alerts, and queries that use the former names or dimensions must be migrated.

## Canonical names and writer output

Metrics are authored once using a canonical namespace and leaf; each writer renders them for its backend. For the HTTP server request-duration histogram:

| Writer | Exported metric name | Duration unit |
|---|---|---|
| CloudWatch | `HttpServerRequestDuration` | milliseconds for this metric, as authored |
| Prometheus | `gosoline_http_server_request_duration_seconds` | seconds |
| OTEL | `gosoline.http.server.request.duration` | `s` |

The Prometheus writer always uses the fixed `gosoline` prefix, not an application-derived prefix. Application identity belongs in scrape-target labels. For framework metrics with canonical namespaces, the OTEL writer prefixes names with `gosoline.`; those names are Gosoline's own, not OpenTelemetry semantic-convention names. Namespace-less custom OTEL metrics are rendered from the leaf without the `gosoline.` prefix.

The `http.server.request.duration` histogram uses canonical dimensions `name`, `route`, `request.method`, and `response.status_code`. The `http.client.request.duration` histogram uses `request.method` with `response.status_code` for responses or `error.type` for failures. Status is the full numeric code, not a `1xx`–`5xx` class. `http.server.rejected.requests` carries only `name`, `route`, and `request.method`; it has no `response.status_code`. In particular, route is `route`, not `http.route`. Prometheus converts dotted dimension keys to valid label names, for example `request.method` to `request_method` and `response.status_code` to `response_status_code`.

The Prometheus writer resolves registered defaults against canonical names and dimensions before translating dotted label
keys and `{{default}}` values, preserving the declared priority, unit, and instrument kind for default-backed samples.

Most duration observations are authored in milliseconds. `cloud.aws.kinesis.acquire.duration` is authored in seconds. Prometheus and OTEL render durations in seconds; CloudWatch retains the actual authored unit, so its duration values are not universally milliseconds.

Explicit histogram bucket bounds use the metric's authored unit. Prometheus and OTEL convert these bounds along with
observations to their base unit; for example, 1, 5, and 10 milliseconds become 0.001, 0.005, and 0.01 seconds.

Framework-authored duration metrics that were Prometheus summaries now use histograms; custom metrics using `KindSummary` remain Prometheus summaries. Use histogram buckets (for example, `histogram_quantile`) for the framework duration metrics. `UnitMillisecondsAverage` and its eight related units resolve to their base unit before rendering, so millisecond histograms do not export UCUM unit `1`.

The four Kafka batch-size metrics—`kafka.produce.batch.size`, `kafka.produce.batch.compressed.size`, `kafka.fetch.batch.size`, and `kafka.fetch.batch.compressed.size`—use byte units rather than count units. Kafka consumer metrics already had topic-level dimensions before this release; their v2 names, instrument types, and attributes change. Separately, `kafka.MetricPair` now emits per-topic/partition series only; its former topic-only `KindTotal` point was CloudWatch-visible and is removed. Prometheus and OTEL already filter `KindTotal`. Aggregate retained partition series in CloudWatch for topic-level views.

Use `metric.RegisterHelp(namespace, metricName, help)` to register package-level descriptions; `Kind.WithHelp(...)` overrides a description for one datum. Prometheus and OTEL resolve help from the datum first, then registered help, then a unit-description fallback for unregistered custom metrics. CloudWatch does not export help text.

OTEL count units derive a singular item annotation from the final metric-name component using the existing inflection dependency: custom leaves such as `queries` and `statuses` export `{query}` and `{status}`. Singular and uncountable leaves retain UCUM unit `1`; physical units are unaffected.

OTEL instrument units and descriptions are resolved only when a counter, gauge, or histogram is first registered, using the writer's existing instrument caches. Repeated samples do not repeat count-unit inflection or description lookup.

## Writer API and custom metrics

The metric writer API now takes the canonical namespace explicitly:

```go
func NewWriter(namespace string, defaults ...*Datum) Writer
func NewWriterWithInterfaces(clock clock.Clock, channel *metricChannel, namespace string) Writer
```

`metric.Datum` also has a `Namespace` field. An empty namespace omits only the canonical namespace from the datum; it does not preserve backend names. Writers still apply their own rendering rules:

| Writer | A custom counter leaf `my.custom.metric` with an empty namespace |
|---|---|
| CloudWatch | `MyCustomMetric` |
| Prometheus | `gosoline_my_custom_metric_total` |
| OTEL | `my.custom.metric` |

The Prometheus interface constructor no longer accepts a namespace. Its current signature is:

```go
func NewPrometheusWriterWithInterfaces(
    logger log.Logger,
    registry *prometheus.Registry,
    metricLimit int64,
    writeGraceTime time.Duration,
) Writer
```

It uses the fixed `gosoline` prefix. Remove the former `metric.writer_settings.prometheus.naming.namespace_pattern` and `namespace_delimiter` settings and update direct constructor calls to the current signature.

The Kafka helper API also changed: `kafka.MetricPair` now takes one `kafka.MetricSpec` instead of positional arguments; its current package signature is `func MetricPair(spec MetricSpec) metric.Data`. `MetricSpec` carries client type/name, namespace, metric name, topic, partition, error type, value, unit, and kind. The former `kafka.DimensionConsumer` and `kafka.DimensionProducer` constants are removed; use `kafka.DimensionClientType` with `kafka.ClientTypeConsumer` or `kafka.ClientTypeProducer`.

## HTTP metric migrations

HTTP client and server request counters, status-class counters, and duration series are consolidated into request-duration histograms. The previous entries below are source-level metric-family identifiers, not literal v1 exporter names; the old exported names depended on the writer and, for some writers, application identity.

| Previous source-level metric family | V2 metric and migration |
|---|---|
| HTTP client `HttpClientRequest`, `HttpClientError`, `HttpClientResponseCode1XX` through `HttpClientResponseCode5XX`, and `HttpRequestDuration` | `http.client.request.duration` is a histogram with `request.method`, full `response.status_code`, and `error.type` dimensions as applicable. In Prometheus, use the histogram's `_count` series in place of the request counter; use the full status-code dimension instead of the former status-class families. |
| HTTP server `HttpRequestCount`, `HttpRequestCountPerRoute`, `HttpStatus1XX` through `HttpStatus5XX`, and `HttpRequestResponseTime` | `http.server.request.duration` is a route/method/full-status histogram. Its Prometheus `_count` replaces the request counters. CloudWatch no longer emits the former server-only `KindTotal` request/duration aggregate; Prometheus and OTEL already filtered `KindTotal`. Aggregate route series in CloudWatch for server-wide views. |
| HTTP server `HttpRequestsRejected` | `http.server.rejected.requests` carries `name`, `route`, and `request.method` only (no `response.status_code`); CloudWatch no longer emits a server-wide rejection aggregate. |

The exported `pkg/httpserver` constants `MetricHttpRequestCount`, `MetricHttpRequestCountPerRoute`, `MetricHttpRequestResponseTime`, and `MetricHttpStatus` are removed. The current `MetricHttpRequestsRejected` constant names the leaf `rejected.requests`.

A cancelled HTTP client request is recorded in `http.client.request.duration` with `error.type="request_canceled"` before the original cancellation error is returned. Other request failures use `error.type="request_failed"`. Valid HTTP responses, including error status codes such as 500, carry the numeric `response.status_code` dimension.

## Canonical name migrations

The former names below are canonical/logical migration keys, not exact v1 exporter names; the old writer and application namespace determined rendered series. Re-key consumer queries and dashboards accordingly.

| Former canonical/logical family | V2 canonical metric |
|---|---|
| `metric.records` | `metric.log.records` |
| SQL repository `db.client.operation.duration` | `db.repo.operation.duration` |
| DynamoDB repository `db.client.operation.duration` | `ddb.operation.duration` |

## Outcome metrics and writer-specific queries

Standalone failure or alternative-outcome families now use the metric for the operation, distinguished by dimensions. The left column is a logical family name or source identifier, not an exact old exported name; v1 names varied by writer and application namespace.

| Earlier family or source identifier | V2 metric | Outcome dimension |
|---|---|---|
| `stream.errors` | `stream.consumed.messages` | `error.type` |
| `kafka.producer.send.errors` | `kafka.producer.sent.messages` | `error.type` |
| `cloud.aws.kinesis.send.errors` | `cloud.aws.kinesis.sent.messages` | `error.type`, using the reason Kinesis reported |
| `cloud.aws.kinesis.consume.errors` | `cloud.aws.kinesis.consumed.messages` | `error.type`; see the overlapping total below |
| `kvstore.hits` | `kvstore.reads` | `hit` (`true` or `false`) |
| `limit.releases`, `limit.throttles`, `limit.errors` | `limit.takes` | `outcome` (`allowed`, `throttled`, or `error`) and, for failures, `error.type` |
| `mdlsub.consumed.events`, `mdlsub.skipped.events`, `mdlsub.consume.errors` | `mdlsub.events` | `outcome` (`applied` or `skipped`) and, for failures, `error.type` |
| `kafka.consumer.consume.errors` | `kafka.consumer.consumed.messages` | `error.type` |
| `ModelEventNotifySuccess` and `ModelEventNotifyFailure` | `db.repo.model_event.notifications` | The success path uses the internal default marker; failures use `error.type="notify_failed"` |

The notification success and failure families are both consolidated into `db.repo.model_event.notifications`, a `UnitCount` counter. There is no separate success label and no error-only replacement metric.

`kafka.consumer.consumed.messages` is also a `UnitCount` counter. Its registered zero-value default and all outcome samples carry the unit and counter kind, including `error.type="processing_failed"` samples. This prevents missing-unit validation errors in idle and active consumers. OTEL exports `gosoline.kafka.consumer.consumed.messages` as a monotonic counter with unit `{message}`; Prometheus exports `gosoline_kafka_consumer_consumed_messages_total` as a counter.

`kafka.producer.sent.messages` uses a `UnitCount` counter for both successful records on the default series and failed
records with `error.type="send_failed"`.

`metric.DimensionDefault` (`"{{default}}"`) is an internal marker, not a value exported by a writer. Its backend representation is:

| Writer | Default representation | Example success/total selection |
|---|---|---|
| Prometheus | Empty label value; `error.type` renders as `error_type` | `error_type=""` |
| OTEL | The `error.type` attribute is absent | Select points with no `error.type` attribute |
| CloudWatch | The default dimension is omitted | Select the series without that dimension |

For notification failures, select `error_type="notify_failed"` in Prometheus, the `error.type="notify_failed"` attribute in OTEL, or the CloudWatch series carrying that error dimension. For other outcome metrics, use the same writer-specific representation for their dimensions.

Where outcome categories are exclusive, sum the outcome series to obtain attempts. `stream.consumed.messages` and `cloud.aws.kinesis.consumed.messages` are exceptions: their default series is the all-processed total, while `error.type="processing_failed"` is an overlapping failed-record subset. Do not add or subtract that subset to infer successful records. Error values name the failed action, not a Go error type.

## Stream and Kinesis semantics

Single-message and batch stream consumers share the v2 metrics `stream.process.duration`, `stream.consumed.messages`, and `stream.retry.operations`:

- `stream.process.duration` records one observation per processing invocation, including a batch invocation; it is not one duration observation per message.
- `stream.consumed.messages` records the all-processed per-invocation message total with the internal default marker: a single-message invocation counts one, and a batch invocation counts its messages. Failed, unacknowledged records emit `error.type="processing_failed"` as an overlapping subset; a non-ignored unknown-model failure is counted once in that subset, with no standalone unknown-model or error metric.
- `stream.retry.operations` records retry queue operations by attempt, using `operation="get"` and `operation="put"`.

The Kinesis record writer reports each `PutRecords` attempt. `cloud.aws.kinesis.sent.messages` counts successful records on the default series and failed records by the error reason Kinesis returned. `cloud.aws.kinesis.batch.records` is a histogram of records in each attempt; a retried record is counted again in the subsequent attempt, so this is attempt-level work rather than a unique-record total. The failed-record subset and retry behavior remain part of the producer contract. The shard reader now emits its metrics—including `cloud.aws.kinesis.reads`, `cloud.aws.kinesis.consumed.messages`, and `cloud.aws.kinesis.lag`—with stream-and-shard dimensions only. The former stream-only `KindTotal` points for shard-reader metrics were CloudWatch-visible and are removed; Prometheus and OTEL already filter `KindTotal`. Aggregate retained shard series in CloudWatch for stream-level views.

Kinsumer no longer emits `ShardTaskRatio` per stream or the application-wide `ShardTaskRatioMax`. The former ratio was `shardCount / clientCount * 100`; the v2 `cloud.aws.kinesis.shard.count` and `cloud.aws.kinesis.client.count` gauges are raw per-stream counts, each dimensioned by `stream.name`. Derive `shard.count / client.count` per stream (ideal ratio `1`, not `100`), then take the maximum across streams when global scaling needs one value; update thresholds rather than carrying forward the old ×100 scale. These are stream gauges, not calculator output. `cloud.aws.kinesis.lag` remains a per-stream/per-shard cumulative histogram in OTEL, with millisecond observations; CloudWatch continues to report the maximum of current samples.

## Calculator and per-runner removals

The `pkg/metric/calculator` package and `application.WithMetricsCalculatorModule(app)` are removed. Remove these old configuration keys:

- `metric.calculator`
- `metric.calculator.handlers.stream_messages_per_runner`
- `metric.calculator.handlers.httpserver_requests_per_runner`

The source-level metrics `PerRunnerStreamMessages`, `PerRunnerHttpServerRequests`, `StreamMessagesAvailable`, and `StreamMessagesSent` are no longer emitted. Replace autoscaling rules that depended on them with application-specific signals. Do not substitute a similarly named configuration key: the removed handlers read the exact keys above.

## Canonical dimension changes

Update queries, dashboards, and alerts to the v2 dimension keys and values:

| Former key | V2 key or value |
|---|---|
| `Consumer` (stream) | `consumer.name` |
| `ProducerDaemon` | `producer.name` |
| `Scheduler` | `name` |
| `ModelId` | `model.id` |
| `model`, `store` (kvstore) | `model.id`, `store.type` |
| `Operation` (blob) | `operation` |
| `Operation` (db, ddb) | `operation.name` |
| `Type` (db connections) | `connection.state` (`used` or `idle`) |
| `Topic` | `topic.name` |
| `StreamName` | `stream.name` |
| Redis list destination | `list.name` |
| `ShardId`, `Partition` | `partition.id` |
| `ClientType`, `Client`, `Broker` | `client.type`, `client.name`, `broker.address` |
| `Method`, `Path`, `ServerName` | `request.method`, `route`, `name` |
| `full_method` | `service` and `method` |
| `name`, `prefix` (limit) | `name`, `prefix` |
| `sampled` | `sampled` |
| `trace_id` (limit) | Removed; trace IDs are unbounded-cardinality and belong on spans |

The names remain Gosoline's own rather than semantic-convention metric names. `error.type` is used for bounded outcome labels, not arbitrary error strings.

## Source-reviewed metric inventory (`v2.0`)

Source review lists 62 framework-authored canonical metric leaves across 18 namespaces for schema `v2.0`.

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
| `kafka.consumer` | `consumed.messages`; `process.duration`; `polls`; `poll.duration`; `commit.duration`; `sleep.duration`; `rebalances` |
| `kafka.producer` | `sent.messages`; `produce.duration`; `batch.records` |
| `kvstore` | `reads`; `writes`; `deletes`; `item.count` |
| `limit` | `takes` |
| `mdlsub` | `events` |
| `metric` | `log.records` |
| `rpc.server` | `request.duration` |
| `smpl` | `decisions` |
| `stream` | `consumed.messages`; `process.duration`; `retry.operations`; `produced.messages`; `batch.messages`; `aggregate.messages`; `idle.duration`; `message.count`; `reads`; `writes` |

The full metric specification, including units, kinds, descriptions, and dimensions, is in `pkg/metric/SEMCONV.md`.

## Existing examples

The existing `examples/otel` application now passes a namespace to `metric.NewWriter`. The v2 writer exports its millisecond-authored duration metric in seconds.

The Prometheus example now supplies a valid count unit for its custom summary. Its tutorial links to the public metric specification and shows the fixed `gosoline` prefix and observed custom-series output.

## Upgrade checklist

1. Read `metric.schema_version`; migrate consumers to schema `v2.0`. Do not expect v1 aliases or dual emission.
2. Re-key metric queries, dashboards, alerts, and recording rules against the canonical inventory; update dimensions and units as well as names.
3. Replace HTTP request/status-class counters with histogram counts and the full numeric `response.status_code`; use route-level server rejections. For CloudWatch server-wide, topic-level, and stream-level views, aggregate the retained HTTP route, Kafka partition, and Kinesis shard series; the old server/topic/stream-only `KindTotal` series are not emitted by CloudWatch (Prometheus and OTEL already filter `KindTotal`).
4. Replace standalone outcome metrics with operation metrics and writer-appropriate `error.type`/default selectors. Preserve the overlapping stream and Kinesis consumer failure-subset semantics.
5. Update `metric.NewWriter` and `NewWriterWithInterfaces` calls, Prometheus writer construction, and removed Prometheus naming settings. Migrate Kafka callers to `kafka.MetricPair(kafka.MetricSpec{...})` and update removed or changed Kafka/HTTP metric constants. Re-key custom metrics for each backend's rendering.
6. Remove the calculator option and exact configuration keys above; replace per-runner scaling inputs with application-specific signals and derive Kinesis capacity ratios from the shard/client gauges.
7. Update duration dashboards for seconds in Prometheus and OTEL, preserve CloudWatch's authored units, and use histogram queries for framework duration metrics while retaining custom `KindSummary` queries.
