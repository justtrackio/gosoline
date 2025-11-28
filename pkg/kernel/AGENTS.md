# Kernel Package Agent Guide

## Scope
- Core runtime orchestrator: stages, modules, middleware, and lifecycle hooks.
- Used by `application` to start/stop modules in dependency order with health signaling.

## Key files
- `kernel.go`, `builder.go` - build and run the kernel with configured stages.
- `module.go`, `module_options.go` - interfaces for modules and factories.
- `stage.go`, `stages.go` - bootstraps ordered stage execution.
- `middleware.go` - cross-cutting hooks invoked before/after modules run.

## Common tasks
- Add stage types: extend `StageConfig`, update builder logic, and document ordering guarantees.
- Introduce middleware: implement the interface in `middleware.go`, wire into `kernel.go`.
- Enhance health reporting: modify `health_check.go` to emit new statuses or metadata.

## Testing
- `go test ./pkg/kernel`—covers builder, middleware, and stage execution.
- Run `go test ./pkg/application` when kernel APIs change to ensure compatibility.

## Stage constants
Modules run in ordered stages (lowest first):
| Stage | Constant | Purpose |
|-------|----------|--------|
| Essential | `StageEssential` | Metrics, telemetry (starts first, stops last) |
| ProducerDaemon | `StageProducerDaemon` | Background message producers |
| Service | `StageService` | Shared services (GeoIP, currency, etc.) |
| Application | `StageApplication` | Your modules (HTTP, consumers, subscribers) |

## Module interface
```go
type Module interface {
    Run(ctx context.Context) error
}

type ModuleFactory func(ctx context.Context, config cfg.Config, logger log.Logger) (Module, error)
```

Optional interfaces: `TypedModule` (essential/background), `StagedModule` (custom stage), `FullModule` (health checks).

## Related packages
- `pkg/application` - wires modules into the kernel
- `pkg/stream` - provides consumer/producer module factories
- `pkg/httpserver` - provides HTTP server module factory

## Tips
- Keep module interfaces backward compatible; changes ripple across most packages.
- Prefer dependency injection via module factories rather than global registries.
- Update docs/examples whenever you introduce new stage names or middleware hooks.
