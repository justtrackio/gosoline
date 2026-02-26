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

## Related packages
- `pkg/cloud/aws/sqs`, `sns`, `kinesis` - AWS transport clients
- `pkg/kafka` - Kafka client integration
- `pkg/mdlsub` - model subscription built on stream

## Tips
- Keep message attributes consistent; mdlsub and metric pipelines rely on canonical headers.
- Use context cancellation carefully—consumers/producers run inside kernel modules.
- Document new module factory names in `examples/stream` so users can discover them quickly.
