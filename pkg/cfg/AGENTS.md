# Configuration Package Agent Guide

## Scope
- Central configuration loader/merger used by every package.
- Provides `Identity` handling + naming template expansion.
- Exposes decoding helpers and merge strategies for YAML/JSON/env sources.

## Key files
- `config.go`, `read.go` - provider interfaces, load & merge process.
- `application_identifiers.go` - `Identity`, `Tags`, identity loading, validation, and `Format()` method for pattern expansion.
- `merge*.go`, `options.go` - precedence rules and customization points.
- `postprocessor.go`, `sanitizer.go` - value normalization and validation hooks.

## Common tasks
- Add config sources: extend `Option` builders in `options.go` and hook a `Provider` implementation.
- Update identity behavior: touch `application_identifiers.go`, update docs + relevant tests.
- Update naming patterns: touch `application_identifiers.go` (Format method), ensure downstream packages (sqs, sns, kinesis, kafka, redis) still pass.
- Enhance sanitization: implement `Sanitizer` and register it via `Option`.

## Testing
- `go test ./pkg/cfg` is mandatory after any change.
- For naming work, ensure downstream packages pass: `go test ./pkg/{cloud/aws/sqs,cloud/aws/sns,cloud/aws/kinesis,kafka,redis,...}`.

## Identity

`Identity` replaces the legacy `AppId` struct with a dynamic tag-based system.

### Fields
| Field | Config Key | Required | Description |
|-------|------------|----------|-------------|
| `Env` | `app.env` | No | Environment name (e.g., `dev`, `prod`) |
| `Name` | `app.name` | No | Application name |
| `Tags` | `app.tags.*` | No | Dynamic tags (project, family, group, etc.) |

### Loading identity
```go
identity, err := cfg.GetIdentity(config)
```

### Padding identity from config
```go
// Fill empty fields of Identity from config
// Useful when you have a partially populated Identity
err := identity.PadFromConfig(config)
```

## NamingTemplate via Identity.Format()

`Identity.Format()` provides placeholder validation and expansion for resource naming patterns.

### Identity placeholders
| Placeholder | Source | Description |
|-------------|--------|-------------|
| `{app.env}` | `identity.Env` | Environment name |
| `{app.name}` | `identity.Name` | Application name |
| `{app.namespace}` | `identity.Namespace` | Pre-configured namespace pattern |
| `{app.tags.<key>}` | `identity.Tags["<key>"]` | Any tag value (fully dynamic) |

Tags are fully dynamic - any `{app.tags.<key>}` placeholder is allowed, where `<key>` can be any non-empty string. Common examples include `project`, `family`, `group`, `region`, `team`, `costCenter`, etc.

### Resource-specific placeholders
Components pass additional placeholders via the `args` parameter:
- SQS: `{queueId}`
- SNS: `{topicId}`
- Kinesis: `{streamName}`
- Kafka: `{topicId}`, `{groupId}`
- Redis: `{name}`, `{key}`
- DynamoDB: `{name}` (model name)
- S3: `{bucketId}`

### Usage
```go
// Format a pattern with resource placeholder
name, err := identity.Format(pattern, delimiter, map[string]string{
    "queueId": "my-queue",
})
```

### Validation behavior
- **Unknown placeholders return error**: `unknown placeholder {foo} in pattern "..."` (typo protection)

### Pattern-driven tag requirements
Tags are only required if the naming pattern uses them:
```go
// Pattern "{app.env}-{queueId}" does NOT require any tags
// Pattern "{app.tags.project}-{app.env}-{queueId}" requires only project tag
// Pattern "{app.tags.region}-{app.tags.team}-{app.env}" requires region and team tags
```

## Related packages
- `pkg/mdl` - ModelId with its own macro system for data model naming
- `pkg/ddb` - DynamoDB table naming uses Identity.Format() with ModelId.ToMap()
- `pkg/cloud/aws/sqs` - SQS queue naming uses Identity.Format()
- `pkg/cloud/aws/sns` - SNS topic naming uses Identity.Format()
- `pkg/cloud/aws/kinesis` - Kinesis stream naming uses Identity.Format()
- `pkg/kafka` - Kafka topic/group naming uses Identity.Format()
- `pkg/redis` - Redis address/key naming uses Identity.Format()

## Tips
- Never call `Config.GetString` when you need raw template values; prefer `Get` + type conversion.
- Document new config keys in package-level README or parent AGENT so other agents can discover them.
- When adding interfaces, update `.mockery.yml` before running `go generate -run='mockery' ./...`.
- Old placeholders like `{env}`, `{project}`, `{family}`, `{group}`, `{app}` are NOT supported in Identity.Format().
