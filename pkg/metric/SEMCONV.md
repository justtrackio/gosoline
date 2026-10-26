# Gosoline metric semantic conventions

Gosoline-owned metrics follow the OpenTelemetry semantic conventions for **grammar, units and
attribute shape**, but deliberately do **not** use their names. Each has a Gosoline namespace, which
the OTEL renderer prefixes with `gosoline.` so the exported names cannot be mistaken for canonical
semantic-convention metrics. A custom metric without a namespace is normalized from its leaf without
that prefix; application authors should avoid canonical semantic-convention names.

This file is the human-facing metric specification. **Any change to a metric name, attribute key, unit
or instrument type updates this file in the same commit.** The emitting packages and their registered
defaults are the source of what is written; the inventory below is maintained manually from that
source and the writer behavior. Focused behavioral tests verify runtime emission and rendering, not
this document, so keep the inventory and exceptions current alongside the source changes.

## Why the names diverge

A canonical semantic-convention name is a contract about exactly what is measured, with which
attributes, at which point. Prefixing or extending one produces a *different* metric that tooling will
not recognise as the convention's, so a half-adopted convention is worse than none: a dashboard built on
`http.server.request.duration` would silently read a metric whose attribute set gosoline chose.

Emitting under gosoline's own names makes the divergence explicit and lets the grammar stay uniform
across every subsystem, which is what actually makes the metrics learnable.

## The grammar

The authored names follow the grammar defined here:

1. The namespace names the owner; never repeat it in the leaf.
2. Event counter: `[qualifier.]<plural-noun>`, qualifier a past participle.
3. Current amount: `<thing>.count`, on a gauge.
4. Elapsed time: `<operation>.duration`, on a histogram, in milliseconds.
5. Bytes: `<thing>.size`. Items per batch: `<thing>.<plural-noun>`.
6. Failure is an `error.type` attribute on the operation metric; `{{default}}` normally marks success,
   except for explicitly documented all-processed totals.
7. Components lowercase, words joined by `_`, hierarchy by `.`.
8. An attribute key never repeats its metric's namespace.

## Authored names and writer rendering

A datum's canonical name consists of an optional namespace and a metric leaf, supplied through
`Datum.Namespace` and `Datum.MetricName`. Gosoline-owned metrics use the grammar above.
`metric.NewWriter(namespace)` stamps its namespace onto datums whose `Namespace` is empty; an
explicit datum namespace takes precedence. An empty writer namespace leaves the metric unnamespaced.
An application can choose its own namespace and leaf for custom metrics; the same writer-specific
rendering rules below apply to them.

The canonical name never includes a backend-specific prefix, a unit suffix, `_total`, or an attribute
value. For example, a writer created with namespace `example` and a count counter named `api_request`
has the canonical name `example.api_request`.

| Writer | Rendering of `http.server` + `request.duration` (`ms`) |
|---|---|
| CloudWatch | `HttpServerRequestDuration`, in milliseconds |
| Prometheus | `gosoline_http_server_request_duration_seconds`, observations scaled to seconds |
| OpenTelemetry | `gosoline.http.server.request.duration`, instrument unit `s` |

The Prometheus writer always applies the fixed `gosoline` namespace; it is not derived from
application configuration. It renders the datum namespace as a subsystem and the metric leaf as the
name, replacing dots with underscores in both. It appends `_seconds` for duration units, `_bytes` for
byte units, and `_total` for counters (a count counter therefore gets `_total` without another unit
suffix). A count-valued summary has no name suffix; Prometheus emits its `_sum` and `_count` series.
With the example namespace above, `api_request` renders as
`gosoline_example_api_request_total`. With an empty namespace, it renders as
`gosoline_api_request_total`. Neither `app.name` nor `app.namespace` is added automatically.

Prometheus dimension keys become label names by replacing dots with underscores; they are not
prefixed by the metric namespace. For example, `request.method` becomes `request_method`, while
`route` and `name` remain unchanged. A value of `{{default}}` becomes the empty label value.

