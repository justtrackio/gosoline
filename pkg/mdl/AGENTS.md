# Model Package Agent Guide

## Scope
- Shared model utilities: `ModelId`, factories, decorators used by DB, DDB, stream, and mdlsub packages.
- Provides macro interpolation logic for model naming (separate from `cfg.NamingTemplate`).

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
- When macros change, rerun `go test ./pkg/{ddb,db-repo}` to ensure naming remains consistent.

## ReplaceMacros method
Use `ModelId.ReplaceMacros(pattern)` to interpolate naming patterns:
```go
modelId := mdl.ModelId{Name: "users"}
modelId.PadFromConfig(config)
tableName := modelId.ReplaceMacros("{project}-{env}-{family}-{group}-{modelId}")
// Result: "myproject-dev-myfamily-mygroup-myproject.myfamily.mygroup.users"
```

**Note:** `ModelId` macros are different from `cfg.NamingTemplate` macros:

| ModelId Macro | Description |
|---------------|-------------|
| `{project}` | Project from ModelId |
| `{env}` | Environment from ModelId |
| `{family}` | Family from ModelId |
| `{group}` | Group from ModelId |
| `{app}` | App from ModelId |
| `{modelId}` | Model's string representation |

## Related packages
- `pkg/cfg` - `NamingTemplate` with AppIdentity macros (for AWS resource naming)
- `pkg/ddb` - uses ModelId for table naming
- `pkg/db-repo` - uses ModelId for SQL table metadata

## Tips
- `ModelId` macros (`{project}`, `{family}`) differ from `cfg.NamingTemplate` macros (`{app.tags.project}`, `{app.tags.family}`).
- DynamoDB and SQL tables use `ModelId`; AWS resources (SQS, SNS, Kinesis) use `cfg.NamingTemplate`.
- Document any new `ModelId` fields in root AGENT so downstream contributors know how to configure them.
