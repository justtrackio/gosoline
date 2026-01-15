# Log Package Agent Guide

## Scope
- Structured logging facade with field enrichers, Sentry integration, sampling, and context propagation.
- Consumed by almost every package via dependency injection.

## Key files
- `logger.go`, `options.go` - logger creation and configuration.
- `handler_*.go`, `formatter.go` - sinks (stdout, file, Sentry) and formatting rules.
- `context.go`, `stacktrace.go` - context helpers and panic capture.
- `config_postprocessor_main_logger.go` - ties logger config into global app settings.

## Common tasks
- Add handler or formatter: create new handler file, register via `options.go`, add tests.
- Extend sampling: adjust `logger_sampling.go` and ensure default config still keeps high-volume services safe.
- Update ECS/Sentry metadata enrichment: edit `ecs_metadata.go` or `sentry.go` modules.

## Testing
- `go test ./pkg/log`.
- For Sentry changes, run `go test ./pkg/log -run TestSentry` and, if possible, hit a dev DSN manually.

## Log levels
| Level | Priority | Use case |
|-------|----------|----------|
| `trace` | 0 | Verbose debugging |
| `debug` | 1 | Development info |
| `info` | 2 | Normal operations |
| `warn` | 3 | Recoverable issues |
| `error` | 4 | Failures requiring attention |
| `none` | max | Disable logging |

## Built-in handlers
| Handler | File | Purpose |
|---------|------|--------|
| IOWriter | `handler_iowriter.go` | Stdout/file output |
| Sentry | `handler_sentry.go` | Error reporting |

## Config keys
```yaml
log.level: info                  # global default level
log.handlers.main.type: iowriter # auto-configured by postprocessor if not set
log.handlers.main.level: info
log.handlers.main.channels: ["*"]
log.handlers.sentry.type: sentry # optional
log.handlers.sentry.dsn: ""
```

## Related packages
- `pkg/tracing` - distributed tracing integration
- `pkg/metric` - metrics emission alongside logging

## Tips
- Avoid global loggers; expose factories via DI modules.
- Document new config options under `log.handlers.<name>.*` and keep defaults safe for production.
- When adding new handler dependencies, update root `go.mod` carefully to avoid bloat.