Registered defaults are matched using canonical names and dimensions before this label rendering, so translating dotted
keys and the internal `{{default}}` marker does not change which metadata applies to a datum.

OpenTelemetry preserves canonical dots between namespace and leaf and prefixes the name with
`gosoline.` when a namespace is present. Without a namespace, it normalizes and exports only the
leaf name: for example, a namespace-less custom metric with `MetricName: "api_request"` is exported
as `api_request`, without a `gosoline.` prefix. Its unit is expressed on the instrument. CloudWatch
concatenates the canonical namespace and leaf in PascalCase and does not add a metric-name prefix; its
CloudWatch namespace identifies the application.

Explicit bounds set with `KindHistogram.WithBuckets` are authored in the datum's unit. Prometheus and OpenTelemetry
scale them with observations when registering the histogram, so duration bucket bounds in milliseconds are converted
to seconds along with the observed values.

## Attributes

`error.type` is the **only** key gosoline takes verbatim from a semantic convention, because its values
are what make an operation metric self-describing. Every other key is gosoline's own and carries no
namespace prefix.

| Key | Meaning | Semconv equivalent |
|---|---|---|
| `error.type` | what failed; `{{default}}` usually means success except for documented all-processed totals | `error.type`, used verbatim |
| `broker.address` | Kafka broker address, used on broker-connect and throttle metrics | none (`server.address` is close) |
| `client.name` | Kafka client name | none |
| `client.type` | `consumer` or `producer` | none |
| `connection.state` | database connection-pool state: `used` or `idle` | derived from `db.client.connection.state` |
| `consumer.name` | stream consumer name | none |
| `hit` | whether a key-value read was served from the store's own data | none |
| `level` | log level a record was written at | none |
| `list.name` | Redis list a message was read from or written to | derived from `messaging.destination.name` |
| `method` | gRPC method | derived from `rpc.method` |
| `model.id` | model an operation applies to | none |
| `name` | name of the owner (HTTP server, scheduler, rate limiter) | none |
| `operation` | blob operation | none |
| `operation.name` | database operation | derived from `db.operation.name` |
| `outcome` | how an operation ended, where success has more than one shape | derived from `*.operation.result` |
| `partition.id` | Kafka partition or Kinesis shard | derived from `messaging.destination.partition.id` |
| `prefix` | rate limiter prefix | none |
| `producer.name` | producer daemon name | none |
| `request.method` | HTTP method | derived from `http.request.method` |
| `response.status_code` | HTTP status code | derived from `http.response.status_code` |
| `response.status_class` | bounded HTTP response class derived from `response.status_code` (`1xx` through `5xx`) | none |
| `route` | HTTP route template | derived from `http.route` |
| `sampled` | whether the sampler kept the item | none |
| `service` | gRPC service | derived from `rpc.service` |
| `store.type` | key-value store implementation | none |
| `stream.name` | Kinesis stream | derived from `messaging.destination.name` |
| `topic.name` | Kafka topic | derived from `messaging.destination.name` |

"Derived from" means the value carries the same meaning as that convention attribute with the namespace
prefix stripped. It is **not** that attribute, and tooling keyed on the convention will not find it.

## The inventory

80 metric leaves across 18 namespaces, including all 30 existing histograms. The 16 restored leaves below
are counter instruments. `unit` is the authored unit; every writer renders it into its own convention (Prometheus
scales durations to seconds and appends `_seconds`, OTEL sets the UCUM unit on the instrument).

### `stream` — `pkg/stream`

