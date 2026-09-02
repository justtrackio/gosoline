# Stream Package Agent Guide

## Scope
- Unified streaming abstraction covering consumers, producers, encoders, retry handlers, and health reporting.
- Supports multiple transports (SQS, SNS, Kinesis, Kafka, Redis, files, in-memory) via pluggable inputs/outputs.
- Powers mdlsub, metrics exporters, and application stream modules.

## Key files
- `consumer*.go`, `producer*.go` - base logic and module factories for stream processing.
- `input_*.go`, `output_*.go` - transport-specific adapters.
- `encoding_*.go`, `message*.go` - serialization formats and message helpers.
- `kinsumer_*` - autoscaling components for Kinesis-based consumers.

## Common tasks
- Add new transport: implement matching input/output files following existing patterns, expose settings structs, document config keys.
- Extend encoding: add codec in `encoding_<format>.go`, wire into `EncodingConfig`.
- Tune retry/backoff: update `retry_*.go` and ensure metrics + logging remain accurate.

## Testing
- `go test ./pkg/stream` (covers transports via mocks).
- Transports with external deps may need integration tests under `test/stream` (run with `-tags integration,fixtures`).

## Transport types
| Input | Output | Config prefix |
|-------|--------|---------------|
| SQS | SQS | `stream.input/output.sqs` |
| SNS | SNS | `stream.input/output.sns` |
| Kinesis | Kinesis | `stream.input/output.kinesis` |
| Kafka | Kafka | `stream.input/output.kafka` |
| Redis | Redis | `stream.input/output.redis` |
| File | File | `stream.input/output.file` |
| InMemory | InMemory | (testing) |

## Config keys

Stream inputs and outputs (SQS, SNS, Kinesis, Kafka) use a `cfg.ResourceIdentifier` embedded directly in their
configuration structs. This means the fields are **flat** — no extra nesting level:

| Field | Config key | Required | Description |
|-------|------------|----------|-------------|
| `application` | `application` | no | Name of the owning application. Defaults to `app.name`. |
| `env` | `env` | no | Environment of the owning application. Defaults to `app.env`. |
| `tags` | `tags` | no | Tags for pattern expansion. Merged with `app.tags`; per-resource keys win. |

**Exception:** Redis list inputs and outputs do **not** use `ResourceIdentifier`. They only require `server_name`, `key`,
and transport-specific settings. Redis naming is handled by the Redis client's own naming configuration
(`redis.<client_name>.naming`).

**Kafka/Kinesis inputs** (`KafkaInputConfiguration`, `KinesisInputConfiguration`) embed their transport
`Settings` struct directly (`kafkaConsumer.Settings`, `kinesis.Settings`), which themselves embed
`cfg.ResourceIdentifier`. The config keys are therefore also flat (`application`, `env`, `tags`).

### Output example (SQS)
```yaml
stream:
  output:
    my-output:
      type: sqs
      application: my-app       # optional, defaults to app.name
      tags:                     # optional, merged with app.tags
        project: my-project
        family: my-family
        group: my-group
      queue_id: my-queue
      client_name: default
```

### Input example (SQS)
```yaml
stream:
  input:
    my-input:
      type: sqs
      application: target-app   # optional, defaults to app.name
      tags:                     # optional
        project: my-project
        family: my-family
        group: my-group
      queue_id: my-queue
```

### SNS input with targets
```yaml
stream:
  input:
    my-sns-input:
      type: sns
      id: my-consumer
      grace_time: 10s          # acknowledgement window after processing drains
      acknowledgement_mode: individual # use batch only when delayed deletion is acceptable
      tags:                     # optional — identity of the SQS queue used for fan-out
        project: my-project
        family: my-family
        group: my-group
      targets:
        - application: target-app   # optional — identity of the SNS topic to subscribe to
          tags:
            project: target-project
            family: target-family
            group: target-group
          topic_id: my-topic
```

### Consumer config
```yaml
stream:
  consumer:
    my-consumer:
      input: sqs
      encoding: json
      retry:
        enabled: true
```

