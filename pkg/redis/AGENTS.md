# Redis Package Agent Guide

## Scope
- Provides Redis client factory, lifecycle hooks, fixtures, and helper exec functions used by cache/stream modules.
- Wraps go-redis with gosoline config, logging, and metrics.

## Key files
- `factory.go`, `client.go` - construct clients from config and expose helper interfaces.
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