| Metric | Unit | Type | Attributes | Derived from |
|---|---|---|---|---|
| `process.duration` | ms | histogram | `consumer.name` | `messaging.process.duration` |
| `consumed.messages` | count | counter | `consumer.name`, `error.type` | `messaging.client.consumed.messages` |
| `retry.operations` | count | counter | `consumer.name`, `operation`, `error.type` | none |
| `produced.messages` | count | counter | `producer.name` | none |
| `batch.messages` | count | histogram | `producer.name` | none |
| `aggregate.messages` | count | histogram | `producer.name` | none |
| `idle.duration` | ms | histogram | `producer.name` | none |
| `message.count` | count | gauge | `list.name` | none |
| `reads` | count | counter | `list.name` | none |
| `writes` | count | counter | `list.name` | none |

Single and batch consumers use the same instruments and attributes. `process.duration` records one
observation per processing invocation: one message for a single consumer, the whole batch for a batch
consumer. It is not a per-message duration when emitted by a batch consumer.

The default `consumed.messages` series remains the all-processed total, including callback failures; a
batch contributes its record count, not one. `error.type="processing_failed"` is an overlapping
record-based failure series, including false or missing acknowledgements. Canceled processing does
not emit that failure series. Every `GetModel` failure also emits `error.type="model_lookup_failed"`,
including failures configured to be ignored. A non-ignored lookup failure overlaps
`processing_failed`; an ignored one does not. Both diagnostic series overlap the default total and must
not be summed with it or subtracted to infer successes. For processing-error totals, filter
`error.type="processing_failed"` rather than summing all diagnostic values.

`retry.operations` counts retry-handler operations with `operation="get"` or `operation="put"`, not
records. Normal attempts carry `error.type="{{default}}"`; a failed retry-handler put also emits
`error.type="retry_handler_put_failed"` on the same `operation="put"` series. That marker is an
overlapping diagnostic subset, not an extra attempt. Select the default error type for total attempts.

### `kafka.consumer` — `pkg/kafka/consumer`

| Metric | Unit | Type | Attributes | Derived from |
|---|---|---|---|---|
| `consumed.messages` | count | counter | `client.type`, `client.name`, `topic.name`, `error.type` | `messaging.client.consumed.messages` |
| `process.duration` | ms | histogram | `client.type`, `client.name`, `topic.name` | `messaging.process.duration` |
| `polls` | count | counter | `client.type`, `client.name`, `topic.name` | none |
| `poll.duration` | ms | histogram | `client.type`, `client.name`, `topic.name` | none |
| `commit.duration` | ms | histogram | `client.type`, `client.name`, `topic.name` | none |
| `sleep.duration` | ms | histogram | `client.type`, `client.name`, `topic.name` | none |
| `rebalances` | count | counter | `client.type`, `client.name`, `topic.name` | none |

### `kafka.producer` — `pkg/kafka/producer`

| Metric | Unit | Type | Attributes | Derived from |
|---|---|---|---|---|
| `sent.messages` | count | counter | `client.type`, `client.name`, `topic.name`, `error.type` | `messaging.client.sent.messages` |
| `produce.duration` | ms | histogram | `client.type`, `client.name`, `topic.name` | `messaging.client.operation.duration` |
| `batch.records` | count | histogram | `client.type`, `client.name`, `topic.name` | none |

`sent.messages` records successful records on the default series and failed records with `error.type="send_failed"`.
Both outcomes use a count-valued counter.

### `kafka` — `pkg/kafka`

Broker-connect and throttle metrics from the franz-go hooks carry `client.type`, `client.name`, and
`broker.address`; broker-connect metrics also carry `error.type`. Produce/fetch batch hooks instead
carry `client.type`, `client.name`, `topic.name`, and `partition.id`, not `broker.address`.

| Metric | Unit | Type |
|---|---|---|
| `connects` | count | counter |
| `throttles` | count | counter |
| `throttle.duration` | ms | histogram |
| `produce.batch.records` | count | histogram |
| `produce.batch.size` | bytes | histogram |
| `produce.batch.compressed.size` | bytes | histogram |
| `fetch.batch.records` | count | histogram |
| `fetch.batch.size` | bytes | histogram |
| `fetch.batch.compressed.size` | bytes | histogram |
| `produced.records` | count | counter |
| `produced.batch.size` | bytes | counter |
| `produced.batch.compressed.size` | bytes | counter |
| `fetched.records` | count | counter |
| `fetched.batch.size` | bytes | counter |
| `fetched.batch.compressed.size` | bytes | counter |

