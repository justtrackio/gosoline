# DB Package Agent Guide

## Scope
- SQL connectivity layer: connection pooling, DSN generation, migrations, and fixtures.
- Supports MySQL/MariaDB, Redshift, TiDB, CrateDB, and custom drivers via `driver_factory.go`.
- Provides lifecycle management hooks for gosoline modules.

## Key files
- `connection.go`, `client.go` - central connection manager + lifecycle integration.
- `driver_*.go` - dialect-specific configuration and registrations.
- `migrations_*.go` - pluggable migration runners (goose, golang-migrate).
- `fixture_*` + `data_*` - seeding/import/export helpers used by tests and CLI tools.

## Common tasks
- Add driver support: implement `Driver` in a new `driver_<name>.go`, register it in `driver_factory.go`, document config keys.
- Extend migrations: update `migrations.go` and the helper specific to your engine.
- Update metrics/logging: `metrics.go` wires health counters; keep names consistent with `metric` package.

## Testing
- `go test ./pkg/db` for unit coverage.
- For driver additions, run targeted tests (e.g., `go test ./pkg/db -run TestMysql...`). Integration tests may need Docker DB instances.

## Supported drivers
| Driver | File | Notes |
|--------|------|-------|
| MySQL/MariaDB | `driver_mysql.go` | Default, most common |
| Redshift | `driver_redshift.go` | AWS data warehouse |
| CrateDB | `driver_cratedb.go` | Distributed SQL |
| TiDB | `tidb/` | TiDB specific error checkers for retries |

## Common config keys
```yaml
db.default.driver: mysql
db.default.hostname: localhost
db.default.port: 3306
db.default.database: mydb
db.default.username: root
db.default.password: ""
db.default.migrations.enabled: true
db.default.migrations.path: migrations
```

## Related packages
- `pkg/db-repo` - repository pattern built on this package
- `pkg/fixtures` - fixture writers for test data seeding
- `pkg/dbx` - sqlx-based query helpers

## Tips
- All config structs must use `cfg:"db.<client>.<key>"` naming to stay consistent with docs.
- Always add fixture-writer support if the new driver will be used in integration tests.
- When touching shared interfaces, regenerate mocks via `go generate -run='mockery' ./pkg/db`.
