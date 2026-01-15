# HTTP Server Package Agent Guide

## Scope
- Provides Gin-based HTTP server with gosoline middleware, auth, health, CRUD helpers, and profiling.
- Supplies declarative handler definition + validation infrastructure used by examples and services.

## Key files
- `server.go`, `definition.go` - server lifecycle and handler registration structures.
- `middleware_*.go` - logging, metrics, recovery, sampling.
- `handler.go`, `handler_static.go`, `response.go` - base handlers and response helpers.
- `auth/`, `crud/`, `sql/` - optional submodules for auth flows and generic CRUD endpoints.

## Common tasks
- Add middleware: implement `Middleware` in `middleware_<name>.go` and register it via `ServerSettings`.
- Introduce new handler helpers: extend `handler.go` and cover edge cases in `handler_test.go`.
- Update health endpoints: modify `health_check.go` and ensure `examples/httpserver` still passes smoke tests.

## Testing
- `go test ./pkg/httpserver`.
- Manual validation: `cd examples/httpserver/simple-handlers && go run .` then hit `/health`.

## Handler patterns
```go
// Definer function for route registration
type Definer func(ctx context.Context, config cfg.Config, logger log.Logger) (*Definitions, error)

// Example handler registration
func DefineRoutes(ctx context.Context, config cfg.Config, logger log.Logger) (*Definitions, error) {
    d := &Definitions{}
    d.GET("/health", HealthHandler())
    d.POST("/api/v1/items", CreateItemHandler(logger))
    return d, nil
}
```

## Config keys
```yaml
httpserver.default.port: 8088
httpserver.default.mode: release  # or debug
httpserver.default.timeout.read: 60s
httpserver.default.timeout.write: 60s
httpserver.default.compression.level: default
```

## Related packages
- `pkg/http` - HTTP client utilities
- `pkg/validation` - request validation helpers
- `pkg/tracing` - request tracing middleware

## Tips
- Keep handler signatures context-aware (always accept `context.Context`).
- When adding protobuf/JSON helpers, ensure you regenerate `handler_test.proto` outputs.
- Document new configuration knobs under `httpserver.<name>.*` in the global AGENT.