The six new high-priority counters use the same `OnProduceBatchWritten` and `OnFetchBatchRead` hooks
as the existing batch histograms, with `client.type`, `client.name`, `topic.name`, and `partition.id`.
Produced/fetched record counters add `NumRecords`; `.batch.size` adds `UncompressedBytes`; and
`.batch.compressed.size` adds `CompressedBytes`. The histograms remain per-batch distributions; the
counters are independent volume instruments, have no zero defaults, and are the source for total
record/byte volume. Do not use a histogram's `_count` or `_sum` as a substitute for these counters.
These hook boundaries are distinct from the application-level
`kafka.producer.sent.messages` and `kafka.consumer.consumed.messages` metrics.

### `cloud.aws.kinesis` — `pkg/cloud/aws/kinesis`

| Metric | Unit | Type | Attributes | Derived from |
|---|---|---|---|---|
| `consumed.messages` | count | counter | `stream.name`, `partition.id`, `error.type` | `messaging.client.consumed.messages` |
| `sent.messages` | count | counter | `stream.name`, `error.type` | `messaging.client.sent.messages` |
| `process.duration` | ms | histogram | `stream.name`, `partition.id` | `messaging.process.duration` |
| `reads` | count | counter | `stream.name`, `partition.id` | none |
| `lag` | ms | histogram | `stream.name`, `partition.id` | `messaging.kafka.consumer.group.lag` in spirit |
| `acquire.duration` | s | histogram | `stream.name`, `partition.id` | none |
| `sleep.duration` | ms | histogram | `stream.name`, `partition.id` | none |
| `wait.duration` | ms | histogram | `stream.name`, `partition.id` | none |
| `shard.count` | count | gauge | `stream.name` | none |
| `client.count` | count | gauge | `stream.name` | none |
| `batch.records` | count | histogram | `stream.name` | none |

The six generic shard-reader metrics in the table also emit a same-name CloudWatch-only `KindTotal` projection
beside each detailed stream-and-shard datum. The projection removes only `partition.id`; it keeps
The metric pipeline applies the existing unit reducers for these CloudWatch statistics:

| Metric | Projection dimensions | Unit | CloudWatch statistic |
|---|---|---|---|
| `acquire.duration` | `stream.name` | seconds (`UnitSecondsMaximum`) | Maximum |
| `sleep.duration` | `stream.name` | ms (`UnitMillisecondsAverage`) | Average |
| `lag` | `stream.name` | ms (`UnitMillisecondsMaximum`) | Maximum |
| `process.duration` | `stream.name` | ms (`UnitMillisecondsAverage`) | Average |
| `reads` | `stream.name` | count (`UnitCount`) | Sum |
| `wait.duration` | `stream.name` | ms (`UnitMillisecondsAverage`) | Average |

The detailed metrics keep their original `partition.id` dimensions and instrument kinds. These
projections are not new metric names: OpenTelemetry and Prometheus filter `KindTotal` and continue
exporting only the detailed stream-and-shard series. Existing raw count defaults for `reads` and the
all-processed `consumed.messages` series already seed their CloudWatch stream series when default
dimensions are omitted; do not register a duplicate projection zero. The `processing_failed`
projection has no corresponding raw default, so it has its own zero default. There is no lag default.
`sent.messages` carries the reason Kinesis gave as its `error.type`, not a Go type, because the API
reports a per-record error code rather than an error value.

RecordWriter explicitly emits `sent.messages` as a counter in this namespace for both successful
and failed records, including failure-only attempts. `batch.records` is a histogram of each
write attempt's batch size, including retries; it is not a Kinsumer read-batch metric.

