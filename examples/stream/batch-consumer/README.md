# Batch consumer

Run `go run .` from this directory. The file input finishes at EOF and the consumer
flushes its two remaining records as one batch.

Use `application.RunBatchConsumer` / `RunBatchConsumers` for typed callbacks, or
`RunUntypedBatchConsumer` / `RunUntypedBatchConsumers` for mixed models. Module
factories are `stream.NewBatchConsumer`, `NewUntypedBatchConsumer`,
`NewBatchConsumerFactory`, and `NewUntypedBatchConsumerFactory`.

`batch_size` triggers processing when enough records have been collected;
`idle_timeout` triggers partial batches and is reset on each flush. `buffer_size`
defaults to `batch_size` and bounds the admission channel. Processing is serial,
so inputs can keep their default single runner even with large batches.

**Transport acknowledgement happens on admission to memory, before processing.**
Batch results use Gosoline's existing retry handlers (`retry.type: sqs` by
default). The local example disables retries; enable `retry.enabled` with AWS
configured to use SQS. `retry.after` controls the initial delay and redelivery
visibility timeout; `retry.max_attempts` controls SQS redrive to the DLQ.
Retry input callbacks wait for business completion and acknowledge only success.
Retry envelopes flush promptly, so a single retry runner works even with large
batch sizes. There is no separate in-memory retry scheduler or backlog.

Shutdown flushes buffered records within the shared consumer `grace_time`.
Failed primary records are written to the retry queue, including during the
final flush; failed retry records remain unacknowledged for SQS redelivery.
A final flush that cannot retain failed primary records returns an error.
The admission buffer remains volatile, so a crash before retry persistence
requires replay. Successful side effects must be idempotent on redelivery.

Aggregate envelopes are flattened into batch records; failed children retry
individually. Primary envelope acknowledgement is on admission, independent of
`aggregate_message_mode`. Retry envelopes acknowledge only when every child
succeeds; failed envelopes can redeliver successful children too. A single
aggregate can exceed the batch-size threshold.

Processed/error/duration metrics and health checks describe actual batch attempts,
not just admission. Retry counters describe queue writes and retry deliveries.
