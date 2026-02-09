# Fixtures Package Agent Guide

## Scope
- Centralized fixture loading framework used by tests, integration suites, and examples.
- Supports typed sequences, auto-numbering, UUIDs, and custom providers.
- Coordinates with storage packages (`db`, `ddb`, `redis`, `blob`, etc.).

## Key files
- `loader.go`, `container.go` - orchestrate fixture set registration and execution.
- `settings.go`, `fixture_set_options.go` - config surface for enabling/disabling fixtures per environment.
- `provider/` - generic provider infrastructure (kernel module, handler interface). Specific fixture writers (DB, DDB, Redis, Blob) live in their respective packages.
- Sequence helpers (`auto_numbered.go`, `uuid_sequence.go`, etc.) for deterministic IDs.

## Common tasks
- Add a provider: implement `Provider` interface in `provider/`, expose options, and document required config keys.
- Customize fixture execution order: extend `FixtureSetOptions` and adjust `loader.go` accordingly.
- Introduce new sequence strategy: add a helper type plus tests covering wrap-around/overflow.

## Testing
- `go test ./pkg/fixtures` after any change.
- When adding a provider, also run the consuming package’s tests (e.g., `pkg/db`, `pkg/ddb`).

## Build tag requirement
All fixture code requires the `fixtures` build tag:
```go
//go:build fixtures

package fixtures
```

Compile/test with: `go test -tags fixtures ./...`

## Config keys
```yaml
fixtures.enabled: true
fixtures.groups:
  - default
  - integration
```

## Built-in providers
| Provider | Package | Purpose |
|----------|---------|--------|
| MySQL ORM | `pkg/db-repo` | GORM-based seeding |
| MySQL SQLX | `pkg/db` | Raw SQL seeding |
| DynamoDB | `pkg/ddb` | DDB item seeding |
| Redis | `pkg/redis` | Key/value seeding |
| Blob | `pkg/blob` | S3/file seeding |

## Tips
- Keep fixtures idempotent—providers should detect existing state when possible.
- Respect `fixtures.enabled` config toggle so CI can skip expensive setup.
- Document new fixture types/examples under `examples/fixtures` if they require external services.