`consumed.messages` is an exception to the usual success meaning of `error.type="{{default}}"`: for Kinesis,
that series is the full batch total emitted when processing completes, including failed records. Each failed
record is also emitted immediately as `error.type="processing_failed"`. Those failure points overlap the
default total; do not sum them with it or subtract them to infer successes. CloudWatch gets a same-name
`KindTotal` projection for each outcome: the default datum retains `error.type="{{default}}"` and CloudWatch
omits it, leaving exactly `stream.name`; `processing_failed` retains `stream.name` and
`error.type="processing_failed"`. Both use `UnitCount`/Sum, and the failed projection is an overlapping
subset of the total, not a separate outcome. The detailed Counter remains stream-and-shard scoped for
all exporters; OpenTelemetry and Prometheus skip the projection.

`lag` retains millisecond samples and `UnitMillisecondsMaximum` aggregation for CloudWatch, including the
stream-only projection. Its detailed `KindHistogram` instrument continues to expose the cumulative
maximum to OpenTelemetry for each stream-and-shard series.

### `kvstore` — `pkg/kvstore`

| Metric | Unit | Type | Attributes |
|---|---|---|---|
| `reads` | count | counter | `model.id`, `store.type`, `hit` |
| `attempted.reads` | count | counter | `model.id`, `store.type` |
| `writes` | count | counter | `model.id`, `store.type` |
| `deletes` | count | counter | `model.id`, `store.type` |
| `item.count` | count | gauge | `model.id`, `store.type` |

`attempted.reads` counts requested read items before delegation: `Contains` and `Get` add one each;
`GetBatch` adds the number of keys after successful key conversion and before delegation. Delegated
failures count, but key-conversion failures do not. It has a high-priority zero default per model and
store. Existing `reads{hit="true"}` and `reads{hit="false"}` count only successful reads and remain
unchanged; their sum is successful read volume, not attempted volume.

### `db.client`, `db.repo`, `ddb` — `pkg/db`, `pkg/db-repo`, `pkg/ddb`

| Metric | Unit | Type | Attributes | Derived from |
|---|---|---|---|---|
| `db.client.connection.count` | count | gauge | `connection.state` | `db.client.connection.count` |
| `db.client.connections` | count | counter | none | none |
| `db.repo.operation.duration` | ms | histogram | `operation.name`, `model.id`, `error.type` | `db.client.operation.duration` |
| `db.repo.model_event.notifications` | count | counter | `model.id`, `error.type` | none |
| `ddb.operation.duration` | ms | histogram | `operation.name`, `model.id`, `error.type` | `db.client.operation.duration` |
| `db.repo.operations` | count | counter | `operation.name`, `model.id`, `error.type` | none |
| `ddb.operations` | count | counter | `operation.name`, `model.id`, `error.type` | none |

`db.client.connections` increments once for each database driver `Open` attempt and has no attributes;
it is not a live pool-connection count. `db.client.connection.count` is a periodic gauge that reports
the pool's current `used` and `idle` connection counts as separate series keyed by `connection.state`
(derived from `db.client.connection.state`).

`db.repo.operations` increments after each SQL repository call returns for `Create`, `Read`, `Update`,
`Delete`, and `Query`; `ddb.operations` increments after the wrapped DynamoDB save (`operation.name="save"`)
returns. Both use `error.type="{{default}}"` on success and `"operation_failed"` on failure, with
high-priority zero defaults for both series per model and operation. The existing duration histograms
remain unchanged and are not replacements for these operation counters.

### `mdlsub` — `pkg/mdlsub`

| Metric | Unit | Type | Attributes |
|---|---|---|---|
| `events` | count | counter | `model.id`, `outcome` (`applied`, `skipped`), `error.type` |

### `http.server`, `http.client`, `rpc.server` — `pkg/httpserver`, `pkg/http`, `pkg/grpcserver`

