# Release notes: metric v2 emission contract

**Target:** next release

**Metric schema:** `v2.0`

This release establishes Gosoline's metric v2 emission contract: canonical namespace and leaf names, instrument kinds, units, and dimensions. Read `metric.schema_version` from the metadata document to identify the contract. There is no overlap period or dual emission; dashboards, alerts, and queries that use the former names or dimensions must be migrated. The version remains `v2.0` while the still-unmerged MR introducing v2 is in progress; its restored counters and corrected semantics do not trigger another version increment, a capability flag, or compatibility aliases.

## Canonical names and writer output

Metrics are authored once using a canonical namespace and leaf; each writer renders them for its backend. For the HTTP server request-duration histogram:

| Writer | Exported metric name | Duration unit |
|---|---|---|
| CloudWatch | `HttpServerRequestDuration` | milliseconds for this metric, as authored |
| Prometheus | `gosoline_http_server_request_duration_seconds` | seconds |
| OTEL | `gosoline.http.server.request.duration` | `s` |

The Prometheus writer always uses the fixed `gosoline` prefix, not an application-derived prefix. Application identity belongs in scrape-target labels. For framework metrics with canonical namespaces, the OTEL writer prefixes names with `gosoline.`; those names are Gosoline's own, not OpenTelemetry semantic-convention names. Namespace-less custom OTEL metrics are rendered from the leaf without the `gosoline.` prefix.

The `http.server.request.duration` histogram uses `name`, `route`, `request.method`, and full numeric `response.status_code`. The `http.client.request.duration` histogram uses `request.method` with `response.status_code` for responses or `error.type` for failures. The restored HTTP response/completion counters also carry bounded `response.status_class` values `1xx` through `5xx` alongside full status codes; the histograms do not carry this class. `http.server.rejected.requests` retains its detailed `name`, `route`, and `request.method` Counter and adds a CloudWatch-only same-name `KindTotal` projection with exactly `name`. In particular, route is `route`, not `http.route`. Prometheus converts dotted dimension keys to labels, for example `request.method` to `request_method` and `response.status_class` to `response_status_class`.

Writers export only samples explicitly written by the application; no metric registry resolves datum metadata or synthesizes startup/flush observations. Each datum must carry its own priority, unit, and instrument kind. Framework histograms retain their existing explicit bucket profile where applicable; otherwise they use the standard implicit exporter bounds. Namespaces are inherited from the writer when a datum's namespace is empty.

SQL and DynamoDB duration observations retain their authored high priority for both success and
failure. Aggregate CloudWatch output no longer inherits a low priority from a synthetic success
sample, so successful `db.repo.operation.duration` and `ddb.operation.duration` series can now appear
where they were previously filtered. Account for these series when reviewing CloudWatch costs.

Most duration observations are authored in milliseconds. `cloud.aws.kinesis.acquire.duration` is authored in seconds. Prometheus and OTEL render durations in seconds; CloudWatch retains the actual authored unit, so its duration values are not universally milliseconds.

Explicit histogram bucket bounds use the metric's authored unit. Prometheus and OTEL convert these bounds along with
observations to their base unit; for example, 1, 5, and 10 milliseconds become 0.001, 0.005, and 0.01 seconds.


### Histogram bucket policies

`HistogramKindBuilder` now provides `WithHttpDurationBuckets()`, `WithRpcDurationBuckets()`,
`WithMessagingDurationBuckets()`, `WithMessagingDurationBucketsInSeconds()`, and
`WithDatabaseDurationBuckets()`. The HTTP request
(`http.server.request.duration`, `http.client.request.duration`), RPC
(`rpc.server.request.duration`), and messaging (`stream.process.duration`,
`kafka.consumer.process.duration`, `cloud.aws.kinesis.process.duration`,
`kafka.producer.produce.duration`, `kafka.consumer.poll.duration`, and
`kafka.consumer.commit.duration`) families use the recommended bounds `[5, 10, 25, 50, 75, 100,
250, 500, 750, 1000, 2500, 5000, 7500, 10000]` ms, exported by Prometheus and OpenTelemetry as
`[0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1, 2.5, 5, 7.5, 10]` seconds. The
database operation histograms (`db.repo.operation.duration`, `ddb.operation.duration`) use
`[1, 5, 10, 50, 100, 500, 1000, 5000, 10000]` ms, exported as
`[0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5, 10]` seconds. Poll and commit reuse the messaging
client-operation recommendation for receive/settle operations without changing their client poll
and offset-commit measurement boundaries; processing duration remains separate. Metric names,
attributes, instrument kinds, and counting behavior are unchanged. The schema remains `v2.0`; no
bucket-policy flag or migration mode is added.

