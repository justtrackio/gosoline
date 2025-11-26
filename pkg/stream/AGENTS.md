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
```yaml
stream.consumer.my-consumer.input: sqs
stream.consumer.my-consumer.sqs.queue_id: my-queue
stream.consumer.my-consumer.encoding: json
stream.consumer.my-consumer.retry.enabled: true
```

## Related packages
- `pkg/cloud/aws/sqs`, `sns`, `kinesis` - AWS transport clients
- `pkg/kafka` - Kafka client integration
- `pkg/mdlsub` - model subscription built on stream

## Tips
- Keep message attributes consistent; mdlsub and metric pipelines rely on canonical headers.
- Use context cancellation carefully—consumers/producers run inside kernel modules.
- Document new module factory names in `examples/stream` so users can discover them quickly.