| Metric | Unit | Type | Attributes | Derived from |
|---|---|---|---|---|
| `http.server.request.duration` | ms | histogram | `name`, `route`, `request.method`, `response.status_code` | `http.server.request.duration` |
| `http.server.requests` | count | counter | `name` | none |
| `http.server.routed.requests` | count | counter | `name`, `route`, `request.method` | none |
| `http.server.responses` | count | counter | `name`, `response.status_code`, `response.status_class` | none |
| `http.server.routed.responses` | count | counter | `name`, `route`, `request.method`, `response.status_code`, `response.status_class` | none |
| `http.server.rejected.requests` | count | counter | `name`, `route`, `request.method` | none |
| `http.server.active_request.count` | count | gauge | `name` | `http.server.active_requests` |
| `http.server.connection.count` | count | gauge | `name` | none |
| `http.client.requests` | count | counter | `request.method` | none |
| `http.client.completed.requests` | count | counter | `request.method`, `response.status_code`, `response.status_class`, `error.type` | none |
| `http.client.request.duration` | ms | histogram | `request.method`, `response.status_code`, `error.type` | `http.client.request.duration` |
| `rpc.server.requests` | count | counter | `service`, `method` | none |
| `rpc.server.request.duration` | ms | histogram | `service`, `method` | `rpc.server.duration` |

The statusless HTTP server counters remain high-priority `UnitCount` monotonic counters with their
existing zero defaults: `requests` has one series keyed only by `name`, and `routed.requests` has one
per application route definition and method. Both still count each matched request once after the
metric middleware runs, including requests aborted by downstream middleware. Matched built-in health
routes count; unmatched Gin routes (an empty route template) remain excluded. `/health` still has no
zero default because it is registered outside the route definitions.

The new `responses` and `routed.responses` counters are high-priority `UnitCount` counters with the
full numeric status code and bounded `response.status_class` values `1xx` through `5xx`. They count
each matched response, including responses from aborted downstream requests and matched built-in
health routes; unmatched Gin routes remain excluded. Neither has a zero default. `responses` is keyed
by `name`, status code, and class; `routed.responses` adds `route` and `request.method`. The duration
histogram remains a separate latency measurement with the full status code and no status class.

For CloudWatch only, each matched routed response Counter observation emits exactly one additional
same-name `KindTotal` datum with the same event boundary, value, unit, and priority. The projection
retains `name`, `route`, `request.method`, and the observed `response.status_class`, and sets
`response.status_code` to `metric.DimensionDefault`, which CloudWatch omits. Each observed
name/route/method/class tuple therefore has one active four-dimension class series; no zero default
creates a series for an unobserved class. This is not a separate canonical metric. OpenTelemetry
skips `KindTotal` before instrument lookup and Prometheus before registry registration, so both
continue to export only the raw five-dimension Counter and do not double-count it.

For CloudWatch only, each matched rejected request also emits one same-name `KindTotal` datum for
`http.server.rejected.requests`, dimensioned by exactly `name`. The detailed Counter remains
dimensioned by `name`, `route`, and `request.method` and is unchanged for every exporter. The
server-level projection is `UnitCount`/Sum and has a zero default per server; existing detailed
route/method rejection zero defaults remain. OpenTelemetry and Prometheus skip `KindTotal`, so they
export only the detailed rejection Counter.

CloudWatch consumers must choose one series shape. Metrics Insights queries that read raw response
Counters use the exact five-dimension `FROM SCHEMA(namespace, name, route, request.method,
response.status_code, response.status_class)`, with the class filter applied to those data. A broad
class query can include both the raw Counter and its projection and double-count. A class-only route
total (for example, `response.status_class="5xx"`) uses a standard CloudWatch MetricStat `Sum` with
exactly `name`, `route`, `request.method`, and `response.status_class` dimensions, with no
`response.status_code`. This bounded HTTP projection does not restore the removed topic-only Kafka
`KindTotal` series; the separate shard-reader stream projections are documented above.