Kinesis `cloud.aws.kinesis.lag`, `acquire.duration`, `sleep.duration`, and `wait.duration` now reuse
the messaging duration profile, alongside the existing `process.duration` histogram. All five export
the same 14 second-valued boundaries listed above. `acquire.duration` remains authored in seconds
and receives equivalent second-valued bounds; the other four remain authored in milliseconds.
CloudWatch observation units, values and reducers, detailed attributes and stream-only projections
are unchanged.

The other 15 histograms have no explicit semantic-convention profile. For unspecified bounds,
OpenTelemetry and Prometheus both use the OpenTelemetry standard boundaries `[0, 5, 10, 25, 50,
75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000]` in exported base units. This changes
Prometheus's previous generic `DefBuckets` layout for implicit histograms; explicit profiles, including
the messaging profile above, remain unchanged. Metric names, dimensions, units, recorded values, and
routing remain unchanged; only implicit Prometheus bucket boundaries are aligned. These defaults
are not evidence of fit to observed
workloads: for custom durations, neither writer has a positive boundary below 5 seconds; the finite
buckets end at 10000 in exported base units. No claim is made that all histograms now have
workload-ideal precision. See `pkg/metric/SEMCONV.md` for the complete unchanged-family list, default
bounds, and primary recommendations.

Histogram changes do not rebucket stored data, so historical coarser distributions cannot recover
finer percentile detail. HTTP percentile aggregation rejects mixed explicit-bound layouts rather
than combining buckets positionally: snapshots remain `Unknown` while their five-complete-minute
window contains mixed layouts, and history has gaps only for mixed output bucket/grain pairs. HTTP
history means share that conservative mixed-layout guard; route and general weighted means are
bounds-independent.

Framework-authored duration metrics that were Prometheus summaries now use histograms; custom metrics using `KindSummary` remain Prometheus summaries. Use histogram buckets (for example, `histogram_quantile`) for framework duration metrics. The histogram `_count` and `_sum` series describe histogram observations and are not substitutes for restored standalone Counter instruments or their event/volume semantics. `UnitMillisecondsAverage` and its eight related units resolve to their base unit before rendering, so millisecond histograms do not export UCUM unit `1`.

Kafka's four existing batch-size distribution histograms—`kafka.produce.batch.size`, `kafka.produce.batch.compressed.size`, `kafka.fetch.batch.size`, and `kafka.fetch.batch.compressed.size`—use byte units. Six independent high-priority hook Counters now add produced/fetched record counts and uncompressed/compressed byte volumes in the corresponding franz-go batch callbacks; see “Restored counter migrations” below for exact names, values, dimensions, and boundaries. Keep the existing `produce.batch.*` and `fetch.batch.*` histograms for per-batch distributions, not as substitutes for the counters. Kafka consumer metrics already had topic-level dimensions before this release; their v2 names, instrument types, and attributes change. Separately, `kafka.MetricPair` now emits per-topic/partition series only; its former topic-only `KindTotal` point was CloudWatch-visible and is removed. Prometheus and OTEL already filter `KindTotal`. Aggregate retained partition series in CloudWatch for topic-level views.

Use `metric.RegisterHelp(namespace, metricName, help)` to register package-level descriptions; `Kind.WithHelp(...)` overrides a description for one datum. Prometheus and OTEL resolve help from the datum first, then registered help, then a unit-description fallback for unregistered custom metrics. CloudWatch does not export help text.

OTEL count units derive a singular item annotation from the final metric-name component using the existing inflection dependency: custom leaves such as `queries` and `statuses` export `{query}` and `{status}`. Singular and uncountable leaves retain UCUM unit `1`; physical units are unaffected.

OTEL instrument units and descriptions are resolved only when a counter, gauge, or histogram is first registered, using the writer's existing instrument caches. Repeated samples do not repeat count-unit inflection or description lookup.

