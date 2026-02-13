# KvStore Package Agent Guide

## Scope
- Key-value storage abstraction with multiple backends (Redis, DynamoDB, In-Memory, Chain).
- Provides `KvStore` interface for basic key-value operations.
- Handles batch operations, configuration, and fixtures.

## Key files
- `kvstore.go` - `KvStore` interface, `Settings`, and factory definitions.
- `configurable.go` - Configuration loading and factory for different backend types.
- `config_postprocessor_redis.go` - Expands `{store}` in key patterns and passes them to the Redis client config.
- `redis.go` - Redis backend implementation.
- `ddb.go` - DynamoDB backend implementation.
- `chain.go` - Chained store implementation (e.g., memory cache in front of Redis).

## Common tasks
- Adding a new backend: implement `KvStore` interface.
- Configuring stores: adjust `kvstore.<name>` settings in `config.dist.yml`.
- Redis Key Naming: configure `kvstore.<name>.redis.key_pattern` or `kvstore.default.redis.key_pattern`.

## Configuration

### Redis Key Naming
Redis key naming can be configured per store or globally using patterns with `cfg.Identity` placeholders.

**Priority:**
1. `kvstore.<name>.redis.key_pattern` (explicit override)
2. `kvstore.default.redis.key_pattern`
3. Default pattern: `{app.namespace}-kvstore-{store}-{key}`

**Supported placeholders:**
- `{app.namespace}` - Resolved namespace (e.g., `{app.tags.project}.{app.env}.{app.tags.family}`)
- `{store}` - The name of the kvstore
- `{key}` - The key being accessed
- `{app.env}` - Environment
- `{app.name}` - Application name
- `{app.tags.<key>}` - Any tag value (e.g., `{app.tags.project}`, `{app.tags.costCenter}`)

**Config examples:**
```yaml
# Global default pattern
kvstore.default.redis.key_pattern: "{app.tags.project}:{app.env}:{store}:{key}"

# Store-specific pattern
kvstore.cache.redis.key_pattern: "cache:{key}"

# Default pattern (uses app.namespace)
kvstore.default.redis.key_pattern: "{app.namespace}-kvstore-{store}-{key}"
```

## Testing
- `go test ./pkg/kvstore` for unit tests.
- Integration tests require Redis and DynamoDB (local or containerized).

## Related packages
- `pkg/redis` - low-level Redis client
- `pkg/ddb` - DynamoDB client and repository
- `pkg/cfg` - configuration and naming template support
