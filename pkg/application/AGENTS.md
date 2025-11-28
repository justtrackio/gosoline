# Application Package Agent Guide

## Scope
- Bootstraps gosoline applications via `application.Run`.
- Wires modules, runners, and metadata server exposure.
- Bridges configuration (`cfg`) and lifecycle management (`kernel`).

## Key files
- `app.go` - core application struct and global Run entrypoint.
- `options.go` - functional options for adding modules, health checks, and shared components.
- `metadata_server.go` - HTTP server exposing build info and module metadata.
- `runners.go` - helpers for wiring background runners/modules.

## Common tasks
- Add or adjust default modules: extend `appOptions` in `options.go` and ensure new dependencies are registered before `kernel.Run`.
- Customize metadata output: update `metadata_server.go` and extend the struct returned by `buildMetadata`.
- Provide new module factories: expose them via `WithModuleFactory` and document required config keys.

## Testing
- Run `go test ./pkg/application` before pushing changes.
- Use `examples/application` to manually validate startup/shutdown flows.

## Required config keys
```yaml
env: dev                    # Environment name
app_project: myproject      # Project identifier
app_family: myfamily        # Family grouping
app_group: mygroup          # Group within family
app_name: myapp             # Application name
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