## Writer API and custom metrics

The metric writer API now takes the canonical namespace explicitly:

```go
func NewWriter(namespace string) Writer
func NewWriterWithInterfaces(clock clock.Clock, channel *metricChannel, namespace string) Writer
```

`metric.NewWriter(namespace)` stamps its namespace onto each written datum whose `Datum.Namespace` is
empty; an explicit datum namespace takes precedence. The writer does not register or amend metric
definitions. Every real sample must provide its own `Priority` and `Unit`; framework samples must
provide their `Kind` (including histogram buckets where applicable). Empty startup and empty-flush
events emit no samples. A missing series means no observation has been written; it is not a measured
zero. Genuine zero-valued gauge observations remain valid. The Kafka producer and Kinesis record
writer set complete sample metadata explicitly.

An empty effective namespace (for example, through `metric.NewWriter("")`) omits only the canonical
namespace; it does not preserve backend names. Writers still apply their own rendering rules:

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

Kafka producer datums carry complete canonical namespace, unit, instrument kind, and histogram bounds
when written to an injected backend writer directly. Batch-size and produce-duration observations
therefore retain their correct identities and types without any registered zero samples.

### Metric and dimension constant names

Go constant identifiers now follow their canonical values, including plural event names, duration
and size suffixes, and attribute qualifiers. This source-only correction does not change emitted
metric names, dimensions, units, histogram boundaries, or schema `v2.0`.

Update callers of these exported constants; the old identifiers have no compatibility aliases:

| Package | Previous identifier | Current identifier | Unchanged value |
|---|---|---|---|
| `pkg/httpserver` | `MetricHttpConcurrentRequests` | `MetricHttpActiveRequestCount` | `active_request.count` |
| `pkg/httpserver` | `MetricHttpOpenConnections` | `MetricHttpConnectionCount` | `connection.count` |
| `pkg/httpserver` | `MetricHttpRequestsRejected` | `MetricHttpRejectedRequests` | `rejected.requests` |
| `pkg/grpcserver` | `MetricRpcServerDuration` | `MetricRpcServerRequestDuration` | `request.duration` |
| `pkg/kafka` | `DimensionBroker` | `DimensionBrokerAddress` | `broker.address` |
| `pkg/kafka` | `DimensionClient` | `DimensionClientName` | `client.name` |
| `pkg/kafka` | `DimensionTopic` | `DimensionTopicName` | `topic.name` |
| `pkg/kafka` | `DimensionPartition` | `DimensionPartitionId` | `partition.id` |
| `pkg/kafka` | `MetricNameBrokerThrottleTime` | `MetricNameBrokerThrottleDuration` | `throttle.duration` |
| `pkg/kafka` | `MetricNameBrokerThrottleCount` | `MetricNameBrokerThrottles` | `throttles` |
| `pkg/kafka` | `MetricNameProduceBatchBytes` | `MetricNameProduceBatchSize` | `produce.batch.size` |
| `pkg/kafka` | `MetricNameProduceBatchBytesCmp` | `MetricNameProduceBatchCompressedSize` | `produce.batch.compressed.size` |
| `pkg/kafka` | `MetricNameFetchBatchBytes` | `MetricNameFetchBatchSize` | `fetch.batch.size` |
| `pkg/kafka` | `MetricNameFetchBatchBytesCmp` | `MetricNameFetchBatchCompressedSize` | `fetch.batch.compressed.size` |
| `pkg/limit` | `MetricNameRateLimitTake` | `MetricNameRateLimitTakes` | `takes` |

## HTTP metric migrations

HTTP client request-start, completion, and server response volumes are restored as independent Counter instruments; the duration histograms remain latency distributions. The previous entries below are source-level metric-family identifiers, not literal v1 exporter names; the old exported names depended on the writer and, for some writers, application identity.

