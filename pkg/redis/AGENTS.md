# Redis Package Agent Guide

## Scope
- Provides Redis client factory, lifecycle hooks, fixtures, and helper exec functions used by cache/stream modules.
- Wraps go-redis with gosoline config, logging, and metrics.
- Address naming uses `cfg.NamingTemplate` with AppIdentity macros.

## Key files
- `factory.go`, `client.go` - construct clients from config and expose helper interfaces.
- `dialer.go` - resolves Redis addresses using naming patterns.
- `lifecycle.go`, `lifecycle_purger.go` - ensure clean startup/shutdown per module/test.
- `fixture_writer_redis.go` - integrates fixtures package for deterministic test data.

## Common tasks
- Add client options: update `ClientSettings` in `factory.go`, propagate to `client.go`, and document config keys.
- Modify lifecycle purging: adjust `lifecycle_purger.go` and ensure tests cover multi-db flushing.
- Instrument metrics/logging: update `exec.go` wrappers so new commands emit telemetry.

## Testing
- `go test ./pkg/redis`.
- For fixture or lifecycle changes, also run `go test ./pkg/stream ./pkg/fixtures` to catch regressions.

## Config keys
```yaml
redis.default.address: localhost:6379
redis.default.mode: standalone  # or cluster
redis.default.database: 0
redis.default.dialer.timeout: 5s
redis.default.dialer.read_timeout: 3s
redis.default.dialer.write_timeout: 3s
redis.default.naming.address_pattern: "{name}.{app.tags.group}.redis.{app.env}.{app.tags.family}"
redis.default.naming.key_pattern: "{app.tags.project}-{app.env}-{app.tags.family}-{app.tags.group}-{app.name}-{key}"
```

## Address naming pattern
Redis address naming uses `cfg.NamingTemplate` with AppIdentity macros:
```yaml
redis.default.naming.address_pattern: "{name}.{app.tags.group}.redis.{app.env}.{app.tags.family}"
```

| Placeholder | Description |
|-------------|-------------|
| `{name}` | Redis client name (resource-specific) |
| `{app.env}` | Environment |
| `{app.tags.family}` | Family tag |
| `{app.tags.group}` | Group tag |

## Key naming pattern
Redis key naming uses `cfg.NamingTemplate` with AppIdentity macros to build fully-qualified keys. This allows flexible key namespacing across environments and applications.

### Configuration
```yaml
redis.default.naming.key_pattern: "{app.tags.project}-{app.env}-{app.tags.family}-{app.tags.group}-{app.name}-{key}"
```

### Supported placeholders
| Placeholder | Description |
|-------------|-------------|
| `{key}` | Application-specific key (required in all patterns) |
| `{app.env}` | Environment |
| `{app.name}` | Application name |
| `{app.tags.*}` | Any dynamic tag (e.g., `{app.tags.project}`, `{app.tags.team}`) |

### Pattern validation rules
1. The `{key}` placeholder **must** be present in all patterns
2. Only registered placeholders are allowed (unknown placeholders return an error)
3. Tags are only required if the pattern references them
4. Typos like `{app.tag.project}` (missing 's') will be caught at runtime

### Example patterns
```yaml
# Default pattern (backward-compatible)
redis.default.naming.key_pattern: "{app.tags.project}-{app.env}-{app.tags.family}-{app.tags.group}-{app.name}-{key}"

# Minimal pattern (no tags required)
redis.default.naming.key_pattern: "{app.env}-{app.name}-{key}"

# Custom tags
redis.default.naming.key_pattern: "{app.tags.region}-{app.tags.team}-{app.env}-{key}"
```

### Usage in code
The `BuildFullyQualifiedKey` function expands patterns:

```go
fullyQualifiedKey, err := redis.BuildFullyQualifiedKey(config, appIdentity, "my-cache-key")
if err != nil {
    return fmt.Errorf("can not build fully qualified key: %w", err)
}
// Result: "myproject-prod-myfamily-mygroup-myapp-my-cache-key"
```

## Client modes
| Mode | Use case |
|------|----------|
| `standalone` | Single Redis instance |
| `cluster` | Redis Cluster |

## Related packages
- `pkg/kvstore` - key-value abstraction (can use Redis backend)
- `pkg/stream` - Redis list input/output for messaging
- `pkg/cache` - caching layer (can use Redis)

## Tips
- Keep default timeouts conservative; production workloads rely on these settings.
- When adding new interfaces, update `.mockery.yml` and regenerate mocks before committing.
- Document any assumptions about Redis version/features (cluster vs standalone) in this file.