`http.client.requests` increments after request and URL validation, immediately before `Execute`,
with `request.method`. `http.client.completed.requests` increments once after `Execute` returns in the
response, transport-error, and cancellation branches. A valid response carries the numeric status
code, its `1xx`–`5xx` class, and `error.type="{{default}}"`; a transport error or cancellation carries
`response.status_code="{{default}}"` and `response.status_class="{{default}}"`, with
`error.type="request_failed"` or `"request_canceled"`, respectively. These counters have no zero
defaults; the duration histogram remains a separate latency instrument.

`rpc.server.requests` increments once at each gRPC `stats.End`, alongside the duration histogram,
for successful and failed calls. It is keyed by `service` and `method` and has no zero defaults.

### Remaining namespaces

| Metric | Unit | Type | Attributes | Owner |
|---|---|---|---|---|
| `blob.batch.operations` | count | counter | `operation` | `pkg/blob` |
| `conc.scheduler.batch.tasks` | count | histogram | `name` | `pkg/conc/scheduler` |
| `conc.scheduler.task.queue.duration` | ms | histogram | `name` | `pkg/conc/scheduler` |
| `limit.attempted.takes` | count | counter | `name`, `prefix` | `pkg/limit` |
| `limit.takes` | count | counter | `name`, `prefix`, `outcome` (`allowed`, `error`), `error.type` | `pkg/limit` |
| `limit.throttled.takes` | count | counter | `name`, `prefix` | `pkg/limit` |
| `smpl.decisions` | count | counter | `sampled` | `pkg/smpl` |
| `metric.log.records` | count | counter | `level` | `pkg/metric` |

## Deliberate exceptions

**Limiter counters separate attempt, terminal result, and throttling.** `limit.attempted.takes` records
one take at `OnTake`; `limit.takes` records only mutually exclusive terminal `allowed` or `error`
outcomes; `limit.throttled.takes` records throttling at `OnThrottle`. All use `name` and `prefix`,
with no zero defaults. Do not sum them as outcomes or count throttled takes through `limit.takes`.
`outcome="throttled"` and the exported `OutcomeThrottled` symbol are removed, with no alias.

**`cloud.aws.kinesis.acquire.duration` is in seconds**, the only duration that is not milliseconds.
Prometheus and OTEL are unaffected — both scale durations to seconds — but a CloudWatch consumer sees
`Seconds`.

**`error.type` names the action that failed, never a Go error type.** A folded metric's failed share
carries a stable, bounded-cardinality marker — `send_failed`, `connect_failed`, `request_failed`,
`request_canceled`, `operation_failed`, `notify_failed`, `apply_failed`, `processing_failed`,
`model_lookup_failed`, `retry_handler_put_failed`, or `rate_limit_error` — not the Go error's type
(which would carry unbounded cardinality) or its message. The successful share usually carries
`{{default}}`. For Prometheus this is an empty label value, for OTEL the attribute is absent, and for
CloudWatch the dimension is omitted. On HTTP client completion counters, missing response code/class
also use the internal `{{default}}` marker. `stream.consumed.messages` and
`cloud.aws.kinesis.consumed.messages` are exceptions: their `{{default}}` series is the total of all
processed records, while `processing_failed` is a per-record subset that overlaps that total. Stream
`model_lookup_failed` diagnostics also overlap the default total; only non-ignored model lookup
failures overlap the `processing_failed` subset. Never sum or subtract those series to derive success.

## Changing a metric

1. Change the emitting package's name and attribute declarations, emission sites, and registered
   defaults as needed.
2. Manually update this inventory and the deliberate exceptions below to reflect the emitted names,
   units, instrument types, and attributes.
3. Register or update the help text with `metric.RegisterHelp` in the emitting package.
4. Run focused behavioral tests for any affected renderer, writer, unit, or emitting path. Existing
   naming and writer tests in `pkg/metric` cover backend rendering; they do not check this inventory.
5. Record the observable change in `RELEASE_NOTES.md`.
6. Removing the last emitter of a metric means removing its inventory entry in the same change, so the
   inventory never claims a metric nothing writes.