| Previous source-level metric family | V2 metric and migration |
|---|---|
| HTTP client `HttpClientRequest`, `HttpClientError`, `HttpClientResponseCode1XX` through `HttpClientResponseCode5XX`, and `HttpRequestDuration` | `http.client.requests` counts validated requests immediately before `Execute`, keyed by `request.method`; `http.client.completed.requests` counts when `Execute` returns, keyed by method, full status code, bounded status class, and `error.type`. Transport errors and cancellations use the default marker for absent code/class; failures use `request_failed` or `request_canceled`. Use these Counters for volume, not the duration histogram's `_count` or `_sum`; for former `HttpClientError` queries select `error.type="request_failed"` because the old error family excluded cancellations. |
| HTTP server `HttpRequestCount`, `HttpRequestCountPerRoute`, `HttpStatus1XX` through `HttpStatus5XX`, and `HttpRequestResponseTime` | `http.server.requests` and `http.server.routed.requests` preserve the statusless request counts; `http.server.responses` and the raw `http.server.routed.responses` Counter count by full status code plus bounded `response.status_class`. CloudWatch also gets a same-name, four-dimension class projection for `http.server.routed.responses`; see the query guidance below. `http.server.request.duration` remains the route/method/full-status latency histogram, not a request-volume Counter. |
| HTTP server `HttpRequestsRejected` | `http.server.rejected.requests` retains its detailed `name`, `route`, and `request.method` Counter; CloudWatch also receives a same-name server-wide `KindTotal` `UnitCount` projection with exactly `name` and `Sum` statistics. |

The existing server Counters are high-priority monotonic `UnitCount` Counters, emitted for matched requests: `requests` has the server dimension and `routed.requests` also has the application route and method. They count each matched request once, including requests aborted by downstream server middleware; unmatched Gin routes remain excluded. Matched built-in health routes count. No startup series is emitted for an idle server or route. The `responses` and raw `routed.responses` Counters emit on matched responses, including responses from aborted downstream requests and matched built-in health routes, while unmatched Gin routes remain excluded. Each raw Counter carries the full numeric code and bounded class (`1xx`–`5xx`), with route and method on the routed metric. The latency histogram remains separate. For example, Prometheus server response volume can use `sum(gosoline_http_server_responses_total{response_status_class="5xx"})`.
Series absence now means no sample was observed, not that the measured value was zero. Migrate
queries and dashboards that relied on seeded idle zeros: select the real emitted samples for event
counts, and use an explicit query-side fill only when a presentation requires displaying zero during
an interval with no observations. Real zero-valued gauge observations remain valid samples.

For CloudWatch only, every matched routed response Counter observation emits exactly one additional same-name `KindTotal` datum with the same event boundary, value, unit, and priority. It retains `name`, `route`, `request.method`, and the observed class; `response.status_code` is `metric.DimensionDefault` and CloudWatch omits it. Only observed name/route/method/class tuples produce a class series. OpenTelemetry skips `KindTotal` before instrument lookup and Prometheus before registry registration, so both retain only the raw five-dimension Counter; the latency histogram remains separate. To query raw CloudWatch Counters, use the exact five-dimension `FROM SCHEMA(namespace, name, route, request.method, response.status_code, response.status_class)` with the class filter; broad queries can include both shapes and double-count. A class-only route total reads the projection with standard CloudWatch MetricStat `Sum` and exactly `name`, `route`, `request.method`, and `response.status_class` dimensions (for example, `response.status_class="5xx"`), with no `response.status_code`. Former topic-only Kafka `KindTotal` points remain removed.

For CloudWatch only, each matched rejected request also emits exactly one same-name `KindTotal` datum with the same value, unit, priority, and event boundary, dimensioned by exactly `name`. This server-level `UnitCount` projection uses `Sum`; the detailed route/method Counter remains unchanged. OpenTelemetry and Prometheus skip `KindTotal` and export only the detailed Counter, so the projection does not double-count in those exporters.

The exported `pkg/httpserver` constants `MetricHttpRequestCount`, `MetricHttpRequestCountPerRoute`, `MetricHttpRequestResponseTime`, and `MetricHttpStatus` remain removed; no v1 Go aliases are restored. The current `MetricHttpRejectedRequests` constant names the leaf `rejected.requests`.

`http.client.requests` and `http.client.completed.requests` emit only on their documented request-start and completion events. A successful response carries the numeric code and class with the default `error.type`; a transport failure or cancellation carries `response.status_code="{{default}}"` and `response.status_class="{{default}}"`, with `error.type="request_failed"` or `"request_canceled"`, respectively. In Prometheus, the default `error.type` label is `error_type=""`; OTEL omits the attribute and CloudWatch omits the default dimension. A 5xx-response volume query can use the rendered counter `sum(gosoline_http_client_completed_requests_total{response_status_class="5xx"})`. The duration histogram continues to record latency separately. Absence of a series indicates no matching event, not an observed zero.

