# Metric Package Agent Guide

## Scope
- Collects metric data through the metric channel and flushes it via the metric daemon.
- Owns the metric emission contract: metric name formatting, dimension keys, unit representation.
- Hosts the writers exporting that contract: `cloudwatch`, `prometheus`, `otel` in-tree, plus the
  `elasticsearch` writer type whose factory is registered from outside this package.

## Key files
- `daemon.go` - `NewDaemonModule` factory, `RegisterWriterFactory`, aggregation and flush loop.
- `channel.go` - buffered channel every `Write`/`WriteOne` call feeds into.
- `writer_*.go` - backend writers selected through `metric.writers`.
- `settings.go` - `Settings` struct read from the `metric` config key.
- `schema_version.go` - metric schema version constant, format validation, metadata registration.

## Metric schema version
The metric schema version identifies the metric emission contract a gosoline build implements, so
tooling (dashboard generators, alert provisioning, metric pipelines) can branch on it without
inspecting the gosoline version.

- Current value: `v1.0`, defined by the exported constant `metric.SchemaVersion` in
  `schema_version.go`. That constant is the single source of truth - no other package may define
  the literal value.
- Metadata key: `metric.schema_version` (`metric.MetadataKeySchemaVersion`). The value is written
  into the `appctx.Metadata` carrier and therefore served by the metadata server's root route.
- Format: `v<MAJOR>.<MINOR>`, both components decimal integers of 1 to 9 digits without leading
  zeros unless the component is exactly `0`. `metric.IsValidSchemaVersion` enforces it.

### Increment rules
- **MAJOR**: a metric name, dimension key, or unit representation is removed or renamed. Increment
  MAJOR by 1 and reset MINOR to 0 - this also applies when the same change adds something.
- **MINOR**: the change is purely additive (new metric name, dimension key, or unit
  representation, nothing removed or renamed). Increment MINOR by 1, leave MAJOR unchanged.
- **unchanged**: every metric name, dimension key, and unit representation stays as it is. Leave
  the version untouched, including for refactorings and performance work.

### Release process
1. Bump `metric.SchemaVersion` in `schema_version.go` in the same commit as the contract change,
   following the increment rules above.
2. Update the current value in this file so the documented value never drifts from the constant.
3. Describe the contract change and the new version in `RELEASE_NOTES.md`.

### Publication requirements
The version is published only by an **enabled metric daemon**: the application must wire
`application.WithMetrics` and be configured with `metric.enabled: true`. `NewDaemonModule`
registers the version right after the enabled check, during kernel build, so the entry is present
before the metadata server serves its first response. With `metric.enabled: false`, or without
`application.WithMetrics`, the metadata document contains no `metric` member at all - the absence
is the intended signal that the application emits no metrics.

## Common tasks
- Add a writer: implement `Writer` and register it through `RegisterWriterFactory`, then document
  its settings key below `metric.writer_settings`.
- Change the emitted contract: adjust the emitting code **and** apply the increment rules above.
- Adjust default metrics: see `defaults.go` and the per-package `metric.Datum` producers.

## Testing
- `go test ./pkg/metric/...` before pushing changes.

## Required config keys
```yaml
metric:
  enabled: true               # Enables the metric daemon; required for schema version publication
  interval: 60s               # Flush interval
  writers:                    # Backends; any subset of the supported writers
    - cloudwatch
```

## Related packages
- `pkg/appctx` - metadata carrier the schema version is registered in
- `pkg/application` - `WithMetrics` wiring and the metadata server exposing the version
- `pkg/kernel` - module lifecycle; the daemon factory runs during kernel build
- `pkg/otel` - shared OTLP exporter configuration used by the `otel` writer

## Tips
- Never derive the reported schema version from configuration - report the constant verbatim.
- Registration performs exactly one attempt and returns a wrapped error; `NewDaemonModule` returns
  that error unchanged, so the kernel build aborts instead of shipping an incomplete metadata
  document.
