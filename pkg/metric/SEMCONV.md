# Gosoline metric semantic conventions

Gosoline follows the OpenTelemetry semantic conventions for **grammar, units and attribute shape**, and
deliberately does **not** use their names. Nothing gosoline exports is a canonical semantic-convention
metric: the OTEL renderer prefixes every name with `gosoline.`, so a backend can never mistake a
gosoline metric for the convention's own.

This file is the specification. **Any change to a metric name, attribute key, unit or instrument type
updates this file in the same commit.** The conformance test in `conformance_test.go` enforces the
inventory; it cannot enforce this document, so keeping it current is on whoever changes a metric.

## Why the names diverge

A canonical semantic-convention name is a contract about exactly what is measured, with which
attributes, at which point. Prefixing or extending one produces a *different* metric that tooling will
not recognise as the convention's, so a half-adopted convention is worse than none: a dashboard built on
`http.server.request.duration` would silently read a metric whose attribute set gosoline chose.

Emitting under gosoline's own names makes the divergence explicit and lets the grammar stay uniform
across every subsystem, which is what actually makes the metrics learnable.

## The grammar

The rules live in `AGENTS.md` under "Naming grammar" and are summarised here:

1. The namespace names the owner; never repeat it in the leaf.
2. Event counter: `[qualifier.]<plural-noun>`, qualifier a past participle.
3. Current amount: `<thing>.count`, on a gauge.
4. Elapsed time: `<operation>.duration`, on a histogram, in milliseconds.
5. Bytes: `<thing>.size`. Items per batch: `<thing>.<plural-noun>`.
6. A failure is not its own metric: `error.type` on the operation metric, `{{default}}` on success.
7. Components lowercase, words joined by `_`, hierarchy by `.`.
8. An attribute key never repeats its metric's namespace.

## Attributes

`error.type` is the **only** key gosoline takes verbatim from a semantic convention, because its values
are what make an operation metric self-describing. Every other key is gosoline's own and carries no
namespace prefix.

| Key | Meaning | Semconv equivalent |
|---|---|---|
| `error.type` | what failed; `{{default}}` when nothing did | `error.type`, used verbatim |
| `broker.address` | Kafka broker the client talked to | none (`server.address` is close) |
| `client.name` | Kafka client name | none |
| `client.type` | `consumer` or `producer` | none |
| `connection.state` | `used` or `idle` | derived from `db.client.connection.state` |
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
| `route` | HTTP route template | derived from `http.route` |
| `sampled` | whether the sampler kept the item | none |
| `service` | gRPC service | derived from `rpc.service` |
| `store.type` | key-value store implementation | none |
| `stream.name` | Kinesis stream | derived from `messaging.destination.name` |
| `topic.name` | Kafka topic | derived from `messaging.destination.name` |

"Derived from" means the value carries the same meaning as that convention attribute with the namespace
prefix stripped. It is **not** that attribute, and tooling keyed on the convention will not find it.

## The inventory

62 metrics across 18 namespaces. `unit` is the authored unit; every writer renders it into its own
convention (Prometheus scales durations to seconds and appends `_seconds`, OTEL sets the UCUM unit on
the instrument).

### `stream` — `pkg/stream`

| Metric | Unit | Type | Attributes | Derived from |
|---|---|---|---|---|
| `process.duration` | ms | histogram | `consumer.name` | `messaging.process.duration` |
| `consumed.messages` | count | counter | `consumer.name`, `error.type` | `messaging.client.consumed.messages` |
| `retry.operations` | count | counter | `consumer.name`, `operation` | none |
| `produced.messages` | count | counter | `producer.name` | none |
| `batch.messages` | count | histogram | `producer.name` | none |
| `aggregate.messages` | count | histogram | `producer.name` | none |
| `idle.duration` | ms | histogram | `producer.name` | none |
| `message.count` | count | gauge | `list.name` | none |
| `reads` | count | counter | `list.name` | none |
| `writes` | count | counter | `list.name` | none |

### `kafka.consumer` — `pkg/kafka/consumer`

| Metric | Unit | Type | Attributes | Derived from |
|---|---|---|---|---|
| `consumed.messages` | count | counter | `client.type`, `client.name`, `topic.name`, `partition.id`, `error.type` | `messaging.client.consumed.messages` |
| `process.duration` | ms | histogram | `client.type`, `client.name`, `topic.name`, `partition.id` | `messaging.process.duration` |
| `polls` | count | counter | `client.type`, `client.name`, `topic.name` | none |
| `poll.duration` | ms | histogram | `client.type`, `client.name`, `topic.name` | none |
| `commit.duration` | ms | histogram | + `partition.id`, `error.type` | none |
| `wait.duration` | ms | histogram | + `partition.id` | none |
| `rebalances` | count | counter | `client.type`, `client.name`, `topic.name` | none |

### `kafka.producer` — `pkg/kafka/producer`

| Metric | Unit | Type | Attributes | Derived from |
|---|---|---|---|---|
| `sent.messages` | count | counter | `client.type`, `client.name`, `topic.name`, `error.type` | `messaging.client.sent.messages` |
| `produce.duration` | ms | histogram | `client.type`, `client.name`, `topic.name` | `messaging.client.operation.duration` |
| `batch.records` | count | histogram | `client.type`, `client.name`, `topic.name` | none |

### `kafka` — `pkg/kafka`