Consumer callback concurrency is defined by the input transport. SQS uses
`stream.input.<name>.runner_count` to control the number of receive loops. Kafka uses the same input setting to bound
concurrent processing across assigned partitions. Its default `processing_mode` is `unordered`, so records from the same
topic-partition may be processed concurrently. Set `processing_mode: ordered` to process records sequentially within each
topic-partition while retaining concurrency across partitions. Kinesis uses the same settings to globally bound record
processing across all shards owned by one kinsumer. In `ordered` mode it processes each shard sequentially while retaining
concurrency across shards; in `unordered` mode records from the same shard may be processed concurrently, but checkpoints
still advance in shard order. In-memory inputs use `runner_count` to control the number of concurrent message-processing
callbacks.

### Delayed consumption

Kafka and Kinesis inputs can hold records back until they reached a minimum age via
`stream.input.<name>.consume_delay` (disabled by default). Each record is delayed individually, so a record already
older than the delay is passed on immediately and a backlog is consumed at full speed. The wait is reported as the
`SleepDuration` metric and is included in `ProcessDuration`. Consumer lag sits at roughly `consume_delay` by design, so
lag alerting has to account for it.

The two transports differ in what the age is measured against:

- Kinesis uses the record's approximate arrival timestamp, which is always assigned by the stream.
- Kafka uses the record's timestamp, which the broker only assigns for topics configured with
  `message.timestamp.type=LogAppendTime`. With the default `CreateTime` it comes from the producer, so it is not
  trustworthy: a record dated into the future is delayed by at most `consume_delay`, and a record without a timestamp is
  not delayed at all.

For Kafka the delay also blocks rebalances, because the reader runs with `kgo.BlockRebalanceOnPoll`. Exceeding
`rebalance_timeout` while a rebalance is pending gets the consumer kicked out of its group and causes duplicate
processing, so `consume_delay` is validated to be strictly below `rebalance_timeout` and the config is rejected
otherwise. Raise `rebalance_timeout` if you need a longer delay. Note that a batch costs roughly
`consume_delay + the timestamp spread within the batch` rather than `max_poll_records × consume_delay`, since records age
while their predecessors are waiting.

A shutdown ends an ongoing wait early. Kafka then still processes the record if it is the first of its work unit,
because that record is committed either way (see the offset gap invariant in `processRecordWorkUnits`); Kinesis leaves
the record for redelivery.

### Shutdown draining

There are two kinds of grace time and they belong to different layers:

| Setting | Bounds |
|---------|--------|
| `stream.consumer.<name>.grace_time` | How long a **record** has to be processed. Single authoritative processing deadline for every input, primary and retry alike. |
| `stream.input.<name>.grace_time` | How long the **input** has to commit what was processed: SQS message acknowledgement, Kafka offset commit. Kinesis uses `release_delay` for its equivalent (final checkpoint persist, shard release, client deregistration). |

Once shutdown starts, inputs must stop fetching new messages. SQS and in-memory inputs may still process messages from an
already fetched receive batch or buffer until the consumer's grace deadline expires. Kafka and Kinesis stop admitting
fetched-but-not-yet-callback records immediately; only records already handed to a callback may continue until that
deadline. At that point, in-flight callback contexts are canceled and remaining messages are left for retry, and only
then does the input's own commit window start.

Inputs never run a processing timer of their own. The consumer publishes its drain deadline on the context it passes to
`Input.Run` via `exec.WithDrainContext`, and Kafka and Kinesis keep in-flight handlers alive until it fires. An input
driven by something other than a `stream.Consumer` receives no drain context and propagates cancellation immediately, so
that caller keeps full control over its own shutdown.

Kafka and Kinesis both commit every record that was handed to the callback, even when the callback returned `ack=false`.
Redelivery is owned by the consumer retry queue, so running these inputs with `retry.enabled: false` drops failed
records. Kinesis stops admitting records from an already fetched batch as soon as shutdown starts and advances the shard
checkpoint only through the contiguous handled prefix.

SQS inputs acknowledge each successfully processed message individually by default. Set
`stream.input.<name>.acknowledgement_mode: batch` to issue one SQS batch delete for all successfully processed messages
from a receive result. Batch acknowledgement reduces delete requests, but waits for the complete receive batch to finish
processing before deleting any of its messages.

## Related packages
- `pkg/cloud/aws/sqs`, `sns`, `kinesis` - AWS transport clients
- `pkg/kafka` - Kafka client integration
- `pkg/mdlsub` - model subscription built on stream

## Tips
- Keep message attributes consistent; mdlsub and metric pipelines rely on canonical headers.
- Use context cancellation carefully—consumers/producers run inside kernel modules.
- Document new module factory names in `examples/stream` so users can discover them quickly.
