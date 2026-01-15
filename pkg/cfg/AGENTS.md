# Configuration Package Agent Guide

## Scope
- Central configuration loader/merger used by every package.
- Provides `AppIdentity` handling + naming template expansion.
- Exposes decoding helpers and merge strategies for YAML/JSON/env sources.

## Key files
- `config.go`, `read.go` - provider interfaces, load & merge process.
- `application_identifiers.go` - `AppIdentity`, `Tags`, identity loading and validation.
- `naming.go` - `NamingTemplate` for strict placeholder validation and expansion.
- `merge*.go`, `options.go` - precedence rules and customization points.
- `postprocessor.go`, `sanitizer.go` - value normalization and validation hooks.

## Common tasks
- Add config sources: extend `Option` builders in `options.go` and hook a `Provider` implementation.
- Update identity behavior: touch `application_identifiers.go`, update docs + relevant tests.
- Update naming patterns: touch `naming.go`, ensure downstream packages (sqs, sns, kinesis, kafka, redis) still pass.
- Enhance sanitization: implement `Sanitizer` and register it via `Option`.

## Testing
- `go test ./pkg/cfg` is mandatory after any change.
- For naming work, ensure downstream packages pass: `go test ./pkg/{cloud/aws/sqs,cloud/aws/sns,cloud/aws/kinesis,kafka,redis,...}`.

## AppIdentity

`AppIdentity` replaces the legacy `AppId` struct with a dynamic tag-based system.

### Fields
| Field | Config Key | Required | Description |
|-------|------------|----------|-------------|
| `Env` | `app.env` | No | Environment name (e.g., `dev`, `prod`) |
| `Name` | `app.name` | No | Application name |
| `Tags` | `app.tags.*` | No | Dynamic tags (project, family, group, etc.) |

### Loading identity
```go
identity, err := cfg.LoadAppIdentity(config)
```

### Requiring tags
```go
// Fails if any listed tag is missing or whitespace-only
err := identity.RequireTags("project", "family", "group")
```

## NamingTemplate

`NamingTemplate` provides strict placeholder validation and expansion for resource naming patterns.

### Identity placeholders
| Placeholder | Source | Description |
|-------------|--------|-------------|
| `{app.env}` | `identity.Env` | Environment name |
| `{app.name}` | `identity.Name` | Application name |
| `{app.tags.<key>}` | `identity.Tags.Get("<key>")` | Any tag value (fully dynamic) |

Tags are fully dynamic - any `{app.tags.<key>}` placeholder is allowed, where `<key>` can be any non-empty string. Common examples include `project`, `family`, `group`, `region`, `team`, `costCenter`, etc.

### Resource-specific placeholders
Components register additional placeholders for their resources:
- SQS: `{queueId}`
- SNS: `{topicId}`
- Kinesis: `{streamName}`
- Kafka: `{topicId}`, `{groupId}`
- Redis: `{name}`

### Usage
```go
// Create template with resource placeholder
tmpl := cfg.NewNamingTemplate(pattern, "queueId")

// Set resource value
tmpl.WithResourceValue("queueId", "my-queue")

// Validate and expand in one call
name, err := tmpl.ValidateAndExpand(identity)
```

### Validation behavior
- **Unknown placeholders return error**: `unknown placeholder(s) {foo} in pattern "..."` (typo protection)
- **Unclosed braces return error**: `unclosed placeholder in pattern "..."`
- **Empty placeholders return error**: `empty placeholder {} in pattern "..."`
- **Empty tag key returns error**: `{app.tags.}` is invalid
- **Missing required tags return error**: `missing required tags: family, project`

### Pattern-driven tag requirements
Tags are only required if the naming pattern uses them:
```go
// Pattern "{app.env}-{queueId}" does NOT require any tags
// Pattern "{app.tags.project}-{app.env}-{queueId}" requires only project tag
// Pattern "{app.tags.region}-{app.tags.team}-{app.env}" requires region and team tags
```

## Related packages
- `pkg/mdl` - ModelId with its own macro system for data model naming
- `pkg/ddb` - DynamoDB table naming uses ModelId (NOT NamingTemplate)
- `pkg/cloud/aws/sqs` - SQS queue naming uses NamingTemplate
- `pkg/cloud/aws/sns` - SNS topic naming uses NamingTemplate
- `pkg/cloud/aws/kinesis` - Kinesis stream naming uses NamingTemplate
- `pkg/kafka` - Kafka topic/group naming uses NamingTemplate
- `pkg/redis` - Redis address naming uses NamingTemplate

## Tips
- Never call `Config.GetString` when you need raw template values; prefer `Get` + type conversion.
- Document new config keys in package-level README or parent AGENT so other agents can discover them.
- When adding interfaces, update `.mockery.yml` before running `go generate -run='mockery' ./...`.
- Old placeholders like `{env}`, `{project}`, `{family}`, `{group}`, `{app}` are NOT supported in NamingTemplate.
