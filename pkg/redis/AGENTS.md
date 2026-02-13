# Redis Package Agent Guide

## Scope
- Provides Redis client factory, lifecycle hooks, fixtures, and helper exec functions used by cache/stream modules.
- Wraps go-redis with gosoline config, logging, and metrics.
- Address naming uses `cfg.Identity.Format()` with pattern-based macros.

## Key files
- `settings.go` - client settings, naming patterns, and address resolution via `cfg.Identity.Format()`.
- `client.go`, `client_pipeliner.go` - construct clients from config and expose helper interfaces.
- `dialer.go` - handles Redis connection dialing (SRV lookup and TCP).
- `lifecycle.go`, `lifecycle_purger.go` - ensure clean startup/shutdown per module/test.
- `fixture_writer_redis.go` - integrates fixtures package for deterministic test data.
- `exec.go` - command wrappers with metrics and logging instrumentation.

## Common tasks
- Add client options: update `Settings` in `settings.go`, propagate to `client.go`, and document config keys.
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
redis.default.naming.key_pattern: "{key}"
```

## Address naming pattern
Redis address naming uses `cfg.Identity.Format()` with pattern-based macros (resolved in `settings.go`):
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
Redis key naming uses `cfg.Identity.Format()` with pattern-based macros to build a key prefix. The pattern is processed in `client.go` by stripping the `{key}` placeholder to extract a prefix, which is then expanded via `Format()` and prepended to all keys.

### Configuration
```yaml
redis.default.naming.key_pattern: "{key}"
```

The default pattern is just `{key}` (no prefix). A common configuration for namespaced keys:
```yaml
redis.default.naming.key_pattern: "{app.namespace}-{app.name}-{key}"
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
# Default pattern (no prefix)
redis.default.naming.key_pattern: "{key}"

# Namespaced pattern (backward-compatible with legacy naming)
redis.default.naming.key_pattern: "{app.namespace}-{app.name}-{key}"

# Minimal pattern (no tags required)
redis.default.naming.key_pattern: "{app.env}-{app.name}-{key}"

# Custom tags
redis.default.naming.key_pattern: "{app.tags.region}-{app.tags.team}-{app.env}-{key}"
```

### How it works in code
The key pattern is processed in `client.go` during client initialization. The `{key}` placeholder is stripped to extract a prefix pattern, which is expanded via `Identity.Format()`:

```go
// The prefix is derived from the key pattern by removing the {key} suffix.
// For example, pattern "{app.namespace}-{app.name}-{key}" produces prefix "{app.namespace}-{app.name}-".
// All Redis operations then prepend this expanded prefix to the application key.
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
