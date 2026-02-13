# DDB Package Agent Guide

## Scope
- DynamoDB integration layer: metadata modeling, repository abstraction, builders, and fixtures.
- Table naming uses `cfg.Identity.Format()` with ModelId fields (via `ModelId.ToMap()`).
- Provides lifecycle helpers and purgers for test environments.

## Key files
- `metadata*.go` - model descriptors and attribute mapping.
- `builder_*.go` - typed request builders for CRUD, query, scan, transact operations.
- `repository*.go` - high-level repos built on builders/services.
- `naming.go` - table naming rules (uses `cfg.Identity.Format()` with `ModelId.ToMap()`).

## Common tasks
- Extend model metadata: update `metadata_factory.go` and add tests covering new annotations.
- Add new builder functionality: follow existing builder pattern and ensure API remains fluent.
- Adjust throughput/capacity defaults: touch `settings.go` and document config keys.

## Testing
- Run `go test ./pkg/ddb` for all builders/repos.
- For changes affecting naming/macros, also test `pkg/cloud/aws/kinesis` and `pkg/stream` to ensure shared expectations.

## Naming with ModelId
Table names are generated via `cfg.Identity.Format()` using the ModelId fields. Default pattern:
```yaml
cloud.aws.dynamodb.clients.default.naming.table_pattern: "{app.namespace}-{name}"
```

**Note:** DynamoDB table names are built via `cfg.Identity.Format()` with `ModelId.ToMap()`. The placeholders are:

| Macro | Description |
|-------|-------------|
| `{app.env}` | Environment |
| `{app.name}` | Application name |
| `{app.namespace}` | Pre-configured namespace pattern |
| `{app.tags.<key>}` | Any tag value (e.g., project, family, group) |
| `{name}` | Model name (from ModelId.Name) |

## Common config keys
```yaml
cloud.aws.dynamodb.clients.default.endpoint: http://localhost:4566
cloud.aws.dynamodb.clients.default.naming.table_pattern: "{app.namespace}-{name}"
```

## Related packages
- `pkg/mdl` - ModelId definition and macro helpers
- `pkg/cloud/aws/dynamodb` - low-level AWS client
- `pkg/fixtures` - DDB fixture writers

## Tips
- Keep request builders composable—avoid hard-coding table names; always take `Metadata` or `ModelId` input.
- Fixture writers (`fixture_writer_ddb*.go`) must stay in sync with metadata parsing.
- Update `.mockery.yml` when adding new interfaces so DynamoDB service mocks stay current.
- DynamoDB naming uses `cfg.Identity.Format()` but with ModelId fields for model-specific identifiers.