## Restored counter migrations

| Counter | Boundary and migration guidance |
|---|---|
| `db.repo.operations` | Counts once after each SQL repository `Create`, `Read`, `Update`, `Delete`, or `Query` returns; dimensions are `operation.name`, `model.id`, and `error.type`. Success uses the default marker, failure uses `operation_failed`. Keep `db.repo.operation.duration` for latency. |
| `ddb.operations` | Counts once after the wrapped DynamoDB save returns, with `operation.name="save"`, `model.id`, and `error.type`; success uses the default marker, failure uses `operation_failed`. Keep `ddb.operation.duration` for latency. |
| `rpc.server.requests` | Counts every gRPC call at `stats.End`, including failed calls, with `service` and `method`. `rpc.server.request.duration` remains the latency histogram. |
| `kvstore.attempted.reads` | Counts attempted read items with `model.id` and `store.type`: `Contains`/`Get` before delegation; `GetBatch` after key conversion and before delegation, by key count. Delegated failures count; conversion failures do not. Existing `kvstore.reads{hit="true"|"false"}` remains successful reads only. |
| `limit.attempted.takes`, `limit.takes`, `limit.throttled.takes` | `attempted.takes` counts at `OnTake`; `takes` carries only mutually exclusive terminal `outcome="allowed"` or `"error"`; `throttled.takes` counts at `OnThrottle`. All are keyed by `name` and `prefix`. Do not sum them as outcomes or use a throttled terminal outcome; `outcome="throttled"` and the exported `OutcomeThrottled` symbol are removed without an alias. |
| `kafka.produced.records`, `kafka.produced.batch.size`, `kafka.produced.batch.compressed.size`, `kafka.fetched.records`, `kafka.fetched.batch.size`, `kafka.fetched.batch.compressed.size` | High-priority Counters at `OnProduceBatchWritten`/`OnFetchBatchRead`, keyed by `client.type`, `client.name`, `topic.name`, and `partition.id`. Record counters add `NumRecords`; `.batch.size` adds `UncompressedBytes`; `.batch.compressed.size` adds `CompressedBytes`. Use them for hook-level volume; existing batch histograms remain distributions, and application `sent.messages`/`consumed.messages` count at different boundaries. |

`ddb.operation.duration` measures elapsed time from immediately before the wrapped DynamoDB save
call until it returns, in milliseconds for both successful and failed calls. This fixes a
zero-valued timer; the metric name, dimensions, instrument kind, unit, and database-duration bucket
boundaries are unchanged.

For restored counter volumes, query the Counter instrument rather than using a histogram's `_count` or `_sum`; a histogram remains the distribution of its observations. `error.type="{{default}}"` is rendered as Prometheus `error_type=""`, as an absent OTEL attribute, and as an omitted CloudWatch dimension.

## Canonical name migrations

The former names below are canonical/logical migration keys, not exact v1 exporter names; the old writer and application namespace determined rendered series. Re-key consumer queries and dashboards accordingly.

| Former canonical/logical family | V2 canonical metric(s) and migration |
|---|---|
| `metric.records` | `metric.log.records` |
| SQL repository `db.client.operation.duration` | Keep `db.repo.operation.duration` for latency; use `db.repo.operations` for operation volume. |
| DynamoDB repository `db.client.operation.duration` | Keep `ddb.operation.duration` for latency; use `ddb.operations` for operation volume. |

## Outcome metrics and writer-specific queries

Most standalone failure or alternative-outcome families use the metric for the operation, distinguished by dimensions. Limiter attempts, terminal outcomes, and throttles remain separate counters because they are counted at distinct event boundaries; see “Restored counter migrations”. The left column is a logical family name or source identifier, not an exact old exported name; v1 names varied by writer and application namespace.