Broker-level metrics from the franz-go hooks. All carry `client.type`, `client.name`, `broker.address`.

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

### `cloud.aws.kinesis` — `pkg/cloud/aws/kinesis`

| Metric | Unit | Type | Attributes | Derived from |
|---|---|---|---|---|
| `consumed.messages` | count | counter | `stream.name`, `partition.id`, `error.type` | `messaging.client.consumed.messages` |
| `sent.messages` | count | counter | `stream.name`, `error.type` | `messaging.client.sent.messages` |
| `process.duration` | ms | histogram | `stream.name`, `partition.id` | `messaging.process.duration` |
| `reads` | count | counter | `stream.name`, `partition.id` | none |
| `lag` | ms | gauge | `stream.name`, `partition.id` | `messaging.kafka.consumer.group.lag` in spirit |
| `acquire.duration` | s | histogram | `stream.name`, `partition.id` | none |
| `sleep.duration` | ms | histogram | `stream.name`, `partition.id` | none |
| `wait.duration` | ms | histogram | `stream.name`, `partition.id` | none |
| `shard.count` | count | gauge | `stream.name` | none |
| `client.count` | count | gauge | `stream.name` | none |
| `batch.records` | count | histogram | `stream.name` | none |

`sent.messages` carries the reason Kinesis gave as its `error.type`, not a Go type, because the API
reports a per-record error code rather than an error value.

### `kvstore` — `pkg/kvstore`

| Metric | Unit | Type | Attributes |
|---|---|---|---|
| `reads` | count | counter | `model.id`, `store.type`, `hit` |
| `writes` | count | counter | `model.id`, `store.type` |
| `deletes` | count | counter | `model.id`, `store.type` |
| `item.count` | count | gauge | `model.id`, `store.type` |

### `db.client`, `db.repo`, `ddb` — `pkg/db`, `pkg/db-repo`, `pkg/ddb`

| Metric | Unit | Type | Attributes | Derived from |
|---|---|---|---|---|
| `db.client.connection.count` | count | gauge | `connection.state` | `db.client.connection.count` |
| `db.client.connections` | count | counter | `connection.state` | none |
| `db.repo.operation.duration` | ms | histogram | `operation.name`, `model.id`, `error.type` | `db.client.operation.duration` |
| `db.repo.model_event.notifications` | count | counter | `model.id`, `error.type` | none |
| `ddb.operation.duration` | ms | histogram | `operation.name`, `model.id`, `error.type` | `db.client.operation.duration` |

### `mdlsub` — `pkg/mdlsub`

| Metric | Unit | Type | Attributes |
|---|---|---|---|
| `events` | count | counter | `model.id`, `outcome` (`applied`, `skipped`), `error.type` |

### `http.server`, `http.client`, `rpc.server` — `pkg/httpserver`, `pkg/http`, `pkg/grpcserver`

| Metric | Unit | Type | Attributes | Derived from |
|---|---|---|---|---|
| `http.server.request.duration` | ms | histogram | `name`, `route`, `request.method`, `response.status_code` | `http.server.request.duration` |
| `http.server.rejected.requests` | count | counter | `name`, `route`, `request.method` | none |
| `http.server.active_request.count` | count | gauge | `name` | `http.server.active_requests` |
| `http.server.connection.count` | count | gauge | `name` | none |
| `http.client.request.duration` | ms | histogram | `request.method`, `response.status_code`, `error.type` | `http.client.request.duration` |
| `rpc.server.request.duration` | ms | histogram | `service`, `method` | `rpc.server.duration` |

### Remaining namespaces

| Metric | Unit | Type | Attributes | Owner |
|---|---|---|---|---|
| `blob.batch.operations` | count | counter | `operation` | `pkg/blob` |
| `conc.scheduler.batch.tasks` | count | histogram | `name` | `pkg/conc/scheduler` |
| `conc.scheduler.task.queue.duration` | ms | histogram | `name` | `pkg/conc/scheduler` |
| `limit.takes` | count | counter | `name`, `prefix`, `outcome` (`allowed`, `throttled`, `error`), `error.type` | `pkg/limit` |
| `smpl.decisions` | count | counter | `sampled` | `pkg/smpl` |
| `metric.log.records` | count | counter | `level` | `pkg/metric` |

## Deliberate exceptions

**`limit.takes` is recorded when a take ends, not when it starts.** The middleware is told `OnTake`
before the outcome is known, so counting there could not carry `outcome`. Attempts are the sum over
`outcome`; there is no separate attempt counter.

**`cloud.aws.kinesis.acquire.duration` is in seconds**, the only duration that is not milliseconds.
Prometheus and OTEL are unaffected — both scale durations to seconds — but a CloudWatch consumer sees
`Seconds`.

## Changing a metric

1. Change the emitting package: the name constant, the attribute keys, the emission sites and the
   registered defaults.
2. Update `authoredNames` and, if a key is new, `authoredDimensionKeys` in `conformance_test.go`, plus
   the expected OTEL unit map.
3. Register or update the help text with `metric.RegisterHelp` in the emitting package.
4. **Update this file** — the inventory table, and the exceptions section if the change adds or
   resolves one.
5. Record the observable change in `RELEASE_NOTES.md`.
6. Removing the last emitter of a metric means removing its inventory entry in the same change, so the
   inventory never claims a metric nothing writes.
