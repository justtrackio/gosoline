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

## PublisherSettings

`PublisherSettings` embeds `mdl.ModelId` **directly** (flat embedding). All `ModelId` fields (`Name`, `Application`,
`Env`, `Tags`, `DomainPattern`) are therefore top-level config keys under `mdlsub.publishers.<name>.*`.

`Name` defaults to the publisher map key when not explicitly set. `Application`, `Env`, and `Tags` are padded from
global app config via `ModelId.PadFromConfig`.

## Config keys
```yaml
mdlsub:
  publishers:
    mymodel:
      output_type: sns          # or sqs / kinesis / kafka / in_memory
      # ModelId fields (all optional — defaults come from app config):
      # name: mymodel           # defaults to the publisher map key
      # application: my-app    # defaults to app.name
      # env: prod               # defaults to app.env
      # tags:
      #   project: my-project

  subscribers:
    mymodel:
      input:
        type: sqs
        queue_id: model-changes
      output:
        type: ddb
```

## Related packages
- `pkg/stream` - underlying transport layer
- `pkg/ddb`, `pkg/db-repo`, `pkg/kvstore` - output targets
- `pkg/mdl` - ModelId for naming and canonical string representation

## Tips
- `PublisherSettings` embeds `mdl.ModelId` — access model fields directly (`settings.Name`, `settings.Application`), not via `settings.ModelId.*`.
- Rely on `mdl.ModelId` for naming; the config postprocessor wires transport identity from ModelId fields automatically.
- Keep transformers stateless and idempotent to simplify retries.
- Update fixture helpers when adding outputs so integration suites can preload state.