| Earlier family or source identifier | V2 metric | Outcome dimension |
|---|---|---|
| `stream.errors` | `stream.consumed.messages` | `error.type` |
| `kafka.producer.send.errors` | `kafka.producer.sent.messages` | `error.type` |
| `cloud.aws.kinesis.send.errors` | `cloud.aws.kinesis.sent.messages` | `error.type`, using the reason Kinesis reported |
| `cloud.aws.kinesis.consume.errors` | `cloud.aws.kinesis.consumed.messages` | `error.type`; see the overlapping total below |
| `kvstore.hits` | `kvstore.reads` | `hit` (`true` or `false`) |
| `limit.releases`, `limit.throttles`, `limit.errors` | `limit.attempted.takes`, `limit.takes`, and `limit.throttled.takes` | `attempted.takes` records starts; terminal `takes` has only `allowed`/`error`; throttles are separate and must not be double-counted. |
| `mdlsub.consumed.events`, `mdlsub.skipped.events`, `mdlsub.consume.errors` | `mdlsub.events` | `outcome` (`applied` or `skipped`) and, for failures, `error.type` |
| `kafka.consumer.consume.errors` | `kafka.consumer.consumed.messages` | `error.type` |
| `ModelEventNotifySuccess` and `ModelEventNotifyFailure` | `db.repo.model_event.notifications` | The success path uses the internal default marker; failures use `error.type="notify_failed"` |

The notification success and failure families are both consolidated into `db.repo.model_event.notifications`, a `UnitCount` counter. There is no separate success label and no error-only replacement metric.

`kafka.consumer.consumed.messages` is a `UnitCount` counter. Each emitted outcome sample carries its unit and counter kind, including `error.type="processing_failed"` samples. No zero-value sample is emitted while idle; an absent series means no event has been observed. OTEL exports `gosoline.kafka.consumer.consumed.messages` as a monotonic counter with unit `{message}`; Prometheus exports `gosoline_kafka_consumer_consumed_messages_total` as a counter.

`kafka.producer.sent.messages` uses a `UnitCount` counter for both successful records on the default series and failed
records with `error.type="send_failed"`.

`metric.DimensionDefault` (`"{{default}}"`) is an internal marker, not a value exported by a writer. Its backend representation is:

| Writer | Default representation | Example success/total selection |
|---|---|---|
| Prometheus | Empty label value; `error.type` renders as `error_type` | `error_type=""` |
| OTEL | The `error.type` attribute is absent | Select points with no `error.type` attribute |
| CloudWatch | The default dimension is omitted | Select the series without that dimension |

For notification failures, select `error_type="notify_failed"` in Prometheus, the `error.type="notify_failed"` attribute in OTEL, or the CloudWatch series carrying that error dimension. For other outcome metrics, use the same writer-specific representation for their dimensions.

Sum exclusive outcome series only within the same metric's event domain. Do not combine the separate limiter attempt, terminal, and throttled counters as one outcome set. `stream.consumed.messages` and `cloud.aws.kinesis.consumed.messages` are exceptions: their default series is the all-processed total, while `error.type="processing_failed"` is an overlapping failed-record subset. Do not add or subtract that subset to infer successful records. Error values name the failed action, not a Go error type.

## Stream and Kinesis semantics

Single-message and batch stream consumers share the v2 metrics `stream.process.duration`, `stream.consumed.messages`, and `stream.retry.operations`:

- `stream.process.duration` records one observation per processing invocation, including a batch invocation; it is not one duration observation per message.
- `stream.consumed.messages` records the all-processed per-invocation message total with the internal
  default marker: a single-message invocation counts one, and a batch invocation counts its messages.
  Actual non-canceled processing errors emit `error.type="processing_failed"` as an overlapping
  subset; batch callback errors count only records with false or missing acknowledgements. A false
  or missing acknowledgement with a nil callback error can request deferred processing or retry and
  no longer counts as a processing failure. Acknowledgement and retry behavior is unchanged.
  Every `GetModel` failure also emits the bounded `error.type="model_lookup_failed"` diagnostic,
  including failures configured to be ignored. Non-ignored lookup failures overlap `processing_failed`;
  ignored ones do not. Both diagnostics overlap the all-processed default total. Filter
  `error_type="processing_failed"` for Prometheus processing-error totals rather than summing every
  nondefault series; the default selector remains empty `error_type=""` in Prometheus, absent
  `error.type` in OTEL, and an omitted CloudWatch dimension.
