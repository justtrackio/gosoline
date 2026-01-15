# Model Package Agent Guide

## Scope
- Shared model utilities: `ModelId`, factories, decorators used by DB, DDB, stream, and mdlsub packages.
- Provides macro interpolation logic for model naming (separate from `cfg.Identity.Format()`).

## Key files
- `model_id.go` - `ModelId`, macros, defaults, and helper methods.
- `parse.go` - `ParseModelId` for parsing canonical model ID strings.
- `factory.go`, `named.go` - builder helpers for typed models.
- `transform.go` - serializer/deserializer helpers for DTOs.

## Common tasks
- Adjust macro behavior or defaults: edit `model_id.go`, update tests, and check dependent packages (ddb, db-repo, stream).
- Introduce helper factories: extend `factory.go` for new naming or metadata strategies.
- Update transforms to support new encoding formats.

## Testing
- `go test ./pkg/mdl`.
- When macros change, rerun `go test ./pkg/{ddb,db-repo}` to ensure naming remains consistent.

## Usage
To get the canonical string representation (defined by `app.model_id.domain_pattern`):
```go
modelId := mdl.ModelId{Name: "users"}
modelId.PadFromConfig(config)
canonical := modelId.String()
// Result: "myproject.production.myfamily.mygroup.users"
```

To format other strings (e.g., table names), use `config.FormatString` with `modelId.ToMap()`:
```go
name, err := config.FormatString("{app.tags.project}-{name}", modelId.ToMap())
```

**Note:** `ModelId` fields map to `Identity` style keys:

| Field | Map Key |
|-------|---------|
| `Env` | `app.env` |
| `App` | `app.name` |
| `Tags` | `app.tags.<key>` |
| `Name` | `name` |

## Related packages
- `pkg/cfg` - `Identity.Format()` for AWS resource naming
- `pkg/ddb` - uses ModelId for table naming
- `pkg/db-repo` - uses ModelId for SQL table metadata

## Tips
- DynamoDB and SQL tables use `ModelId` (via `cfg.Identity.Format()` with `ModelId.ToMap()`); AWS resources (SQS, SNS, Kinesis) use `cfg.Identity.Format()`.
- Document any new `ModelId` fields in root AGENT so downstream contributors know how to configure them.

## Canonical Model IDs
For canonical model IDs (used in message routing, etc.), the pattern works differently and is configured via `app.model_id.domain_pattern`.

- It supports standard `{app.env}`, `{app.name}`, and `{app.tags.*}` placeholders
- `{modelId}` is **NOT** used; the model name is automatically appended as the last segment (dot-separated)
- Patterns may freely mix placeholders with static text and use any delimiter between placeholders
- Example patterns:
  - `{app.tags.project}.{app.env}` -> `myProject.production.myModel`
  - `prefix-{app.env}` -> `prefix-production.myModel`
  - `{app.tags.project}-{app.env}` -> `myProject-production.myModel`
  - `ns-{app.tags.project}.{app.env}-live` -> `ns-myProject.production-live.myModel`
- Parsing uses regex-based matching: each placeholder matches non-dot characters (`[^.]+`), and the model name is everything after the final dot
