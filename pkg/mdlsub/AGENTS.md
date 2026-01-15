# Model Subscription Package Agent Guide

## Scope
- Handles publishing/subscribing flows for data model changes across DB, DDB, KVStore, and stream outputs.
- Provides config postprocessors, fixtures, and transformer helpers for mdl change pipelines.

## Key files
- `publisher*.go`, `output_*.go` - publish model changes to DB, DDB, KVStore, or stream layers.
- `subscriber_*.go` - consumer side logic (callbacks, factories, settings).
- `fixtures.go` - fixture wiring for integration suites.

## Common tasks
- Add a new output target: extend `output_<target>.go`, update settings, ensure tests cover retries/backoff.
- Modify subscriber pipeline: update `subscriber_core.go` and keep `subscriber_factory.go` in sync.
- Adjust config postprocessors so `application.yml` keys remain ergonomic.

## Testing
- `go test ./pkg/mdlsub`.
- Integration coverage lives under `test/mdlsub`; run with `go test -tags integration,fixtures ./test/mdlsub/...` when touching IO-heavy code.

## Publisher/subscriber flow
```
[DB/DDB change] → [Publisher] → [Stream output] → [Subscriber] → [Output target]
```

Output targets: `db`, `ddb`, `kvstore`

## Config keys
```yaml
mdlsub.publishers.mymodel.output.type: sns
mdlsub.publishers.mymodel.output.topic_id: model-changes

mdlsub.subscribers.mymodel.input.type: sqs
mdlsub.subscribers.mymodel.input.queue_id: model-changes
mdlsub.subscribers.mymodel.output.type: ddb
```

## Related packages
- `pkg/stream` - underlying transport layer
- `pkg/ddb`, `pkg/db-repo`, `pkg/kvstore` - output targets
- `pkg/mdl` - ModelId for naming

## Tips
- Rely on `mdl.ModelId` and `cfg.Identity` for naming; never duplicate logic locally.
- Keep transformers stateless and idempotent to simplify retries.
- Update fixture helpers when adding outputs so integration suites can preload state.
