# Model Package Agent Guide

## Scope
- Shared model utilities: `ModelId`, factories, decorators used by DB, DDB, stream, and mdlsub packages.
- Central place for macro interpolation logic tied to config AppId/Realm.

## Key files
- `model.go` - `ModelId`, macros, defaults, and helper methods.
- `factory.go`, `named.go` - builder helpers for typed models.
- `transform.go` - serializer/deserializer helpers for DTOs.

## Common tasks
- Adjust macro behavior or defaults: edit `model.go`, update tests, and check dependent packages (ddb, db-repo, stream).
- Introduce helper factories: extend `factory.go` for new naming or metadata strategies.
- Update transforms to support new encoding formats.

## Testing
- `go test ./pkg/mdl`.
- When macros change, rerun `go test ./pkg/{cfg,ddb,cloud/aws}` to ensure naming remains consistent.

## ReplaceMacros method
Use `ModelId.ReplaceMacros(pattern)` to interpolate naming patterns:
```go
modelId := mdl.ModelId{Name: "users"}
modelId.PadFromConfig(config)
tableName := modelId.ReplaceMacros("{project}-{env}-{family}-{group}-{modelId}")
// Result: "myproject-dev-myfamily-mygroup-myproject.myfamily.mygroup.users"
```

Built-in macros: `{project}`, `{env}`, `{family}`, `{group}`, `{app}`, `{modelId}`

## Related packages
- `pkg/cfg` - AppId with similar macro system
- `pkg/ddb` - uses ModelId for table naming
- `pkg/db-repo` - uses ModelId for SQL table metadata

## Tips
- Keep macro names aligned with `cfg.AppId` fields; avoid config-key style placeholders.
- Document any new `ModelId` fields in root AGENT so downstream contributors know how to configure them.
