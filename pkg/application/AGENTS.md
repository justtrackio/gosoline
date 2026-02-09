# Application Package Agent Guide

## Scope
- Bootstraps gosoline applications via `application.Run`.
- Wires modules, runners, and metadata server exposure.
- Bridges configuration (`cfg`) and lifecycle management (`kernel`).

## Key files
- `app.go` - core application struct, `Default()` and `New()` factory functions.
- `options.go` - functional options for adding modules, health checks, and shared components.
- `runners.go` - `Run()` entrypoint and helpers for wiring background runners/modules.
- `metadata_server.go` - HTTP server exposing build info and module metadata.

## Common tasks
- Add or adjust default modules: extend `appOptions` in `options.go` and ensure new dependencies are registered before `kernel.Run`.
- Customize metadata output: update `metadata_server.go` to expose additional metadata from `appctx.Metadata`.
- Provide new module factories: expose them via `WithModuleFactory` and document required config keys.

## Testing
- Run `go test ./pkg/application` before pushing changes.
- Use `examples/application` to manually validate startup/shutdown flows.

## Required config keys
```yaml
app:
  env: dev                    # Environment name (required)
  name: myapp                 # Application name (required)
  tags:                       # Tags for resource naming
    project: myproject        # Project identifier
    family: myfamily          # Family grouping
    group: mygroup            # Group within family
```

## Related packages
- `pkg/kernel` - module lifecycle, stages, middleware
- `pkg/cfg` - configuration loading, AppId resolution
- `pkg/log` - logger injection and channel management
- `pkg/appctx` - cross-module state container

## Tips
- Keep module registration deterministic; avoid side effects in package `init`.
- Prefer `appctx` for cross-module shared state instead of singletons.
- Update root `AGENTS.md` when introducing new application-wide options that agents must know about.
