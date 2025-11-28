# Configuration Package Agent Guide

## Scope
- Central configuration loader/merger used by every package.
- Provides AppId/Realm handling + macro interpolation.
- Exposes decoding helpers and merge strategies for YAML/JSON/env sources.

## Key files
- `config.go`, `read.go` - provider interfaces, load & merge process.
- `application_identifiers.go` - `AppId`, realm macros, `ReplaceMacros` utilities.
- `merge*.go`, `options.go` - precedence rules and customization points.
- `postprocessor.go`, `sanitizer.go` - value normalization and validation hooks.

## Common tasks
- Add config sources: extend `Option` builders in `options.go` and hook a `Provider` implementation.
- Update macro behavior: touch `application_identifiers.go`, update docs + relevant tests.
- Enhance sanitization: implement `Sanitizer` and register it via `Option`.

## Testing
- `go test ./pkg/cfg` is mandatory after any change.
- For macro work, ensure downstream packages (cfg, mdl, ddb, cloud/aws) still pass: `go test ./pkg/{cfg,mdl,cloud/aws,...}`.

## Available macros in ReplaceMacros
| Macro | Source | Description |
|-------|--------|-------------|
| `{project}` | `app_project` | Project identifier |
| `{env}` | `env` | Environment name |
| `{family}` | `app_family` | Family grouping |
| `{group}` | `app_group` | Group within family |
| `{app}` | `app_name` | Application name |

## Related packages
- `pkg/mdl` - ModelId with similar macro system for data models
- `pkg/cloud/aws/*` - AWS services consume AppId for naming
- `pkg/ddb` - DynamoDB naming uses ModelId which extends AppId concepts

## Tips
- Never call `Config.GetString` when you need raw template values; prefer `Get` + type conversion.
- Document new config keys in package-level README or parent AGENT so other agents can discover them.
- When adding interfaces, update `.mockery.yml` before running `go generate -run='mockery' ./...`.