- `stream.retry.operations` records retry-handler get/put operations by attempt, with `operation="get"` or `"put"` and `error.type="{{default}}"` on normal attempts. A failed retry-handler put also emits `error.type="retry_handler_put_failed"` on the same put series; this is an overlapping diagnostic subset, not another attempt. Filter the default error type for total attempts. |

The test logger's error diagnostics now support `%w` wrapping like the production logger, instead of
printing `%!w(...)`. This corrects captured failure logs, including stream retry-handler failures;
production error formatting, error identity, severity and cancellation behavior are unchanged.

The Kinesis record writer reports each `PutRecords` attempt. `cloud.aws.kinesis.sent.messages` counts successful records on the default series and failed records by the error reason Kinesis returned. `cloud.aws.kinesis.batch.records` is a histogram of records in each attempt; a retried record is counted again in the subsequent attempt, so this is attempt-level work rather than a unique-record total. The failed-record subset and retry behavior remain part of the producer contract. The shard reader retains its detailed stream-and-shard metrics and restores same-name CloudWatch-only `KindTotal` projections that remove only `partition.id`:

| Metric | Projection dimensions | Unit | CloudWatch statistic |
|---|---|---|---|
| `cloud.aws.kinesis.acquire.duration` | `stream.name` | seconds (`UnitSecondsMaximum`) | Maximum |
| `cloud.aws.kinesis.sleep.duration` | `stream.name` | ms (`UnitMillisecondsAverage`) | Average |
| `cloud.aws.kinesis.lag` | `stream.name` | ms (`UnitMillisecondsMaximum`) | Maximum |
| `cloud.aws.kinesis.process.duration` | `stream.name` | ms (`UnitMillisecondsAverage`) | Average |
| `cloud.aws.kinesis.reads` | `stream.name` | count (`UnitCount`) | Sum |
| `cloud.aws.kinesis.wait.duration` | `stream.name` | ms (`UnitMillisecondsAverage`) | Average |
| `cloud.aws.kinesis.consumed.messages` (`error.type="{{default}}"`) | datum: `stream.name`, default `error.type`; CloudWatch: `stream.name` | count (`UnitCount`) | Sum |
| `cloud.aws.kinesis.consumed.messages` (`error.type="processing_failed"`) | `stream.name`, `error.type="processing_failed"` | count (`UnitCount`) | Sum |

Kinesis `reads` and `consumed.messages` stream projections appear only when their corresponding real detailed samples are observed; default-valued dimensions are omitted in CloudWatch. The `processing_failed` projection is emitted only for observed failures, and lag has no synthetic default.

Kinsumer no longer emits `ShardTaskRatio` per stream or the application-wide `ShardTaskRatioMax`. The former ratio was `shardCount / clientCount * 100`; the v2 `cloud.aws.kinesis.shard.count` and `cloud.aws.kinesis.client.count` gauges are raw per-stream counts, each dimensioned by `stream.name`. Derive `shard.count / client.count` per stream (ideal ratio `1`, not `100`), then take the maximum across streams when global scaling needs one value; update thresholds rather than carrying forward the old ×100 scale. These are stream gauges, not calculator output. `cloud.aws.kinesis.lag` remains a per-stream/per-shard cumulative histogram in OTEL, with millisecond-authored observations exported in seconds; CloudWatch retains milliseconds and the existing Maximum reducer for detailed and stream-only lag series.

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

Source review lists 80 framework-authored canonical metric leaves across 18 namespaces for schema `v2.0`, including all 30 retained histograms and 16 restored Counter leaves. These additions and corrected semantics remain within the still-unmerged v2-introducing MR; the schema version stays `v2.0`.

