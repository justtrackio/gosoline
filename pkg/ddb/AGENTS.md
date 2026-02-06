# DDB Package Agent Guide

## Scope
- DynamoDB integration layer: metadata modeling, repository abstraction, builders, and fixtures.
- Table naming uses `mdl.ModelId` with its own macro system (NOT `cfg.NamingTemplate`).
- Provides lifecycle helpers and purgers for test environments.

## Key files
- `metadata*.go` - model descriptors and attribute mapping.
- `builder_*.go` - typed request builders for CRUD, query, scan, transact operations.
- `repository*.go` - high-level repos built on builders/services.
- `naming.go` - table naming rules (relies on `ModelId.ReplaceMacros`).

## Common tasks
- Extend model metadata: update `metadata_factory.go` and add tests covering new annotations.
- Add new builder functionality: follow existing builder pattern and ensure API remains fluent.
- Adjust throughput/capacity defaults: touch `settings.go` and document config keys.

## Testing
- Run `go test ./pkg/ddb` for all builders/repos.
- For changes affecting naming/macros, also test `pkg/cloud/aws/kinesis` and `pkg/stream` to ensure shared expectations.

## Naming with ModelId
Table names are generated via `config.FormatString` using the ModelId (converted to a map). Default pattern:
```yaml
ddb.default.naming.pattern: "{app.namespace}-{name}"
```

**Note:** DynamoDB uses `AppIdentity`-style macros. The placeholders are:

| Macro | Description |
|-------|-------------|
| `{app.tags.project}` | Project tag |
| `{app.env}` | Environment |
| `{app.tags.family}` | Family tag |
| `{app.tags.group}` | Group tag |
| `{app.name}` | App name |
| `{name}` | Model name (from metadata) |

## Common config keys
```yaml
cloud.aws.dynamodb.clients.default.endpoint: http://localhost:4566
ddb.default.naming.pattern: "{app.namespace}-{name}"
```

## Related packages
- `pkg/mdl` - ModelId definition and macro helpers
- `pkg/cloud/aws/dynamodb` - low-level AWS client
- `pkg/fixtures` - DDB fixture writers

## Tips
- Keep request builders composable—avoid hard-coding table names; always take `Metadata` or `ModelId` input.
- Fixture writers (`fixture_writer_ddb*.go`) must stay in sync with metadata parsing.
- Update `.mockery.yml` when adding new interfaces so DynamoDB service mocks stay current.
- DynamoDB naming is intentionally separate from `cfg.NamingTemplate` to support model-specific identifiers.