| Namespace | Canonical leaves |
|---|---|
| `blob` | `batch.operations` |
| `cloud.aws.kinesis` | `consumed.messages`; `sent.messages`; `process.duration`; `reads`; `lag`; `acquire.duration`; `sleep.duration`; `wait.duration`; `shard.count`; `client.count`; `batch.records` |
| `conc.scheduler` | `batch.tasks`; `task.queue.duration` |
| `db.client` | `connection.count`; `connections` |
| `db.repo` | `operation.duration`; `operations`; `model_event.notifications` |
| `ddb` | `operation.duration`; `operations` |
| `http.client` | `request.duration`; `requests`; `completed.requests` |
| `http.server` | `request.duration`; `requests`; `routed.requests`; `responses`; `routed.responses`; `rejected.requests`; `active_request.count`; `connection.count` |
| `kafka` | `connects`; `throttles`; `throttle.duration`; `produce.batch.records`; `produce.batch.size`; `produce.batch.compressed.size`; `fetch.batch.records`; `fetch.batch.size`; `fetch.batch.compressed.size`; `produced.records`; `produced.batch.size`; `produced.batch.compressed.size`; `fetched.records`; `fetched.batch.size`; `fetched.batch.compressed.size` |
| `kafka.consumer` | `consumed.messages`; `process.duration`; `polls`; `poll.duration`; `commit.duration`; `sleep.duration`; `rebalances` |
| `kafka.producer` | `sent.messages`; `produce.duration`; `batch.records` |
| `kvstore` | `reads`; `attempted.reads`; `writes`; `deletes`; `item.count` |
| `limit` | `attempted.takes`; `takes`; `throttled.takes` |
| `mdlsub` | `events` |
| `metric` | `log.records` |
| `rpc.server` | `request.duration`; `requests` |
| `smpl` | `decisions` |
| `stream` | `consumed.messages`; `process.duration`; `retry.operations`; `produced.messages`; `batch.messages`; `aggregate.messages`; `idle.duration`; `message.count`; `reads`; `writes` |

The full metric specification, including units, kinds, descriptions, and dimensions, is in `pkg/metric/SEMCONV.md`.

## Existing examples

The existing `examples/otel` application now passes a namespace to `metric.NewWriter`. The v2 writer exports its millisecond-authored duration metric in seconds.

The Prometheus example now supplies a valid count unit for its custom summary. Its tutorial links to the public metric specification and shows the fixed `gosoline` prefix and observed custom-series output.

## Upgrade checklist

1. Read `metric.schema_version`; migrate consumers to schema `v2.0`. Do not expect v1 aliases or dual emission.
2. Re-key metric queries, dashboards, alerts, and recording rules against the canonical inventory; update dimensions and units as well as names.
3. Use `http.server.requests` for per-server HTTP request counts and `http.server.routed.requests` for route/method request counts; use raw `http.server.responses` and `http.server.routed.responses` Counters for full numeric status counts, and use the CloudWatch-only `http.server.rejected.requests` stream projection with `name`/`Sum` while retaining its detailed route/method Counter in other exporters. Use `http.client.requests` for starts and `http.client.completed.requests` for completions/statuses; retain duration histograms for latency only. For raw 5xx CloudWatch Metrics Insights queries, select the exact five-dimension `FROM SCHEMA(namespace, name, route, request.method, response.status_code, response.status_class)` before filtering `response.status_class="5xx"`; broad queries can include the routed four-dimension projection and double-count. Read the observed class projection for a 5xx total with a standard CloudWatch MetricStat `Sum` and exactly `name`, `route`, `request.method`, and `response.status_class="5xx"` dimensions, without `response.status_code`. For topic-level Kafka views aggregate retained partition series; its former topic-only `KindTotal` stays removed. Use the restored Kinesis shard-reader projections for stream-level views.
4. Replace standalone outcome metrics with operation metrics and writer-appropriate `error.type`/default selectors. Preserve the overlapping stream and Kinesis consumer failure-subset semantics.
5. Update `metric.NewWriter` and `NewWriterWithInterfaces` calls, Prometheus writer construction, and removed Prometheus naming settings. `NewWriter` now accepts only a namespace string; migrate middleware construction to `NewMetricMiddleware`, which returns only a `gin.HandlerFunc` and no setup callback. Migrate Kafka callers to `kafka.MetricPair(kafka.MetricSpec{...})` and update removed or changed Kafka/HTTP metric constants. Re-key custom metrics for each backend's rendering.
6. Remove the calculator option and exact configuration keys above; replace per-runner scaling inputs with application-specific signals and derive Kinesis capacity ratios from the shard/client gauges.
7. Update duration dashboards for seconds in Prometheus and OTEL, preserve CloudWatch's authored units, and use histogram queries for framework duration distributions while retaining custom `KindSummary` queries. Use the restored operation/event Counters—not histogram `_count`/`_sum`—for their documented volume totals.
